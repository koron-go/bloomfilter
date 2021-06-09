package bloomfilter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/dgryski/go-metro"
	"github.com/go-redis/redis/v8"
)

const redisWatchRetryMax = 5

func watchWithRetry(ctx context.Context, uc redis.UniversalClient, fn func(tx *redis.Tx) error, keys ...string) error {
	for retries := redisWatchRetryMax; retries > 0; retries-- {
		err := uc.Watch(ctx, fn, keys...)
		if err != nil && errors.Is(err, redis.TxFailedErr) {
			continue
		}
		return err
	}
	return fmt.Errorf("transaction failed %d times: %w", 5, redis.TxFailedErr)
}

// VBF3Redis provides VBF3 with Redis backend.
type VBF3Redis struct {
	c       redis.UniversalClient
	keyData string
	keyGen  string
	m       int
	k       int
}

// VBF3Gen codes generation parameters of VBF3.
type VBF3Gen struct {
	Bottom uint8 `json:"bottom"`
	Top    uint8 `json:"top"`
	Max    uint8 `json:"max"`
}

func (g *VBF3Gen) isValid(n uint8) bool {
	if n == 0 {
		return false
	}
	a := g.Bottom <= n
	b := g.Top >= n
	if g.Bottom <= g.Top {
		return a && b
	}
	return a || b
}

func NewVBF3Redis(uc redis.UniversalClient, name string, m, k int) *VBF3Redis {
	return &VBF3Redis{
		c:       uc,
		keyData: name,
		keyGen:  name + "_gen",
		m:       m,
		k:       k,
	}
}

func (rf *VBF3Redis) getGen(ctx context.Context, c redis.Cmdable) (*VBF3Gen, error) {
	b, err := c.Get(ctx, rf.keyGen).Bytes()
	if err != nil {
		return nil, fmt.Errorf("failed to get generation info with key %q: %w", rf.keyGen, err)
	}
	var v *VBF3Gen
	err = json.Unmarshal(b, &v)
	if err != nil {
		return nil, fmt.Errorf("invalid format of generation info: %w", err)
	}
	return v, nil
}

func (rf *VBF3Redis) getData(ctx context.Context, c redis.Cmdable, x int) (uint8, error) {
	r, err := c.BitField(ctx, rf.keyData, "GET", "u8", x*8).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get data at #%d: %w", x, err)
	}
	if len(r) != 1 {
		return 0, fmt.Errorf("unexpected length of response for get data %+v", r)
	}
	return uint8(r[0]), nil
}

func (rf *VBF3Redis) setData(ctx context.Context, c redis.Cmdable, x int, v uint8) error {
	_, err := c.BitField(ctx, rf.keyData, "SET", "u8", x*8, v).Result()
	if err != nil {
		return fmt.Errorf("failed to set data at #%d: %w", x, err)
	}
	return nil
}

func (rf *VBF3Redis) hash(d []byte, n int) int {
	return int(metro.Hash64(d, uint64(n)) % uint64(rf.m))
}

func (rf *VBF3Redis) Put(ctx context.Context, d []byte, life uint8) error {
	return rf.put2(ctx, d, life)
}

func (rf *VBF3Redis) put1(ctx context.Context, d []byte, life uint8) error {
	gen, err := rf.getGen(ctx, rf.c)
	if err != nil {
		return err
	}
	if life > gen.Max {
		return fmt.Errorf("life should be less than (<=) %d", gen.Max)
	}
	nv := m255p1add(gen.Bottom, life-1)
	for i := 0; i < rf.k; i++ {
		x := rf.hash(d, i)
		v, err := rf.getData(ctx, rf.c, x)
		if err != nil {
			return err
		}
		var curr uint8
		if gen.isValid(v) {
			curr = v - gen.Bottom + 1
			if v < gen.Bottom {
				curr--
			}
		}
		if curr == 0 || life > curr {
			err := rf.setData(ctx, rf.c, x, nv)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (rf *VBF3Redis) put2(ctx context.Context, d []byte, life uint8) error {
	gen, err := rf.getGen(ctx, rf.c)
	if err != nil {
		return err
	}
	if life > gen.Max {
		return fmt.Errorf("life should be less than (<=) %d", gen.Max)
	}

	xx := make([]int, rf.k)
	readArgs := make([]interface{}, 0, rf.k*3)
	for i := 0; i < rf.k; i++ {
		x := rf.hash(d, i)
		xx[i] = x
		readArgs = append(readArgs, "GET", "u8", x*8)
	}
	r, err := rf.c.BitField(ctx, rf.keyData, readArgs...).Result()
	if err != nil {
		return fmt.Errorf("failed to get data for put (args=%+v): %w", readArgs, err)
	}

	nv := m255p1add(gen.Bottom, life-1)
	writeArgs := make([]interface{}, 0, rf.k*3)
	for i := 0; i < rf.k; i++ {
		v := uint8(r[i])
		var curr uint8
		if gen.isValid(v) {
			curr = v - gen.Bottom + 1
			if v < gen.Bottom {
				curr--
			}
		}
		if curr == 0 || life > curr {
			writeArgs = append(writeArgs, "SET", "u8", xx[i]*8, nv)
		}
	}
	if len(writeArgs) > 0 {
		_, err := rf.c.BitField(ctx, rf.keyData, writeArgs...).Result()
		if err != nil {
			return fmt.Errorf("failed to set data for put: %w", err)
		}
	}
	return nil
}

type vbf3pair struct {
	x int
	v uint8
}

func (rf *VBF3Redis) Check(ctx context.Context, d []byte) (bool, error) {
	gen, err := rf.getGen(ctx, rf.c)
	if err != nil {
		return false, err
	}
	xx := make([]int, rf.k)
	args := make([]interface{}, 0, 3*rf.k)
	for i := 0; i < rf.k; i++ {
		x := rf.hash(d, i)
		args = append(args, "GET", "u8", x*8)
		xx[i] = x
	}
	vv, err := rf.c.BitField(ctx, rf.keyData, args...).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check/get data: %w", err)
	}
	invalids := make([]vbf3pair, 0, len(vv))
	retval := true
	for i, v := range vv {
		v8 := uint8(v)
		if !gen.isValid(v8) {
			retval = false
			invalids = append(invalids, vbf3pair{x: xx[i], v: v8})
		}
	}
	if len(invalids) >= 0 {
		args := make([]interface{}, 0, 3*len(invalids))
		for _, d := range invalids {
			args = append(args, "SET", "u8", d.x*8, 0)
		}
		_, err := rf.c.BitField(ctx, rf.keyData, args...).Result()
		if err != nil {
			return retval, fmt.Errorf("failed to clear invalids: %w", err)
		}
	}
	return retval, nil
}

func (rf *VBF3Redis) AdvanceGeneration(ctx context.Context, generations uint8) error {
	return watchWithRetry(ctx, rf.c, func(tx *redis.Tx) error {
		gen, err := rf.getGen(tx.Context(), tx)
		if err != nil {
			return err
		}
		gen.Bottom = m255p1add(gen.Bottom, generations)
		gen.Top = m255p1add(gen.Top, generations)
		next, err := json.Marshal(gen)
		if err != nil {
			return err
		}
		_, err = tx.Pipelined(tx.Context(), func(pipe redis.Pipeliner) error {
			pipe.Set(tx.Context(), rf.keyGen, next, 0)
			return nil
		})
		return err
	}, rf.keyGen)
}

func (rf *VBF3Redis) Sweep(ctx context.Context) error {
	defer func(st time.Time) {
		log.Printf("Sweep took %s", time.Since(st))
	}(time.Now())
	gen, err := rf.getGen(ctx, rf.c)
	if err != nil {
		return err
	}
	return watchWithRetry(ctx, rf.c, func(tx *redis.Tx) error {
		b, err := tx.Get(tx.Context(), rf.keyData).Bytes()
		if err != nil {
			return err
		}
		var modified bool
		for i, d := range b {
			if d != 0 && !gen.isValid(d) {
				b[i] = 0
				modified = true
			}
		}
		if !modified {
			return nil
		}
		_, err = tx.Pipelined(tx.Context(), func(pipe redis.Pipeliner) error {
			pipe.Set(tx.Context(), rf.keyData, b, 0)
			return nil
		})
		return err
	}, rf.keyData)
}

func m255p1add(a, b uint8) uint8 {
	v := uint16(a) + uint16(b)
	if v <= 255 {
		return uint8(v)
	}
	return uint8(v - 255)
}

func (rf *VBF3Redis) Delete(ctx context.Context) error {
	_, err := rf.c.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Del(ctx, rf.keyData)
		pipe.Del(ctx, rf.keyGen)
		return nil
	})
	return err
}

func (rf *VBF3Redis) Prepare(ctx context.Context, maxLife uint8) error {
	gen, err := rf.getGen(ctx, rf.c)
	if err == nil {
		// FIXME: maxLife
		return nil
	}
	// FIXME: check error contents
	gen = &VBF3Gen{
		Bottom: 1,
		Top:    maxLife,
		Max:    maxLife,
	}
	next, err := json.Marshal(gen)
	if err != nil {
		return err
	}
	_, err = rf.c.Set(ctx, rf.keyGen, next, 0).Result()
	return err
}

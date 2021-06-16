package vbf3redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"

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

type keyBase string

func (kb keyBase) data(n int) string {
	return string(kb) + "_" + strconv.Itoa(n)
}

func (kb keyBase) props() string {
	return string(kb) + "_props"
}

func (kb keyBase) gen() string {
	return string(kb) + "_gen"
}

// VBF3Redis provides VBF3 with Redis backend.
type VBF3Redis struct {
	key keyBase

	vbf3props

	c redis.UniversalClient

	pageNum int
}

// vbf3props codes constant properties of VBF3.
type vbf3props struct {
	M uint64 `json:"m"`
	K uint   `json:"k"`

	MaxLife uint8 `json:"max_life"`

	SeedBase uint64 `json:"seed_base"` // for future use
}

func propsGet(ctx context.Context, c redis.Cmdable, key keyBase) (*vbf3props, bool, error) {
	b, err := c.Get(ctx, key.props()).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("failed to get properties info with key %q: %w", key.props(), err)
	}
	var v *vbf3props
	err = json.Unmarshal(b, &v)
	if err != nil {
		return nil, false, fmt.Errorf("invalid format of generation info: %w", err)
	}
	return v, true, nil
}

func propsPut(ctx context.Context, c redis.Cmdable, key keyBase, p *vbf3props) error {
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	_, err = c.Set(ctx, key.props(), b, 0).Result()
	if err != nil {
		return fmt.Errorf("failed to put properties with key %q: %w", key.props(), err)
	}
	return nil
}

// vbf3gen codes generation parameters of VBF3.
type vbf3gen struct {
	Bottom uint8 `json:"bottom"`
	Top    uint8 `json:"top"`
}

func getGen(ctx context.Context, c redis.Cmdable, key keyBase) (*vbf3gen, error) {
	b, err := c.Get(ctx, key.gen()).Bytes()
	if err != nil {
		return nil, fmt.Errorf("failed to get generation info with key %q: %w", key.gen(), err)
	}
	var v *vbf3gen
	err = json.Unmarshal(b, &v)
	if err != nil {
		return nil, fmt.Errorf("invalid format of generation info: %w", err)
	}
	return v, nil
}

func putGen(ctx context.Context, c redis.Cmdable, key keyBase, g *vbf3gen) error {
	b, err := json.Marshal(g)
	if err != nil {
		return err
	}
	_, err = c.Set(ctx, key.gen(), b, 0).Result()
	if err != nil {
		return fmt.Errorf("failed to put generation info with key %q: %w", key.gen(), err)
	}
	return nil
}

func (g *vbf3gen) isValid(n uint8) bool {
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

const pageSize = 512 * 1024 * 1024

// Open open a VBF3Redis instance when exists, otherwise create it.
// When parameters are not match with existing one, this will fail.
func Open(ctx context.Context, uc redis.UniversalClient, name string, m uint64, k uint, maxLife uint8) (*VBF3Redis, error) {
	var key = keyBase(name)
	// FIXME: introduce transaction.
	p, ok, err := propsGet(ctx, uc, key)
	if err != nil {
		return nil, err
	}
	props := vbf3props{
		M:       m,
		K:       k,
		MaxLife: maxLife,
	}
	if ok && props != *p {
		return nil, fmt.Errorf("mismatch parameter: want=%+v got=%+v", props, *p)
	}
	if !ok {
		// create/setup a new VBF3 instance on Redis.
		err := propsPut(ctx, uc, key, &props)
		if err != nil {
			return nil, err
		}
		err = putGen(ctx, uc, key, &vbf3gen{
			Bottom: 1,
			Top:    maxLife,
		})
		if err != nil {
			return nil, err
		}
	}
	return &VBF3Redis{
		key:       key,
		vbf3props: props,
		c:         uc,
		pageNum:   int(m/pageSize + 1),
	}, nil
}

func (rf *VBF3Redis) hash(d []byte, n uint) int {
	return int(metro.Hash64(d, uint64(n)) % rf.M)
}

type pos struct {
	page  uint64
	index uint64
}

func (a pos) less(b pos) bool {
	return a.page < b.page || (a.page == b.page && a.index < b.index)
}

func (rf *VBF3Redis) hashArray(d []byte) []pos {
	xx := make([]pos, rf.K)
	for i := uint(0); i < rf.K; i++ {
		x := metro.Hash64(d, uint64(i)) % rf.M
		xx[i] = pos{
			page:  x / pageSize,
			index: x % pageSize,
		}
	}
	sort.Slice(xx, func(i, j int) bool {
		return xx[i].less(xx[j])
	})
	return xx
}

func (rf *VBF3Redis) Put(ctx context.Context, d []byte, life uint8) error {
	if life > rf.MaxLife {
		return fmt.Errorf("life should be less than (<=) %d", rf.MaxLife)
	}
	gen, err := getGen(ctx, rf.c, rf.key)
	if err != nil {
		return err
	}

	// TODO: support pagings

	xx := make([]int, rf.K)
	readArgs := make([]interface{}, 0, rf.K*3)
	for i := uint(0); i < rf.K; i++ {
		x := rf.hash(d, i)
		xx[i] = x
		readArgs = append(readArgs, "GET", "u8", x*8)
	}
	r, err := rf.c.BitField(ctx, rf.key.data(0), readArgs...).Result()
	if err != nil {
		return fmt.Errorf("failed to get data for put (args=%+v): %w", readArgs, err)
	}

	nv := m255p1add(gen.Bottom, life-1)
	writeArgs := make([]interface{}, 0, rf.K*3)
	for i := uint(0); i < rf.K; i++ {
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
		_, err := rf.c.BitField(ctx, rf.key.data(0), writeArgs...).Result()
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
	gen, err := getGen(ctx, rf.c, rf.key)
	if err != nil {
		return false, err
	}
	// TODO: support pagings
	xx := make([]int, rf.K)
	args := make([]interface{}, 0, 3*rf.K)
	for i := uint(0); i < rf.K; i++ {
		x := rf.hash(d, i)
		args = append(args, "GET", "u8", x*8)
		xx[i] = x
	}
	vv, err := rf.c.BitField(ctx, rf.key.data(0), args...).Result()
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
		_, err := rf.c.BitField(ctx, rf.key.data(0), args...).Result()
		if err != nil {
			return retval, fmt.Errorf("failed to clear invalids: %w", err)
		}
	}
	return retval, nil
}

func (rf *VBF3Redis) AdvanceGeneration(ctx context.Context, generations uint8) error {
	return watchWithRetry(ctx, rf.c, func(tx *redis.Tx) error {
		gen, err := getGen(tx.Context(), tx, rf.key)
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
			pipe.Set(tx.Context(), rf.key.gen(), next, 0)
			return nil
		})
		return err
	}, rf.key.gen())
}

func (rf *VBF3Redis) Sweep(ctx context.Context) error {
	//defer func(st time.Time) {
	//	log.Printf("Sweep took %s", time.Since(st))
	//}(time.Now())
	gen, err := getGen(ctx, rf.c, rf.key)
	if err != nil {
		return err
	}
	for pn := 0; pn < rf.pageNum; pn++ {
		keyData := rf.key.data(pn)
		err := watchWithRetry(ctx, rf.c, func(tx *redis.Tx) error {
			b, err := tx.Get(tx.Context(), keyData).Bytes()
			if err != nil {
				if errors.Is(err, redis.Nil) {
					return nil
				}
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
				pipe.Set(tx.Context(), keyData, b, 0)
				return nil
			})
			return err
		}, keyData)
		if err != nil {
			return fmt.Errorf("failed to sweep key:%q: %w", keyData, err)
		}
	}
	return nil
}

func m255p1add(a, b uint8) uint8 {
	v := uint16(a) + uint16(b)
	if v <= 255 {
		return uint8(v)
	}
	return uint8(v - 255)
}

func (rf *VBF3Redis) Drop(ctx context.Context) error {
	_, err := rf.c.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for i := 0; i < rf.pageNum; i++ {
			pipe.Del(ctx, rf.key.data(i))
		}
		pipe.Del(ctx, rf.key.gen())
		pipe.Del(ctx, rf.key.props())
		return nil
	})
	return err
}

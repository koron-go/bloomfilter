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

func (rf *VBF3Redis) hashArray(dd ...[]byte) []pos {
	pp := make([]pos, rf.K)
	seen := map[uint64]struct{}{}
	for _, d := range dd {
		for i := uint(0); i < rf.K; i++ {
			x := metro.Hash64(d, uint64(i)) % rf.M
			if _, ok := seen[x]; ok {
				continue
			}
			seen[x] = struct{}{}
			pp[i] = pos{
				page:  x / pageSize,
				index: (x % pageSize) * 8,
			}
		}
	}
	sort.Slice(pp, func(i, j int) bool {
		return pp[i].less(pp[j])
	})
	return pp
}

// getValues gets current values by hashed `d` keys.
func (rf *VBF3Redis) getValues(ctx context.Context, pp []pos) ([]*redis.IntSliceCmd, error) {
	pages := make([]int, rf.pageNum)
	args := make([]interface{}, 0, len(pp)*3)
	for _, p := range pp {
		pages[p.page]++
		args = append(args, "GET", "u8", p.index)
	}

	cmds := make([]*redis.IntSliceCmd, 0, rf.pageNum)
	_, err := rf.c.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		x := 0
		for i, n := range pages {
			if n == 0 {
				continue
			}
			cmds = append(cmds, pipe.BitField(ctx, rf.key.data(i), args[x:x+n*3]...))
			x += n * 3
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to check/get data: %w", err)
	}
	return cmds, nil
}

func (rf *VBF3Redis) setValues(ctx context.Context, pp []pos, pages []int, value uint8) error {
	_, err := rf.c.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		base := 0
		for i, n := range pages {
			if n == 0 {
				continue
			}
			args := make([]interface{}, 0, n*4)
			for j := 0; j < n; j++ {
				args = append(args, "SET", "u8", pp[base+j].index, value)
			}
			base += n
			pipe.BitField(ctx, rf.key.data(i), args...)
		}
		return nil
	})
	return err
}

// Put puts a value with life.
func (rf *VBF3Redis) Put(ctx context.Context, d []byte, life uint8) error {
	if life > rf.MaxLife {
		return fmt.Errorf("life should be less than (<=) %d", rf.MaxLife)
	}
	gen, err := getGen(ctx, rf.c, rf.key)
	if err != nil {
		return err
	}
	pp := rf.hashArray(d)
	return rf.put(ctx, gen, pp, life)
}

// PutAll puts all values with life
func (rf *VBF3Redis) PutAll(ctx context.Context, life uint8, dd [][]byte) error {
	// preparation
	if life > rf.MaxLife {
		return fmt.Errorf("life should be less than (<=) %d", rf.MaxLife)
	}
	if len(dd) == 0 {
		return nil
	}
	gen, err := getGen(ctx, rf.c, rf.key)
	if err != nil {
		return err
	}
	pp := rf.hashArray(dd...)
	return rf.put(ctx, gen, pp, life)
}

// put updates postions.
func (rf *VBF3Redis) put(ctx context.Context, gen *vbf3gen, pp []pos, life uint8) error {
	cmds, err := rf.getValues(ctx, pp)
	if err != nil {
		return err
	}
	// detect updates
	updatePages := make([]int, rf.pageNum)
	updates := make([]pos, 0, len(pp))
	updateIndex := 0
	for _, cmd := range cmds {
		if cmd == nil {
			continue
		}
		for _, v := range cmd.Val() {
			v8 := uint8(v)
			var curr uint8
			if gen.isValid(v8) {
				curr = v8 - gen.Bottom + 1
				if v8 < gen.Bottom {
					curr--
				}
			}
			if curr == 0 || life > curr {
				updatePages[pp[updateIndex].page]++
				updates = append(updates, pp[updateIndex])
			}
			updateIndex++
		}
	}
	if len(updates) == 0 {
		return nil
	}
	// apply updates
	nv := m255p1add(gen.Bottom, life-1)
	err = rf.setValues(ctx, updates, updatePages, nv)
	if err != nil {
		return fmt.Errorf("failed to set data for put: %w", err)
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
	pp := rf.hashArray(d)
	cmds, err := rf.getValues(ctx, pp)
	if err != nil {
		return false, err
	}

	// detect invalids
	validAll := true
	invalidPages := make([]int, rf.pageNum)
	invalids := make([]pos, 0, len(pp))
	invalidIndex := 0
	for _, cmd := range cmds {
		if cmd == nil {
			continue
		}
		for _, v := range cmd.Val() {
			v8 := uint8(v)
			if !gen.isValid(v8) {
				validAll = false
				if v8 != 0 {
					invalidPages[pp[invalidIndex].page]++
					invalids = append(invalids, pp[invalidIndex])
				}
			}
			invalidIndex++
		}
	}
	if validAll {
		return true, nil
	}
	if len(invalids) == 0 {
		return false, nil
	}

	// clear invalids
	err = rf.setValues(ctx, invalids, invalidPages, 0)
	if err != nil {
		return false, fmt.Errorf("failed to clear invalids: %w", err)
	}
	return false, nil
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

func Drop(ctx context.Context, c redis.UniversalClient, name string) error {
	keys, err := c.Keys(ctx, name+"_").Result()
	if err != nil {
		return fmt.Errorf("faild to list up keys: %w", err)
	}
	if len(keys) == 0 {
		return nil
	}
	_, err = c.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for _, k := range keys {
			pipe.Del(ctx, k)
		}
		return nil
	})
	return err
}

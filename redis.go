package bloomfilter

import (
	"context"

	"github.com/dgryski/go-metro"
	"github.com/go-redis/redis/v8"
)

// Redis provides volatile bloom filter which use redis as store.
type Redis struct {
	c redis.UniversalClient
	n string
	m int
	k int
}

//const redisNbits = 8
const redisMax = 255

// NewRedis creates a new Redis bloom filter.
func NewRedis(uc redis.UniversalClient, name string, m, k int) *Redis {
	return &Redis{
		c: uc,
		n: name,
		m: m,
		k: k,
	}
}

func (rf *Redis) Put(ctx context.Context, d []byte) error {
	// using "BITFIELD OVERFLOW SAT INCRBY ...{max value (255)}...", mark hash
	// bits be available.
	args := make([]interface{}, 0, 2+4*rf.k)
	args = append(args, "OVERFLOW", "SAT")
	for i := 0; i < rf.k; i++ {
		x := int64(metro.Hash64(d, uint64(i)) % uint64(rf.m))
		args = append(args, "INCRBY", "u8", x*8, redisMax)
	}
	_, err := rf.c.BitField(ctx, rf.n, args...).Result()
	if err != nil {
		return err
	}
	return nil
}

func (rf *Redis) Check(ctx context.Context, d []byte, bias uint8) (bool, error) {
	// using "BITFIELD GET ... GET ..." obtain all values by a command
	args := make([]interface{}, 0, 3*rf.k)
	for i := 0; i < rf.k; i++ {
		x := int64(metro.Hash64(d, uint64(i)) % uint64(rf.m))
		args = append(args, "GET", "u8", x*8)
	}
	r, err := rf.c.BitField(ctx, rf.n, args...).Result()
	if err != nil {
		return false, err
	}
	for _, v := range r {
		if v <= int64(bias) {
			return false, nil
		}
	}
	return true, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (rf *Redis) Subtract(ctx context.Context, delta uint8) error {
	const bulk = 256
	val := -int(delta)
	args := make([]interface{}, 0, 2+4*bulk)
	args = append(args, "OVERFLOW", "SAT")
	for i := 0; i < rf.m; i += bulk {
		// check if context canceled.
		if err := ctx.Err(); err != nil {
			return err
		}
		start, end := i, min(i+bulk, rf.m)
		args = args[:2]
		for j := start; j < end; j++ {
			args = append(args, "INCRBY", "u8", j*8, val)
		}
		_, err := rf.c.BitField(ctx, rf.n, args...).Result()
		if err != nil {
			return err
		}
	}
	return nil
}

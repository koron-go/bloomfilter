package vbf3redis

import (
	"context"
	"fmt"
	"math"
	"testing"
)

func Test20211118a(t *testing.T) {
	t.Run("10K", func(t *testing.T) { run20211118a(t, 10000, 0.001) })
	t.Run("100K", func(t *testing.T) { run20211118a(t, 100000, 0.001) })
	t.Run("1M", func(t *testing.T) { run20211118a(t, 1000000, 0.001) })
	t.Run("10M", func(t *testing.T) { run20211118a(t, 10000000, 0.001) })
	t.Run("20M", func(t *testing.T) { run20211118a(t, 20000000, 0.001) })
	t.Run("30M", func(t *testing.T) { run20211118a(t, 30000000, 0.001) })
	t.Run("40M", func(t *testing.T) { run20211118a(t, 40000000, 0.001) })
	t.Run("50M", func(t *testing.T) { run20211118a(t, 50000000, 0.001) })
	t.Run("60M", func(t *testing.T) { run20211118a(t, 60000000, 0.001) })
	t.Run("70M", func(t *testing.T) { run20211118a(t, 70000000, 0.001) })
	t.Run("80M", func(t *testing.T) { run20211118a(t, 80000000, 0.001) })
	t.Run("100M", func(t *testing.T) { run20211118a(t, 100000000, 0.001) })
}

func np2mk(n, p float64) (m uint64, k uint) {
	m64 := math.Ceil(n * math.Log(p) / math.Log(1/math.Pow(2, math.Log(2))))
	k64 := math.Round((m64 / n) * math.Log(2))
	return uint64(m64), uint(k64)
}

func run20211118a(t *testing.T, n float64, p float64) {
	const q = 100

	m, k := np2mk(n, p)
	t.Logf("n=%g p=%g m=%d k=%d", n, p, m, k)

	ctx := context.Background()
	c := newTestRedisClient(t)
	rf, err := Open(ctx, c, t.Name(), m, k, 10)
	if err != nil {
		t.Fatalf("failed to create: %s", err)
	}
	t.Cleanup(func() {
		rf.Drop(ctx)
	})

	vals := make([][]byte, 0, q)
	for i := 0; i < q; i++ {
		vals = append(vals, []byte(fmt.Sprintf("value%04d", i)))
	}
	err = rf.PutAll(ctx, 1, vals)
	if err != nil {
		t.Fatalf("put failed: %s", err)
	}
	vals = nil

	for i := 0; i < q; i++ {
		has, err := rf.Check(ctx, []byte(fmt.Sprintf("value%04d", i)))
		if err != nil {
			t.Fatalf("check failed: m=%d k=%d i=%d: %s", m, k, i, err)
		}
		if !has {
			t.Fatalf("check return false unexpectedly: m=%d k=%d i=%d", m, k, i)
		}
	}
}

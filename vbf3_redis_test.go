package bloomfilter

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"testing"
)

func checkVBF3Redis(t *testing.T, m, k, num int, f float64) {
	t.Run(fmt.Sprintf("m=%d k=%d num=%d f=%f", m, k, num, f), func(t *testing.T) {
		ctx := context.Background()

		mid := int(float64(num) * f)
		c := newTestRedisClient(t)
		rf := NewVBF3Redis(c, t.Name(), m, k)
		err := rf.Prepare(ctx, 10)
		if err != nil {
			t.Fatalf("failed to prepare: %s", err)
		}
		t.Cleanup(func() {
			rf.Delete(ctx)
		})
		for i := 0; i < mid; i++ {
			err := rf.Put(ctx, []byte(strconv.Itoa(i)), 1)
			if err != nil {
				t.Fatalf("put failed: %s", err)
			}
		}

		falsePositive := 0
		for i := mid; i < num; i++ {
			has, err := rf.Check(ctx, []byte(strconv.Itoa(i)))
			if err != nil {
				t.Fatalf("check failed: %s", err)
			}
			if has {
				falsePositive++
			}
		}
		errRate := float64(falsePositive) / float64(num) * 100
		if errRate > 1 {
			t.Errorf("too big error rate: %.2f%% false_positive=%d m=%d k=%d n=%d f=%f", errRate, falsePositive, m, k, num, f)
		}
	})
}

func TestVBF3RedisBasic(t *testing.T) {
	checkVBF3Redis(t, 1000, 7, 200, 0.1)
	checkVBF3Redis(t, 1000, 7, 200, 0.5)
	checkVBF3Redis(t, 1000, 7, 200, 0.9)
	checkVBF3Redis(t, 1000, 7, 400, 0.1)
	checkVBF3Redis(t, 1000, 7, 700, 0.1)
	checkVBF3Redis(t, 1000, 7, 1000, 0.1)
}

func BenchmarkVBF3RedisPut(b *testing.B) {
	c := newTestRedisClient(b)
	rf := NewVBF3Redis(c, b.Name(), b.N*10, 7)
	ctx := context.Background()
	err := rf.Prepare(ctx, 10)
	if err != nil {
		b.Fatalf("failed to prepare: %s", err)
	}
	b.Cleanup(func() {
		rf.Delete(ctx)
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := rf.Put(ctx, []byte(strconv.Itoa(i)), 1)
		if err != nil {
			b.Fatalf("put failed: %s", err)
		}
	}
}

func BenchmarkVBF3RedisCheck(b *testing.B) {
	c := newTestRedisClient(b)
	rf := NewVBF3Redis(c, b.Name(), b.N*10, 7)
	ctx := context.Background()
	err := rf.Prepare(ctx, 10)
	if err != nil {
		b.Fatalf("failed to prepare: %s", err)
	}
	b.Cleanup(func() {
		rf.Delete(ctx)
	})

	for i := 0; i < b.N; i++ {
		rf.Put(ctx, []byte(strconv.Itoa(i)), 1)
	}
	max := b.N * 10
	fail := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x := int(rand.Int31n(int32(max)))
		want := x < b.N
		got, err := rf.Check(ctx, []byte(strconv.Itoa(x)))
		if err != nil {
			b.Fatalf("runtime error: %s", err)
		}
		if got != want {
			if !got {
				b.Fatalf("boom!: N=%d x=%d", b.N, x)
			}
			fail++
		}
	}
	rate := float64(fail) / float64(b.N) * 100
	if rate > 1 && b.N > 100 {
		b.Logf("too big error rate: %.2f%% failure=%d total=%d", rate, fail, b.N)
	}
}

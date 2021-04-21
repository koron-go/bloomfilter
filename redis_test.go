package bloomfilter

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"testing"

	"github.com/go-redis/redis/v8"
)

func newTestRedisClient(tb testing.TB) *redis.Client {
	u, ok := os.LookupEnv("REDIS_URL")
	if !ok {
		tb.Skip("please set REDIS_URL to enable this test/benchmark")
	}
	opts, err := redis.ParseURL(u)
	if err != nil {
		tb.Helper()
		tb.Fatal(err)
	}
	c := redis.NewClient(opts)
	tb.Cleanup(func() {
		_, err := c.Del(context.Background(), tb.Name()).Result()
		if err != nil {
			tb.Helper()
			tb.Errorf("failed to cleanup: %s", err)
		}
	})
	return c
}

func BenchmarkRedisPut(b *testing.B) {
	c := newTestRedisClient(b)
	rf := NewRedis(c, b.Name(), b.N*10, 7)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := rf.Put(ctx, []byte(strconv.Itoa(i)))
		if err != nil {
			b.Fatalf("put failed: %s", err)
		}
	}
}

func BenchmarkRedisCheck(b *testing.B) {
	c := newTestRedisClient(b)
	rf := NewRedis(c, b.Name(), b.N*10, 7)
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		rf.Put(ctx, []byte(strconv.Itoa(i)))
	}
	max := b.N * 10
	fail := 0
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x := int(rand.Int31n(int32(max)))
		want := x < b.N
		got, err := rf.Check(ctx, []byte(strconv.Itoa(x)), 0)
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

func BenchmarkRedisSubtract(b *testing.B) {
	c := newTestRedisClient(b)
	rf := NewRedis(c, b.Name(), b.N*10, 7)
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		rf.Put(ctx, []byte(strconv.Itoa(i)))
	}
	b.ResetTimer()
	err := rf.Subtract(ctx, 128)
	if err != nil {
		b.Fatalf("runtime error: %s", err)
	}
}

func checkRedis(t *testing.T, m, k, num int, f float64) {
	t.Run(fmt.Sprintf("m=%d k=%d num=%d f=%f", m, k, num, f), func(t *testing.T) {
		ctx := context.Background()

		mid := int(float64(num) * f)
		c := newTestRedisClient(t)
		rf := NewRedis(c, t.Name(), m, k)
		for i := 0; i < mid; i++ {
			err := rf.Put(ctx, []byte(strconv.Itoa(i)))
			if err != nil {
				t.Fatalf("put failed: %s", err)
			}
		}

		falsePositive := 0
		for i := mid; i < num; i++ {
			has, err := rf.Check(ctx, []byte(strconv.Itoa(i)), 0)
			if err != nil {
				t.Fatalf("check failed: %s", err)
			}
			if has {
				falsePositive++
			}
		}
		errRate := float64(falsePositive) / float64(m) * 100
		if errRate > 1 {
			t.Errorf("too big error rate: %.2f%% false_positive=%d m=%d k=%d n=%d f=%f", errRate, falsePositive, m, k, num, f)
		}
	})
}

func TestRedisBasic(t *testing.T) {
	checkRedis(t, 1000, 7, 200, 0.1)
	checkRedis(t, 1000, 7, 200, 0.5)
	checkRedis(t, 1000, 7, 200, 0.9)
	checkRedis(t, 1000, 7, 400, 0.1)
	checkRedis(t, 1000, 7, 700, 0.1)
	checkRedis(t, 1000, 7, 1000, 0.1)
}

func TestRedisSubtract(t *testing.T) {
	c := newTestRedisClient(t)
	rf := NewRedis(c, t.Name(), 2048, 8)
	ctx := context.Background()
	for i := 1; i <= 255; i++ {
		rf.Subtract(ctx, 1)
		rf.Put(ctx, []byte(strconv.Itoa(i)))
	}
	failure, total := 0, 0
	for bias := 0; bias <= 255; bias++ {
		for i := 1; i <= 255; i++ {
			total++
			want := i > bias
			got, _ := rf.Check(ctx, []byte(strconv.Itoa(i)), uint8(bias))
			if got != want {
				if !got {
					t.Fatalf("false negative on: bias=%d i=%d", bias, i)
				}
				// record false positive.
				failure++
			}
		}
	}
	rate := float64(failure) / float64(total) * 100
	//t.Logf("error rate: %.2f%% failure=%d total=%d", rate, failure, total)
	if rate > 1 {
		t.Logf("too big error rate: %.2f%% failure=%d total=%d", rate, failure, total)
	}
}

func TestRedisSubtractOverflow(t *testing.T) {
	c := newTestRedisClient(t)
	rf := NewRedis(c, t.Name(), 2048, 8)
	ctx := context.Background()
	tail := 256 + 127
	for i := 1; i <= tail; i++ {
		rf.Subtract(ctx, 1)
		rf.Put(ctx, []byte(strconv.Itoa(i)))
	}
	failure, total := 0, 0
	for bias := 0; bias <= 255; bias++ {
		for i := 1; i <= 511; i++ {
			total++
			want := i > tail-255+bias && i <= tail
			got, _ := rf.Check(ctx, []byte(strconv.Itoa(i)), uint8(bias))
			if got != want {
				if !got {
					t.Fatalf("false negative on: bias=%d i=%d", bias, i)
				}
				// record false positive.
				failure++
			}
		}
	}
	rate := float64(failure) / float64(total) * 100
	//t.Logf("error rate: %.2f%% failure=%d total=%d", rate, failure, total)
	if rate > 1 {
		t.Logf("too big error rate: %.2f%% failure=%d total=%d", rate, failure, total)
	}
}

package vbf3redis

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"testing"
)

func checkVBF3Redis(t *testing.T, m uint64, k uint, num int, f float64) {
	t.Run(fmt.Sprintf("m=%d k=%d num=%d f=%f", m, k, num, f), func(t *testing.T) {
		ctx := context.Background()

		mid := int(float64(num) * f)
		c := newTestRedisClient(t)
		rf, err := Open(ctx, c, t.Name(), m, k, 10)
		if err != nil {
			t.Fatalf("failed to create: %s", err)
		}
		t.Cleanup(func() {
			rf.Drop(ctx)
		})

		vals := make([][]byte, 0, mid)
		for i := 0; i < mid; i++ {
			vals = append(vals, []byte(strconv.Itoa(i)))
		}
		err = rf.PutAll(ctx, 1, vals)
		if err != nil {
			t.Fatalf("put failed: %s", err)
		}

		falsePositive := 0
		for i := mid; i < num; i++ {
			has, err := rf.Check(ctx, []byte(strconv.Itoa(i)))
			if err != nil {
				t.Fatalf("check failed #%d: %s", i, err)
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

func TestVBF3RedisLarge(t *testing.T) {
	checkVBF3Redis(t, 8*512*1024*1024, 7, 1000, 0.1)
}

func TestVBF3RedisLife(t *testing.T) {
	c := newTestRedisClient(t)
	ctx := context.Background()
	rf, err := Open(ctx, c, t.Name(), 100, 7, 10)
	if err != nil {
		t.Fatalf("failed to create: %s", err)
	}
	t.Cleanup(func() {
		rf.Drop(ctx)
	})

	err = rf.Put(ctx, []byte("foo"), 1)
	if err != nil {
		t.Fatalf("put \"foo\" failed: %s", err)
	}
	err = rf.Put(ctx, []byte("bar"), 2)
	if err != nil {
		t.Fatalf("put \"bar\" failed: %s", err)
	}
	err = rf.Put(ctx, []byte("baz"), 3)
	if err != nil {
		t.Fatalf("put \"baz\" failed: %s", err)
	}

	for i := 0; i <= 3; i++ {
		for _, tc := range []struct {
			name string
			life int
		}{
			{"foo", 1},
			{"bar", 2},
			{"baz", 3},
		} {
			want := tc.life > i
			got, err := rf.Check(ctx, []byte(tc.name))
			if err != nil {
				t.Fatalf("check %q (#%d) failed: %s", tc.name, i, err)
			}
			if got != want {
				t.Errorf("unexpected check %q (%d): want=%t got=%t", tc.name, i, want, got)
			}
		}
		err := rf.AdvanceGeneration(ctx, 1)
		if err != nil {
			t.Fatalf("failed to AdvanceGeneration (%d): %s", i, err)
		}
	}
}

func benchmarkPut(b *testing.B, m uint64, k uint, maxLife uint8) {
	c := newTestRedisClient(b)
	ctx := context.Background()
	rf, err := Open(ctx, c, b.Name(), m, 7, 10)
	if err != nil {
		b.Fatalf("failed to create: %s", err)
	}
	b.Cleanup(func() {
		rf.Drop(ctx)
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := rf.Put(ctx, []byte(strconv.Itoa(i)), 1)
		if err != nil {
			b.Fatalf("put failed: %s", err)
		}
	}
}

func BenchmarkVBF3RedisPut(b *testing.B) {
	benchmarkPut(b, uint64(b.N*10), 7, 10)
}

func BenchmarkLargeVBF3RedisPut(b *testing.B) {
	for _, n := range []uint64{1, 2, 4, 8} {
		size := pageSize * n
		b.Run(fmt.Sprintf("size=%d n=%d", size, n), func(b *testing.B) {
			benchmarkPut(b, size, 7, 10)
		})
	}
}

func benchmarkCheck(b *testing.B, m uint64, k uint, maxLife uint8) {
	c := newTestRedisClient(b)
	ctx := context.Background()
	rf, err := Open(ctx, c, b.Name(), m, k, maxLife)
	if err != nil {
		b.Fatalf("failed to create: %s", err)
	}
	b.Cleanup(func() {
		rf.Drop(ctx)
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
				b.Fatalf("false negative detected!: N=%d x=%d", b.N, x)
			}
			fail++
		}
	}
	rate := float64(fail) / float64(b.N) * 100
	if rate > 1 && b.N > 100 {
		b.Logf("too big error rate: %.2f%% failure=%d total=%d", rate, fail, b.N)
	}
}

func BenchmarkVBF3RedisCheck(b *testing.B) {
	benchmarkCheck(b, uint64(b.N*10), 7, 10)
}

func BenchmarkLargeVBF3RedisCheck(b *testing.B) {
	for _, n := range []uint64{1, 2, 4, 8} {
		size := pageSize * n
		b.Run(fmt.Sprintf("size=%d page=%d", size, n), func(b *testing.B) {
			benchmarkCheck(b, size, 7, 10)
		})
	}
}

func BenchmarkVBF3RedisAdvanceGeneration(b *testing.B) {
	c := newTestRedisClient(b)
	ctx := context.Background()
	rf, err := Open(ctx, c, b.Name(), 10*1000, 7, 10)
	if err != nil {
		b.Fatalf("failed to create: %s", err)
	}
	b.Cleanup(func() {
		rf.Drop(ctx)
	})

	for i := 0; i < 1000; i++ {
		rf.Put(ctx, []byte(strconv.Itoa(i)), 128)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err = rf.AdvanceGeneration(ctx, 1)
		if err != nil {
			b.Fatalf("runtime error: %s", err)
		}
	}
}

func BenchmarkVBF3RedisSweep(b *testing.B) {
	c := newTestRedisClient(b)
	ctx := context.Background()
	rf, err := Open(ctx, c, b.Name(), uint64(b.N*10), 7, 10)
	if err != nil {
		b.Fatalf("failed to create: %s", err)
	}
	b.Cleanup(func() {
		rf.Drop(ctx)
	})

	buf := make([]byte, b.N*10)
	rand.Read(buf)
	_, err = c.Set(ctx, rf.key.data(0), buf, 0).Result()
	if err != nil {
		b.Fatalf("failed to setup: %s", err)
	}

	b.ResetTimer()
	err = rf.Sweep(ctx)
	if err != nil {
		b.Fatalf("runtime error: %s", err)
	}
}

func testVBF3RedisTopBottom(ctx context.Context, t *testing.T, rf *VBF3Redis, bottom, top uint8) {
	t.Helper()
	gen, err := getGen(ctx, rf.c, rf.key)
	if err != nil {
		t.Errorf("faield to get vbf3gen: %s", err)
		return
	}
	if gen.Bottom != bottom {
		t.Errorf("bottom mismatch: want=%d got=%d", bottom, gen.Bottom)
	}
	if gen.Top != top {
		t.Errorf("top mismatch: want=%d got=%d", top, gen.Top)
	}
}

func TestVBF3RedisAdvanceGeneration(t *testing.T) {
	c := newTestRedisClient(t)
	ctx := context.Background()
	rf, err := Open(ctx, c, t.Name(), 256, 1, 1)
	if err != nil {
		t.Fatalf("failed to create: %s", err)
	}
	t.Cleanup(func() {
		rf.Drop(ctx)
	})

	testVBF3RedisTopBottom(ctx, t, rf, 1, 1)
	if t.Failed() {
		t.Fatal("failed at first round")
	}

	for i := 2; i <= 255; i++ {
		rf.AdvanceGeneration(ctx, 1)
		want := uint8(i)
		testVBF3RedisTopBottom(ctx, t, rf, want, want)
		if t.Failed() {
			t.Fatalf("failed at round #%d", i)
		}
	}

	rf.AdvanceGeneration(ctx, 1)
	testVBF3RedisTopBottom(ctx, t, rf, 1, 1)
}

func TestCheckAll(t *testing.T) {
	ctx := context.Background()
	c := newTestRedisClient(t)
	rf, err := Open(ctx, c, t.Name(), 1000, 7, 10)
	if err != nil {
		t.Fatalf("failed to create: %s", err)
	}
	t.Cleanup(func() {
		rf.Drop(ctx)
	})

	var (
		foo = []byte("foo")
		bar = []byte("bar")
		baz = []byte("baz")
		qux = []byte("qux")
		xyz = []byte("xyz")
		non = []byte("non")
	)

	err = rf.PutAll(ctx, 1, [][]byte{foo, bar, baz})
	if err != nil {
		t.Fatalf("failed to PutAll: %s", err)
	}

	for i, c := range []struct {
		keys  [][]byte
		wants []bool
	}{
		{[][]byte{foo, bar, baz}, []bool{true, true, true}},
		{[][]byte{foo}, []bool{true}},
		{[][]byte{bar}, []bool{true}},
		{[][]byte{baz}, []bool{true}},

		{[][]byte{qux, xyz, non}, []bool{false, false, false}},
		{[][]byte{qux}, []bool{false}},
		{[][]byte{xyz}, []bool{false}},
		{[][]byte{non}, []bool{false}},

		{[][]byte{foo, qux, xyz}, []bool{true, false, false}},
		{[][]byte{xyz, foo, qux}, []bool{false, true, false}},
		{[][]byte{qux, xyz, foo}, []bool{false, false, true}},
	} {
		gots, err := rf.CheckAll(ctx, c.keys)
		if err != nil {
			t.Fatalf("failed to CheckAll i=%d c=%+v: %s", i, c, err)
		}
		for j, want := range c.wants {
			if gots[j] != want {
				t.Errorf("unmatch i=%d j=%d want=%t got=%t", i, j, want, gots[j])
			}
		}
	}
}

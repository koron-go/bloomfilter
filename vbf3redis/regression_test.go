package vbf3redis

import (
	"context"
	"fmt"
	"strconv"
	"testing"
)

func TestLargeFalseNegative(t *testing.T) {
	const m = 2147483648
	const target = 0

	for _, tc := range []struct {
		m uint64
		k uint
		l uint8
		x int
	}{
		{m: 2147483648, k: 7, l: 10, x: 0},
	} {
		m, k, l, x := tc.m, tc.k, tc.l, tc.x
		t.Run(fmt.Sprintf("m=%d x=%d", m, x), func(t *testing.T) {
			c := newTestRedisClient(t)
			ctx := context.Background()
			Drop(ctx, c, t.Name())
			rf, err := Open(ctx, c, t.Name(), m, k, l)
			if err != nil {
				t.Fatalf("open failed: %s", err)
			}
			t.Cleanup(func() {
				rf.Drop(ctx)
			})

			v := []byte(strconv.Itoa(x))
			err = rf.Put(ctx, v, 1)
			if err != nil {
				t.Fatalf("put failed: %s", err)
			}
			got, err := rf.Check(ctx, v)
			if err != nil {
				t.Fatalf("check failed: %s", err)
			}
			if !got {
				t.Errorf("false negative: m=%d x=%d", m, x)
			}
		})
	}
}

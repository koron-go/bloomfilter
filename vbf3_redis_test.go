package bloomfilter

import (
	"context"
	"fmt"
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

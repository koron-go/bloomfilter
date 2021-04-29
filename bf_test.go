package bloomfilter

import (
	"context"
	"strconv"
	"testing"
)

func checkBlooFilter(tb testing.TB, m, k, n int, f float64) {
	tb.Helper()
	mid := int(float64(n) * f)
	bf := New(m, k, nil, nil)
	ctx := context.Background()
	for i := 0; i < mid; i++ {
		s := strconv.Itoa(i)
		err := bf.PutString(ctx, s)
		if err != nil {
			tb.Fatal(err)
		}
	}

	// check no false negative entries.
	for i := 0; i < mid; i++ {
		s := strconv.Itoa(i)
		has, err := bf.CheckString(ctx, s)
		if err != nil {
			tb.Fatal(err)
		}
		if !has {
			tb.Errorf("false negative: %d", i)
		}
	}

	falsePositive := 0
	for i := mid; i < n; i++ {
		s := strconv.Itoa(i)
		has, err := bf.CheckString(ctx, s)
		if err != nil {
			tb.Fatal(err)
		}
		if has {
			falsePositive++
		}
	}
	errRate := float64(falsePositive) / float64(n) * 100
	//tb.Logf("error rate: %.2f%% false_positive=%d m=%d k=%d n=%d f=%f", errRate, falsePositive, m, k, n, f)
	if errRate > 1 {
		tb.Errorf("too big error rate: %.2f%% false_positive=%d m=%d k=%d n=%d f=%f", errRate, falsePositive, m, k, n, f)
	}
}

func TestBF_Basic(t *testing.T) {
	checkBlooFilter(t, 1000, 7, 200, 0.1)
	checkBlooFilter(t, 1000, 7, 200, 0.5)
	checkBlooFilter(t, 1000, 7, 200, 0.9)

	checkBlooFilter(t, 1000, 7, 400, 0.1)
	checkBlooFilter(t, 1000, 7, 700, 0.1)
	checkBlooFilter(t, 1000, 7, 1000, 0.1)
}

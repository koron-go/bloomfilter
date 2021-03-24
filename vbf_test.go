package bloomfilter

import (
	"strconv"
	"testing"
)

func TestVBFSimple8(t *testing.T) {
	// constructor
	vf, err := NewVBF(100, 8, 255)
	if err != nil {
		t.Fatal(err)
	}
	if vf.nbits != 8 {
		t.Errorf("unexpected vf.nbits: want=8 got=%d", vf.nbits)
	}
	if len(vf.data) != 100 {
		t.Errorf("unexpected len(vf.data): want=100 got=%d", len(vf.data))
	}
	if vf.max != 255 {
		t.Errorf("unexpected vf.max: want=255 got=%d", vf.max)
	}
	if vf.curr != 1 {
		t.Errorf("unexpected vf.curr: want=1 got=%d", vf.curr)
	}

	samples := make([]uint8, 100)
	for i := range samples {
		samples[i] = uint8(i)
		vf.putData(i, uint8(i))
	}
	for i := range samples {
		x := vf.getData(i)
		if x != samples[i] {
			t.Errorf("data mismatch at %d: want=%02x got=%02x", i, samples[i], x)
			break
		}
	}
}

func checkVFB(tb testing.TB, m, k, n int, f float64, ttl uint8) {
	tb.Helper()
	mid := int(float64(n) * f)
	vf, err := NewVBF(m, k, ttl)
	if err != nil {
		tb.Fatal(err)
	}
	for i := 0; i < mid; i++ {
		s := strconv.Itoa(i)
		vf.Put([]byte(s))
	}

	// check no false negative entries.
	for i := 0; i < mid; i++ {
		s := strconv.Itoa(i)
		has := vf.Check([]byte(s), 0)
		if !has {
			tb.Errorf("false negative: %d", i)
		}
	}

	falsePositive := 0
	for i := mid; i < n; i++ {
		s := strconv.Itoa(i)
		has := vf.Check([]byte(s), 0)
		if err != nil {
			tb.Fatal(err)
		}
		if has {
			falsePositive++
		}
	}
	errRate := float64(falsePositive) / float64(m) * 100
	tb.Logf("error rate: %.2f%% false_positive=%d m=%d k=%d n=%d f=%f", errRate, falsePositive, m, k, n, f)
	if errRate > 1 {
		tb.Errorf("too big error rate: %.2f%% false_positive=%d m=%d k=%d n=%d f=%f ttl=%d", errRate, falsePositive, m, k, n, f, ttl)
	}
}

func TestVBFBasic1(t *testing.T) {
	checkVFB(t, 1000, 7, 200, 0.1, 1)
	checkVFB(t, 1000, 7, 200, 0.5, 1)
	checkVFB(t, 1000, 7, 200, 0.9, 1)
	checkVFB(t, 1000, 7, 400, 0.1, 1)
	checkVFB(t, 1000, 7, 700, 0.1, 1)
	checkVFB(t, 1000, 7, 1000, 0.1, 1)
}

func TestVBFBasic3(t *testing.T) {
	checkVFB(t, 1000, 7, 200, 0.1, 3)
	checkVFB(t, 1000, 7, 200, 0.5, 3)
	checkVFB(t, 1000, 7, 200, 0.9, 3)
	checkVFB(t, 1000, 7, 400, 0.1, 3)
	checkVFB(t, 1000, 7, 700, 0.1, 3)
	checkVFB(t, 1000, 7, 1000, 0.1, 3)
}

func TestVBFBasic15(t *testing.T) {
	checkVFB(t, 1000, 7, 200, 0.1, 15)
	checkVFB(t, 1000, 7, 200, 0.5, 15)
	checkVFB(t, 1000, 7, 200, 0.9, 15)
	checkVFB(t, 1000, 7, 400, 0.1, 15)
	checkVFB(t, 1000, 7, 700, 0.1, 15)
	checkVFB(t, 1000, 7, 1000, 0.1, 15)
}

func TestVBFBasic255(t *testing.T) {
	checkVFB(t, 1000, 7, 200, 0.1, 255)
	checkVFB(t, 1000, 7, 200, 0.5, 255)
	checkVFB(t, 1000, 7, 200, 0.9, 255)
	checkVFB(t, 1000, 7, 400, 0.1, 255)
	checkVFB(t, 1000, 7, 700, 0.1, 255)
	checkVFB(t, 1000, 7, 1000, 0.1, 255)
}

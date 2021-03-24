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

func checkVBF(tb testing.TB, m, k, n int, f float64, ttl uint8) {
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
	checkVBF(t, 1000, 7, 200, 0.1, 1)
	checkVBF(t, 1000, 7, 200, 0.5, 1)
	checkVBF(t, 1000, 7, 200, 0.9, 1)
	checkVBF(t, 1000, 7, 400, 0.1, 1)
	checkVBF(t, 1000, 7, 700, 0.1, 1)
	checkVBF(t, 1000, 7, 1000, 0.1, 1)
}

func TestVBFBasic3(t *testing.T) {
	checkVBF(t, 1000, 7, 200, 0.1, 3)
	checkVBF(t, 1000, 7, 200, 0.5, 3)
	checkVBF(t, 1000, 7, 200, 0.9, 3)
	checkVBF(t, 1000, 7, 400, 0.1, 3)
	checkVBF(t, 1000, 7, 700, 0.1, 3)
	checkVBF(t, 1000, 7, 1000, 0.1, 3)
}

func TestVBFBasic15(t *testing.T) {
	checkVBF(t, 1000, 7, 200, 0.1, 15)
	checkVBF(t, 1000, 7, 200, 0.5, 15)
	checkVBF(t, 1000, 7, 200, 0.9, 15)
	checkVBF(t, 1000, 7, 400, 0.1, 15)
	checkVBF(t, 1000, 7, 700, 0.1, 15)
	checkVBF(t, 1000, 7, 1000, 0.1, 15)
}

func TestVBFBasic255(t *testing.T) {
	checkVBF(t, 1000, 7, 200, 0.1, 255)
	checkVBF(t, 1000, 7, 200, 0.5, 255)
	checkVBF(t, 1000, 7, 200, 0.9, 255)
	checkVBF(t, 1000, 7, 400, 0.1, 255)
	checkVBF(t, 1000, 7, 700, 0.1, 255)
	checkVBF(t, 1000, 7, 1000, 0.1, 255)
}

func TestVBFMargin(t *testing.T) {
	vf, err := NewVBF(2048, 8, 255)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i <= 255; i++ {
		vf.SetCurr(uint8(i))
		vf.Put([]byte(strconv.Itoa(i)))
	}
	failure, total := 0, 0
	for margin := 0; margin <= 255; margin++ {
		threshold := vf.Curr() - uint8(margin)
		for i := 0; i <= 255; i++ {
			total++
			has := vf.Check([]byte(strconv.Itoa(i)), uint8(margin))
			want := uint8(i) >= threshold
			if has != want {
				// detect false negative: should not be
				if !has {
					t.Fatalf("false negative on: margin=%d i=%d", margin, i)
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

func TestVBFMargin2(t *testing.T) {
	vf, err := NewVBF(2048, 6, 255)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i <= 255; i++ {
		vf.SetCurr(uint8(i))
		vf.Put([]byte(strconv.Itoa(i)))
	}
	for i := 0; i <= 127; i++ {
		vf.SetCurr(uint8(i))
		vf.Put([]byte(strconv.Itoa(i)))
	}
	failure, total := 0, 0

	for margin := 0; margin <= 120; margin++ {
		threshold := vf.Curr() - uint8(margin)
		for i := 0; i <= 255; i++ {
			total++
			has := vf.Check([]byte(strconv.Itoa(i)), uint8(margin))
			want := uint8(i) <= threshold
			if has != want {
				// detect false negative: should not be
				if !has {
					t.Fatalf("false negative on: margin=%d i=%d, threshold=%d", margin, i, threshold)
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

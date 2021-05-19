package bloomfilter

import (
	"strconv"
	"testing"
)

func TestVBF3currLife1(t *testing.T) {
	f := NewVBF3(256, 1, 1)
	for i := range f.data {
		f.data[i] = byte(i)
	}
	for i := 1; i <= 255; i++ {
		f.top = uint8(i)
		f.bottom = uint8(i)
		for j := range f.data {
			want := j == i
			got := f.currLife(j) == 1
			if got != want {
				t.Errorf("unexpected: i=%d j=%d want=%t got=%t", i, j, want, got)
				break
			}
		}
	}
}

func TestVBF3currLife(t *testing.T) {
	for maxLife := 1; maxLife <= 255; maxLife++ {
		f := NewVBF3(256, 1, uint8(maxLife))
		for i := range f.data {
			f.data[i] = byte(i)
		}
		for i := 0; i <= 255; i++ {
			for j, v := range f.data {
				var want uint8
				if f.bottom <= f.top {
					if v >= f.bottom && v <= f.top {
						want = v - f.bottom + 1
					}
				} else {
					if v >= f.bottom {
						want = v - f.bottom + 1
					} else if v <= f.top && v != 0 {
						want = v - f.bottom
					}
				}
				got := f.currLife(j)
				if got != want {
					t.Fatalf("unexpected life at bottom=%d top=%d data[%d]=%d: want=%d got=%d", f.bottom, f.top, j, f.data[j], want, got)
				}
			}
			f.AdvanceGeneration(1)
		}
	}
}

func testTopBottom(t *testing.T, f *VBF3, bottom, top uint8) {
	t.Helper()
	if f.bottom != bottom {
		t.Errorf("bottom mismatch: want=%d got=%d", bottom, f.bottom)
	}
	if f.top != top {
		t.Errorf("top mismatch: want=%d got=%d", top, f.top)
	}
}

func TestVBF3AdvanceGeneration(t *testing.T) {
	f := NewVBF3(256, 1, 1)
	testTopBottom(t, f, 1, 1)
	if t.Failed() {
		t.Fatal("failed at first round")
	}
	for i := 2; i <= 255; i++ {
		f.AdvanceGeneration(1)
		want := uint8(i)
		testTopBottom(t, f, want, want)
		if t.Failed() {
			t.Fatalf("failed at round #%d", i)
		}
	}
	f.AdvanceGeneration(1)
	testTopBottom(t, f, 1, 1)
}

func checkVBF3(tb testing.TB, m, k, num int, f float64) {
	tb.Helper()
	mid := int(float64(num) * f)
	vf := NewVBF3(m, k, 64)
	for i := 0; i < mid; i++ {
		s := strconv.Itoa(i)
		vf.Put([]byte(s), 1)
	}

	// check no false negative entries.
	for i := 0; i < mid; i++ {
		s := strconv.Itoa(i)
		has := vf.Check([]byte(s))
		if !has {
			tb.Fatalf("false negative: %d", i)
		}
	}

	falsePositive := 0
	for i := mid; i < num; i++ {
		s := strconv.Itoa(i)
		has := vf.Check([]byte(s))
		if has {
			falsePositive++
		}
	}
	errRate := float64(falsePositive) / float64(num) * 100
	if errRate > 1 {
		tb.Errorf("too big error rate: %.2f%% false_positive=%d m=%d k=%d n=%d f=%f", errRate, falsePositive, m, k, num, f)
	}
}

func TestVBF3Basic(t *testing.T) {
	checkVBF3(t, 1000, 7, 200, 0.1)
	checkVBF3(t, 1000, 7, 200, 0.5)
	checkVBF3(t, 1000, 7, 200, 0.9)
	checkVBF3(t, 1000, 7, 400, 0.1)
	checkVBF3(t, 1000, 7, 700, 0.1)
	checkVBF3(t, 1000, 7, 1000, 0.1)
}

func TestVBF3EvaporateOne(t *testing.T) {
	keyOne := []byte("1")
	keyTwo := []byte("2")

	f := NewVBF3(1000, 7, 64)
	f.Put(keyOne, 1)
	f.Put(keyTwo, 2)

	hasOne := f.Check(keyOne)
	if !hasOne {
		t.Fatalf("keyOne is not stored")
	}
	hasTwo := f.Check(keyTwo)
	if !hasTwo {
		t.Fatalf("keyTwo is not stored")
	}

	f.AdvanceGeneration(1)
	has := f.Check(keyOne)
	if has {
		t.Errorf("keyOne should be evaporated")
	}
	hasTwo = f.Check(keyTwo)
	if !hasTwo {
		t.Fatalf("keyTwo is not stored after 1 advanced")
	}
}

func TestVBF3Evaporate64(t *testing.T) {
	f := NewVBF3(1000, 7, 64)
	for i := 1; i <= 64; i++ {
		f.Put([]byte(strconv.Itoa(i)), uint8(i))
	}

	// check no false negative entries.
	for i := 1; i <= 64; i++ {
		has := f.Check([]byte(strconv.Itoa(i)))
		if !has {
			t.Fatalf("false negative: %d", i)
		}
	}

	for i := 1; i <= 64; i++ {
		f.AdvanceGeneration(1)
		for j := 1; j <= 64; j++ {
			want := j > i
			got := f.Check([]byte(strconv.Itoa(j)))
			if got != want {
				t.Errorf("mismatch i=%d j=%d want=%t got=%t", i, j, want, got)
			}
		}
		if t.Failed() {
			break
		}
	}
}

func TestVBF3EvaporateWrap(t *testing.T) {
	for i := 1; i <= 64; i++ {
		f := NewVBF3(1000, 7, 64)
		for j := 1; j <= 64; j++ {
			f.Put([]byte(strconv.Itoa(j)), uint8(j))
		}

		f.AdvanceGeneration(255)
		f.AdvanceGeneration(uint8(i))

		for j := 1; j <= 64; j++ {
			want := j > i
			got := f.Check([]byte(strconv.Itoa(j)))
			if got != want {
				t.Errorf("mismatch i=%d j=%d want=%t got=%t", i, j, want, got)
			}
		}
		if t.Failed() {
			break
		}
	}
}

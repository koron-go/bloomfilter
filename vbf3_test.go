package bloomfilter

import "testing"

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
			f.Subtract(1)
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

func TestVBF3Subtract(t *testing.T) {
	f := NewVBF3(256, 1, 1)
	testTopBottom(t, f, 1, 1)
	if t.Failed() {
		t.Fatal("failed at first round")
	}
	for i := 2; i <= 255; i++ {
		f.Subtract(1)
		want := uint8(i)
		testTopBottom(t, f, want, want)
		if t.Failed() {
			t.Fatalf("failed at round #%d", i)
		}
	}
	f.Subtract(1)
	testTopBottom(t, f, 1, 1)
}

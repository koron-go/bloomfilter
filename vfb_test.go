package bloomfilter

import "testing"

func TestVBFBasic8(t *testing.T) {
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

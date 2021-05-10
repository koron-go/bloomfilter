package bloomfilter

import (
	"fmt"

	"github.com/dgryski/go-metro"
)

type VBF3 struct {
	m    int
	k    int
	data []byte

	bottom uint8
	top    uint8
	max    uint8
}

func NewVBF3(m, k int, maxLife uint8) *VBF3 {
	return &VBF3{
		m:    m,
		k:    k,
		data: make([]byte, m),

		bottom: 1,
		top:    maxLife,
		max:    maxLife,
	}
}

func (f *VBF3) m255p1add(a, b uint8) uint8 {
	v := uint16(a) + uint16(b)
	if v <= 255 {
		return uint8(v)
	}
	return uint8(v - 255)
}

func (f *VBF3) hash(d []byte, n int) int {
	return int(metro.Hash64(d, uint64(n)) % uint64(f.m))
}

func (f *VBF3) isValid(n uint8) bool {
	if n == 0 {
		return false
	}
	a := f.bottom <= n
	b := f.top >= n
	if f.bottom <= f.top {
		return a && b
	}
	return a || b
}

func (f *VBF3) currLife(x int) uint8 {
	v := f.data[x]
	if !f.isValid(v) {
		return 0
	}
	d := v - f.bottom + 1
	if v < f.bottom {
		d--
	}
	return d
}

func (f *VBF3) Put(d []byte, life uint8) {
	if life > f.max {
		panic(fmt.Sprintf("life should be <= %d", f.max))
	}
	nv := f.m255p1add(f.bottom, life-1)
	for i := 0; i < f.k; i++ {
		x := f.hash(d, i)
		v := f.currLife(x)
		if v == 0 || life > v {
			f.data[x] = nv
		}
	}
}

func (f *VBF3) Check(d []byte) bool {
	retval := true
	for i := 0; i < f.k; i++ {
		x := f.hash(d, i)
		v := f.data[x]
		if !f.isValid(v) {
			retval = false
			if v != 0 {
				f.data[x] = 0
			}
		}
	}
	return retval
}

func (f *VBF3) AdvanceGeneration(generations uint8) {
	f.bottom = f.m255p1add(f.bottom, generations)
	f.top = f.m255p1add(f.top, generations)
}

func (f *VBF3) Sweep() {
	for i, v := range f.data {
		if !f.isValid(v) {
			f.data[i] = 0
		}
	}
}

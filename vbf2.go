package bloomfilter

import (
	"fmt"

	"github.com/dgryski/go-metro"
)

type VBF2 struct {
	m int
	k int

	nbits uint8
	data  []byte
	max   uint8
}

func NewVBF2(m, k int, nbits uint8) *VBF2 {
	if nbits < 1 || nbits > 8 {
		panic(fmt.Sprintf("nbits out of range"))
	}
	return &VBF2{
		m:     m,
		k:     k,
		nbits: nbits,
		data:  make([]byte, (int(nbits)*m+7)/8),
		max:   uint8((uint16(1) << nbits) - 1),
	}
}

func (vf *VBF2) putData(x int, v uint8) {
	switch vf.nbits {
	case 1:
		y := 7 - x%8
		d := vf.data[x/8]
		d &= ^(0x01 << y)
		d |= (v & 0x01) << y
		vf.data[x/8] = d

	case 2:
		y := 6 - (x%4)*2
		d := vf.data[x/4]
		d &= ^(0x03 << y)
		d |= (v & 0x03) << y
		vf.data[x/4] = d

	case 4:
		y := 4 - (x%2)*4
		d := vf.data[x/2]
		d &= ^(0x0f << y)
		d |= (v & 0x0f) << y
		vf.data[x/2] = d

	case 8:
		vf.data[x] = v

	default:
		panic(fmt.Sprintf("unsuppoted nbits: %d", vf.nbits))
	}
}

func (vf *VBF2) getData(x int) uint8 {
	switch vf.nbits {
	case 1:
		y := 7 - x%8
		return (vf.data[x/8] >> y) & 0x01
	case 2:
		y := 6 - (x%4)*2
		return (vf.data[x/4] >> y) & 0x03
	case 4:
		y := 4 - (x%2)*4
		return (vf.data[x/2] >> y) & 0x0f
	case 8:
		return vf.data[x]
	default:
		panic(fmt.Sprintf("unsuppoted nbits: %d", vf.nbits))
	}
}

func (vf *VBF2) Put(d []byte) {
	for i := 0; i < vf.k; i++ {
		x := int(metro.Hash64(d, uint64(i)) % uint64(vf.m))
		vf.putData(x, vf.max)
	}
}

func (vf *VBF2) Check(d []byte, bias uint8) bool {
	for i := 0; i < vf.k; i++ {
		x := int(metro.Hash64(d, uint64(i)) % uint64(vf.m))
		v := vf.getData(x)
		if v <= bias {
			return false
		}
	}
	return true
}

func (vf *VBF2) Subtract(delta uint8) {
	for i := 0; i < vf.m; i++ {
		v := vf.getData(i)
		if v > delta {
			v -= delta
		} else {
			v = 0
		}
		vf.putData(i, v)
	}
}

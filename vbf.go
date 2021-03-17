package bloomfilter

import (
	"context"
	"fmt"

	"github.com/dgryski/go-metro"
)

type VBF struct {
	m int
	k int

	nbits int
	data  []byte

	max  uint8
	curr uint8
}

func vbfBits(ttl uint8) int {
	for i := 1; i <= 31; i++ {
		if (1<<i)-1 >= ttl {
			return i
		}
	}
	return -1
}

func NewVBF(m, k int, ttl uint8) (*VBF, error) {
	if ttl < 1 {
		ttl = 1
	}
	nbits := vbfBits(ttl)
	if nbits < 1 || nbits > 8 {
		return nil, fmt.Errorf("over TTL: ttl=%d nbits=%d", ttl, nbits)
	}
	return &VBF{
		m:     m,
		k:     k,
		nbits: nbits,
		data:  make([]byte, (nbits*m+7)/8),
		max:   ttl,
		curr:  1,
	}, nil
}

func (vf *VBF) indexes(d []byte) []int {
	indexes := make([]int, vf.k)
	for i := 0; i < vf.k; i++ {
		h := metro.Hash64(d, uint64(i))
		indexes[i] = int(h % uint64(vf.m))
	}
	return indexes
}

func (vf *VBF) putData(x int, v uint8) {
	switch vf.nbits {
	case 1:
		y := 7 - x%8
		d := vf.data[x/8]
		d &= ^(1 << y)
		d |= (v & 1) << y
		vf.data[x/8] = d
	case 8:
		vf.data[x] = v
	default:
		// TODO:
	}
}

func (vf *VBF) getData(x int) uint8 {
	switch vf.nbits {
	case 1:
		y := 7 - x%8
		return (vf.data[x/8] >> y) & 0x01
	case 8:
		return vf.data[x]
	default:
		// TODO:
		return 0
	}
}

func (vf *VBF) Put(d []byte) {
	indexes := vf.indexes(d)
	for _, x := range indexes {
		vf.putData(x, vf.curr)
	}
}

func (vf *VBF) Check(ctx context.Context, d []byte, margin uint8) bool {
	indexes := vf.indexes(d)
	for _, x := range indexes {
		v := vf.getData(x)
		if v == 0 {
			return false
		}
		if v <= vf.curr {
			v += vf.max - vf.curr
		} else {
			v -= vf.curr
		}
		if v < vf.curr-margin {
			return false
		}
	}
	return true
}

func (vf *VBF) SetCurr(curr uint8) bool {
	if curr == 0 || curr > vf.max {
		return false
	}
	vf.curr = vf.max
	return true
}

func (vf *VBF) Curr() uint8 {
	return vf.curr
}

func (vf *VBF) Max() uint8 {
	return vf.max
}

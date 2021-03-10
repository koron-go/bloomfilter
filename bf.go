package bloomfilter

import (
	"context"
	"fmt"
)

// BF provides standard bloom filter algorithm.
type BF struct {
	m int
	k int
	h Hasher
	s Store
}

// New creates a bloom filter.
func New(m, k int, h Hasher, s Store) *BF {
	if h == nil {
		h = NewHasher(k, m)
	}
	if s == nil {
		s = NewMemoryStore(m)
	}
	return &BF{
		m: m,
		k: k,
		h: h,
		s: s,
	}
}

func (bf *BF) indexes(ctx context.Context, d []byte) ([]int, error) {
	indexes := make([]int, bf.k)
	for i := 0; i < bf.k; i++ {
		x, err := bf.h.Hash(ctx, i, d)
		if err != nil {
			return nil, fmt.Errorf("hash failed for k=%d: %w", i, err)
		}
		if x < 0 || x >= bf.m {
			return nil, fmt.Errorf("hasher out of range: k=%d got=%d want=0~%d", i, x, bf.m)
		}
		indexes[i] = x
	}
	return indexes, nil
}

// Put puts a byte array to the filter.
func (bf *BF) Put(ctx context.Context, d []byte) error {
	indexes, err := bf.indexes(ctx, d)
	if err != nil {
		return err
	}
	err = bf.s.SetBits(ctx, indexes...)
	if err != nil {
		return fmt.Errorf("store SetBits failed: indexes=%+v: %w", indexes, err)
	}
	return nil
}

// PutString puts a string to the filter.
func (bf *BF) PutString(ctx context.Context, s string) error {
	return bf.Put(ctx, []byte(s))
}

// Check checks that a byte array is in the filter.
func (bf *BF) Check(ctx context.Context, d []byte) (bool, error) {
	indexes, err := bf.indexes(ctx, d)
	if err != nil {
		return false, err
	}
	r, err := bf.s.CheckBits(ctx, indexes...)
	if err != nil {
		return false, fmt.Errorf("store CheckBits failed: indexes=%+v: %w", indexes, err)
	}
	return r, nil
}

// CheckString checks that a byte array is in the filter.
func (bf *BF) CheckString(ctx context.Context, s string) (bool, error) {
	return bf.Check(ctx, []byte(s))
}

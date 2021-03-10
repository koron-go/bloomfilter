package bloomfilter

import (
	"context"
)

// Store defines bits store for bloom filter (BF).
type Store interface {
	// SetBits sets bits on indexes in the store.
	SetBits(ctx context.Context, indexes ...int) error
	// CheckBits checks all bits are `true` on indexes in the store.
	CheckBits(ctx context.Context, indexes ...int) (bool, error)
}

// MemoryStore provides Store interface with memory.
type MemoryStore []byte

// NewMemoryStore creates a memory store for bloom filter.
func NewMemoryStore(nbits int) MemoryStore {
	nbytes := (nbits + 7) / 8
	regs := make([]byte, nbytes)
	return MemoryStore(regs)
}

// SetBits sets bits on indexes in the store.
func (ms MemoryStore) SetBits(_ context.Context, indexes ...int) error {
	for _, x := range indexes {
		ms[x/8] |= 1 << (x % 8)
	}
	return nil
}

// CheckBits checks all bits are `true` on indexes in the store.
func (ms MemoryStore) CheckBits(_ context.Context, indexes ...int) (bool, error) {
	if len(indexes) == 0 {
		return false, nil
	}
	for _, x := range indexes {
		if ms[x/8]&(1<<(x%8)) == 0 {
			return false, nil
		}
	}
	return true, nil
}

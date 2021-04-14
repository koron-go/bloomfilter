package bloomfilter

import (
	"context"

	"github.com/dgryski/go-metro"
)

// Hasher defines hashing method for bloom filter.
type Hasher interface {
	// Hash compute
	Hash(ctx context.Context, k int, d []byte) (int, error)
}

type metroHash struct {
	k int
	m int
}

// NewHasher creates a default hasher.
func NewHasher(k, m int) Hasher {
	return &metroHash{k: k, m: m}
}

func (mh *metroHash) Hash(_ context.Context, k int, d []byte) (int, error) {
	// FIXME: should be check that `k` is between 0 and (mh.k-1)?
	h := metro.Hash64(d, uint64(k))
	return int(h % uint64(mh.m)), nil
}

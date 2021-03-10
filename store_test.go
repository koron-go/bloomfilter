package bloomfilter

import (
	"context"
	"testing"
)

func checkStoreTrue(ctx context.Context, tb testing.TB, s Store, indexes ...int) {
	tb.Helper()
	r, err := s.CheckBits(ctx, indexes...)
	if err != nil {
		tb.Fatalf("CheckBits failed: %s", err)
	}
	if !r {
		tb.Errorf("CheckBits returns false unexpectedly for: %+v", indexes)
	}
}

func checkStoreFalse(ctx context.Context, tb testing.TB, s Store, indexes ...int) {
	tb.Helper()
	r, err := s.CheckBits(ctx, indexes...)
	if err != nil {
		tb.Fatalf("CheckBits failed: %s", err)
	}
	if r {
		tb.Errorf("CheckBits returns true unexpectedly for: %+v", indexes)
	}
}

func TestMemoryStore(t *testing.T) {
	ms := NewMemoryStore(32)
	ctx := context.Background()
	err := ms.SetBits(ctx, 0, 1, 2, 3, 8, 9, 10, 11, 12, 13, 14, 15)
	if err != nil {
		t.Fatal(err)
	}

	checkStoreTrue(ctx, t, ms, 0)
	checkStoreTrue(ctx, t, ms, 0, 1)
	checkStoreTrue(ctx, t, ms, 1, 2)
	checkStoreTrue(ctx, t, ms, 2, 3)
	checkStoreTrue(ctx, t, ms, 0, 1, 2, 3)

	checkStoreFalse(ctx, t, ms, 4)
	checkStoreFalse(ctx, t, ms, 5)
	checkStoreFalse(ctx, t, ms, 6)
	checkStoreFalse(ctx, t, ms, 7)
	checkStoreFalse(ctx, t, ms, 0, 4)
	checkStoreFalse(ctx, t, ms, 0, 1, 2, 3, 4)
	checkStoreTrue(ctx, t, ms, 0)

	checkStoreTrue(ctx, t, ms, 8)
	checkStoreTrue(ctx, t, ms, 12)
	checkStoreTrue(ctx, t, ms, 15)
}

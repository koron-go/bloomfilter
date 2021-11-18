package vbf3redis

import (
	"context"
	"fmt"
	"testing"
)

func Test20211118(t *testing.T) {
	const (
		m      = 140 * 1024
		k      = 10
		nValid = 100
	)
	ctx := context.Background()
	c := newTestRedisClient(t)
	rf, err := Open(ctx, c, t.Name(), m, k, 10)
	if err != nil {
		t.Fatalf("failed to create: %s", err)
	}
	t.Cleanup(func() {
		rf.Drop(ctx)
	})

	//vals := make([][]byte, 0, nValid)
	/*
		for i := 0; i < nValid; i++ {
			err := rf.Put(ctx, []byte(fmt.Sprintf("value%04d", i)), 1)
			if err != nil {
				t.Fatalf("put failed: m=%d k=%d nValid=%d i=%d: %s", m, k, nValid, i, err)
			}
		}
	*/

	vals := make([][]byte, 0, nValid)
	for i := 0; i < nValid; i++ {
		vals = append(vals, []byte(fmt.Sprintf("value%04d", i)))
	}
	err = rf.PutAll(ctx, 1, vals)
	if err != nil {
		t.Fatalf("put failed: %s", err)
	}
	vals = nil

	for i := 0; i < nValid; i++ {
		has, err := rf.Check(ctx, []byte(fmt.Sprintf("value%04d", i)))
		if err != nil {
			t.Fatalf("check failed: m=%d k=%d nValid=%d i=%d: %s", m, k, nValid, i, err)
		}
		if !has {
			t.Fatalf("check return false unexpectedly: m=%d k=%d nValid=%d i=%d", m, k, nValid, i)
		}
	}
}

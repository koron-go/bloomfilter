package vbf3redis

import (
	"os"
	"testing"

	"github.com/go-redis/redis/v8"
)

func newTestRedisClient(tb testing.TB) *redis.Client {
	u, ok := os.LookupEnv("REDIS_URL")
	if !ok {
		tb.Skip("please set REDIS_URL to enable this test/benchmark")
	}
	opts, err := redis.ParseURL(u)
	if err != nil {
		tb.Helper()
		tb.Fatal(err)
	}
	c := redis.NewClient(opts)
	return c
}

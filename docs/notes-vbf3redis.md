# VBF3 Redisのパフォーマンス測定&改良

VBF3 RedisのPutはVBF2 Redisの13倍遅い。

```console
$ go test -v -run no -bench 'Benchmark(VBF3)?RedisPut'
goos: windows
goarch: amd64
pkg: github.com/koron-go/bloomfilter
cpu: Intel(R) Core(TM) i9-9900K CPU @ 3.60GHz
BenchmarkRedisPut
BenchmarkRedisPut-16                2233            518175 ns/op
BenchmarkVBF3RedisPut
BenchmarkVBF3RedisPut-16             184           6715018 ns/op
PASS
ok      github.com/koron-go/bloomfilter 3.211s
```

もともとコマンド1個で書くだけだったのが、読んで書いてを各k回行うようになり、
ベンチマークでは `k = 7` なので妥当な速度低下。

まとめて読み書きするように直してみるか。

BitField を2回にしてみた結果3倍程度まで圧縮できた。

```console
$ go test -v -run no -bench 'Benchmark(VBF3)?RedisPut'
goos: windows
goarch: amd64
pkg: github.com/koron-go/bloomfilter
cpu: Intel(R) Core(TM) i9-9900K CPU @ 3.60GHz
BenchmarkRedisPut
BenchmarkRedisPut-16                2293            516261 ns/op
BenchmarkVBF3RedisPut
BenchmarkVBF3RedisPut-16             783           1510346 ns/op
PASS
ok      github.com/koron-go/bloomfilter 2.675s
```

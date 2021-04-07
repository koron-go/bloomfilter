# koron-go/bloomfilter

[![PkgGoDev](https://pkg.go.dev/badge/github.com/koron-go/bloomfilter)](https://pkg.go.dev/github.com/koron-go/bloomfilter)
[![GoDoc](https://godoc.org/github.com/koron-go/bloomfilter?status.svg)](https://godoc.org/github.com/koron-go/bloomfilter)
[![Actions/Go](https://github.com/koron-go/bloomfilter/workflows/Go/badge.svg)](https://github.com/koron-go/bloomfilter/actions?query=workflow%3AGo)
[![Go Report Card](https://goreportcard.com/badge/github.com/koron-go/bloomfilter)](https://goreportcard.com/report/github.com/koron-go/bloomfilter)

## Volatile Bloom Filter

Volatile Bloom Filter(以下VBF)は通常のBloom Filterにはない、疑似的な削除・忘却
機能を追加したBloom Filterです。

通常のブルームフィルターはキーの有無という情報を実質boolean値として保存します。
一方でVBFはキーに世代という整数値をキーに紐づけて保存します。検査時にはキーの有
無だけにとどまらずキーに紐づいた世代が有効な範囲に収まっているかどうかを検査で
きます。また指定された世代よりも古いキーを削除することで疑似的な削除・忘却機能
を提供します。

### VBFの使い方

1. `NewVBF` で作る
2. `Put(d []byte)` でキーを登録する
3. `Check(d []byte, margin uint8) bool` で検査する
4. `Curr`, `Max`, `SetCurr` で世代を更新する

**現在はまだ古いキーを削除する機能は存在しません**

## Benchmark result

### 2021/04/07

```console
$ go test -bench . -benchmem
goos: windows
goarch: amd64
pkg: github.com/koron-go/bloomfilter
cpu: Intel(R) Core(TM) i9-9900K CPU @ 3.60GHz
BenchmarkVF2_Put_Nbits1-16               7561330               218.2 ns/op             7 B/op          0 allocs/op
BenchmarkVF2_Put_Nbits2-16               7871154               339.5 ns/op             7 B/op          0 allocs/op
BenchmarkVF2_Put_Nbits4-16               7502404               415.7 ns/op             7 B/op          0 allocs/op
BenchmarkVF2_Put_Nbits8-16               8940404               162.8 ns/op             7 B/op          0 allocs/op
BenchmarkVF2_Check_Nbits1-16            10458026               146.8 ns/op             8 B/op          0 allocs/op
BenchmarkVF2_Check_Nbits2-16             9936044               193.7 ns/op             7 B/op          0 allocs/op
BenchmarkVF2_Check_Nbits4-16            10686909               223.1 ns/op             8 B/op          0 allocs/op
BenchmarkVF2_Check_Nbits8-16             8825073               232.3 ns/op             7 B/op          0 allocs/op
BenchmarkVF2_K1-16                      21713011               108.2 ns/op             7 B/op          0 allocs/op
BenchmarkVF2_K2-16                      16090125               125.9 ns/op             7 B/op          0 allocs/op
BenchmarkVF2_K3-16                      13338120               156.2 ns/op             7 B/op          0 allocs/op
BenchmarkVF2_K4-16                      11460726               173.8 ns/op             7 B/op          0 allocs/op
BenchmarkVF2_K5-16                       9920084               185.6 ns/op             7 B/op          0 allocs/op
BenchmarkVF2_K6-16                       8568252               193.3 ns/op             7 B/op          0 allocs/op
BenchmarkVF2_K7-16                       7577766               195.5 ns/op             7 B/op          0 allocs/op
PASS
ok      github.com/koron-go/bloomfilter 45.082s
```

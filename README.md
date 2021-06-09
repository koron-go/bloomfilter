# koron-go/bloomfilter

[![PkgGoDev](https://pkg.go.dev/badge/github.com/koron-go/bloomfilter)](https://pkg.go.dev/github.com/koron-go/bloomfilter)
[![GoDoc](https://godoc.org/github.com/koron-go/bloomfilter?status.svg)](https://godoc.org/github.com/koron-go/bloomfilter)
[![Actions/Go](https://github.com/koron-go/bloomfilter/workflows/Go/badge.svg)](https://github.com/koron-go/bloomfilter/actions?query=workflow%3AGo)
[![Go Report Card](https://goreportcard.com/badge/github.com/koron-go/bloomfilter)](https://goreportcard.com/report/github.com/koron-go/bloomfilter)

## Volatile Bloom Filter

Volatile Bloom Filter(以下VBF)は通常のBloom Filterにはない、疑似的な削除・忘却
機能を追加したBloom Filterです。

通常のブルームフィルターのデータの有無という情報を実質boolean値として保存しま
す。一方でVBFはデータに世代という整数値を紐づけて保存します。検査時にはデータの
有無だけにとどまらずデータに紐づいた世代が有効な範囲に収まっているかどうかを検
査できます。また指定された世代よりも古いデータを削除することで疑似的な削除・忘
却機能を提供します。

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

## VBF2の実装詳細

ブルームフィルターではデータを複数の性質の異なる関数で整数値に射影し複数のイン
デックスを得ます。そうして求めた複数のインデックスをビット配列のインデックスと
して該当する複数のビットを `1` にします。特定のデータに対して求められる全部のイ
ンデックスについて、ビット配列の対応するビットが1であればそのデータが集合に含ま
れるという判定になります。

参照: [ブルームフィルタ- Wikipedia](https://ja.wikipedia.org/wiki/%E3%83%96%E3%83%AB%E3%83%BC%E3%83%A0%E3%83%95%E3%82%A3%E3%83%AB%E3%82%BF)

VBF2の実装ではビット配列の代わりに非負の整数の配列を用いています。整数値のサイ
ズは1, 2, 4, 8ビットから選択できます。なお整数値のサイズが1ビットの際は通常のブ
ルームフィルターとまったく同じ動作およびメモリ効率になります。

```go
// 1 bit の作成例
vbf1 := bloomfilter.NewVBF2(m, k, 1)

// 2 bits の作成例
vbf2 := bloomfilter.NewVBF2(m, k, 2)

// 4 bits の作成例
vbf4 := bloomfilter.NewVBF2(m, k, 4)

// 8 bits の作成例
vbf8 := bloomfilter.NewVBF2(m, k, 8)

// m と k はそれぞれ配列の長さと、1データあたりのインデックスの個数
```

8ビット未満のサイズについては複数の値を1つにまとめてバイト配列として管理してい
ます。すなわち1ビットなら8個ずつ、2ビットなら4個ずつ、4ビットなら2個ずつを束ね
てバイト配列として格納しています。そのため8ビット未満のサイズを選択した場合は、
8ビットを選択した際にくらべて各オペレーションに若干のオーバーヘッドを伴います。

```
1-bit size:

    +-+-+-+-+-+-+-+-+
    |0|0|0|1|1|0|1|1| ...
    +-+-+-+-+-+-+-+-+

2-bits size:

    +---+---+---+---+
    |0 0|0 1|1 0|1 1| ...
    +---+---+---+---+

4-bits size:

    +-------+-------+
    |0 0 0 1|1 0 1 1| ...
    +-------+-------+

8-bits size:

    +---------------+
    |0 0 0 1 1 0 1 1| ...
    +---------------+
```

整数配列にはデータの寿命を格納しています。新しいデータを格納する際には、記録す
る値はその整数値のサイズで表現できる最大の数を格納します。つまり1ビットなら
`1` 、2ビットなら `3` 、4ビットなら `15` 、8ビットなら `255` です。

```go
// データを格納する例
vbf8.Put([]byte("data1"))

vbf8.Put([]byte("data2"))

vbf8.Put([]byte("data3"))

// 各データに対応する複数のインデックスすべてに255が格納される
```

データがVBF2に含まれるかどうかの「検査」はデータから求められた複数のインデック
スのに紐づいた非負整数値=寿命の全てが0かどうかで判定します。1つでも `0` が含ま
れていたらそのデータは存在しないことになります。

```go
// チェックする例
has := vbf8.Check([]byte("data1"), 0) // `has` should be `true`

notHave := vbf8.Check([]byte("data9"), 0) // `notHave` should be `false`

// Check()の第2引数の0はいまは無視してください
```

格納している寿命はまとめて減算処理できます。減算処理とは指定した値を全ての寿命
から引いて全ての寿命を更新することです。引いたことで `0` 未満になる寿命は `0`
として処理されます。この操作は疑似的な削除操作となります。

```go
// 減算する例: 格納している全データの寿命を0減らす
vbf8.Subtract(10)
```


前の検査時には寿命が尽きたことの基準として `0` を採用していましたが、検査時には
その基準値(`bias`)を指定できます。この基準値以下の寿命の値は尽きたものとして扱
われます。この基準値を指定する機能は、寿命のビット数から決まる最大値よりも小さ
い寿命を用いたい時に使用します。

一例として100世代の寿命を扱いたい場合を考えてみます。この時の寿命のサイズは8
ビットを選択する必要があります。4ビットでは15世代しか管理できず足りないためで
す。そうして基準値=`bias` は最大寿命 `255` から有効としたい世代 `100` を引いた
ものであるため `155` となります。

```go
// 基準値を指定してチェックする例:
has := vbf8.Check([]byte("old_data"), 155)

// 255-155 = 100世代前までを有効とするチェック
```

以上によりVBF2では疑似寿命付きのブルームフィルターを実現しています。

## Redisを使った実装のテスト

```
$ docker run --rm --name vbf-redis -p 6379:6379 -d redis:6.2.3-alpine3.13

$ export REDIS_URL=redis://127.0.0.1:6379/0

$ go test -run Redis -v

$ docker exec -it vbf-redis redis-cli

$ docker stop vbf-redis
```

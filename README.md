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

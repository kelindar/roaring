module github.com/kelindar/roaring-bench

go 1.23.2

require (
	github.com/RoaringBitmap/roaring v1.9.4
	github.com/kelindar/bench v0.2.0
	github.com/kelindar/roaring v0.0.0
)

require (
	github.com/bits-and-blooms/bitset v1.12.0 // indirect
	github.com/kelindar/bitmap v1.5.3 // indirect
	github.com/kelindar/simd v1.1.2 // indirect
	github.com/klauspost/cpuid/v2 v2.2.4 // indirect
	github.com/mschoch/smat v0.2.0 // indirect
	golang.org/x/exp v0.0.0-20190125153040-c74c464bbbf2 // indirect
	golang.org/x/sys v0.0.0-20220704084225-05e143d24a9e // indirect
	gonum.org/v1/gonum v0.8.2 // indirect
)

replace github.com/kelindar/roaring => ../

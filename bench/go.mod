module github.com/kelindar/roaring-bench

go 1.25.0

require (
	github.com/RoaringBitmap/roaring v1.9.4
	github.com/kelindar/bench v0.2.0
	github.com/kelindar/roaring v0.0.0
)

require (
	github.com/bits-and-blooms/bitset v1.24.4 // indirect
	github.com/kelindar/bitmap v1.5.5 // indirect
	github.com/kelindar/simd v1.2.0 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/mschoch/smat v0.2.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	gonum.org/v1/gonum v0.17.0 // indirect
)

replace github.com/kelindar/roaring => ../

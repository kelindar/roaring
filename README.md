<p align="center">
<img width="300" height="100" src=".github/logo.png" border="0" alt="kelindar/roaring">
<br>
<img src="https://img.shields.io/github/go-mod/go-version/kelindar/roaring" alt="Go Version">
<a href="https://pkg.go.dev/github.com/kelindar/roaring"><img src="https://pkg.go.dev/badge/github.com/kelindar/roaring" alt="PkgGoDev"></a>
<a href="https://goreportcard.com/report/github.com/kelindar/roaring"><img src="https://goreportcard.com/badge/github.com/kelindar/roaring" alt="Go Report Card"></a>
<a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License"></a>
<a href="https://coveralls.io/github/kelindar/roaring"><img src="https://coveralls.io/repos/github/kelindar/roaring/badge.svg" alt="Coverage"></a>
</p>

## Roaring: Roaring Bitmap for Go

This library provides a fast, memory-efficient Go implementation of [roaring bitmaps](https://roaringbitmap.org/), a compressed bitmap data structure for sets of 32-bit integers. It is designed for high-throughput analytics, set operations, and efficient serialization. While most of you should probably use [the original, well maintained implementation](https://github.com/RoaringBitmap/roaring), this implementation uses [kelindar/bitmap](https://github.com/kelindar/bitmap) for its dense implementation, and tries to optimize `AND`/`AND NOT`/`OR`,`XOR` operations. 

- **High Performance:** Optimized for fast set operations (AND, OR, XOR, AND NOT) and iteration.
- **Memory Efficient:** Uses containerization and compression for sparse and dense data.
- **Go Idioms:** Clean, concise API with Go-style patterns and minimal dependencies.

**Use When**

- ✅ You need to store and manipulate large sets of 32-bit integers efficiently.
- ✅ You want fast set operations (union, intersection, difference, symmetric difference).
- ✅ You want a dependency-free, pure Go implementation.

**Not For:**

- ❌ If you need a mature, and interoperable implementation.
- ❌ Sets of non-integer or non-uint32 data.

## Quick Start

```go
import "github.com/kelindar/roaring"

func main() {
    // Create a new bitmap
    bm := roaring.New()

    // Add values
    bm.Set(1)
    bm.Set(42)
    bm.Set(100000)

    // Check membership
    if bm.Contains(42) {
        // Do something
    }

    // Remove a value
    bm.Remove(1)

    // Count values
    fmt.Println("Count:", bm.Count())

    // Iterate values
    bm.Range(func(x uint32) {
        fmt.Println(x)
    })

    // Set operations
    bm2 := roaring.New()
    bm2.Set(42)
    bm2.Set(7)
    bm.Or(bm2) // Union

    // Serialization
    data := bm.ToBytes()
    bm3 := roaring.FromBytes(data)
    fmt.Println(bm3.Contains(42)) // true
}
```

## API Highlights

- `Set(x uint32)`: Add a value.
- `Remove(x uint32)`: Remove a value.
- `Contains(x uint32) bool`: Check if a value is present.
- `Count() int`: Number of values in the bitmap.
- `Range(func(x uint32))`: Iterate all values.
- `And`, `Or`, `Xor`, `AndNot`: Set operations.
- `ToBytes`, `FromBytes`, `WriteTo`, `ReadFrom`: Serialization.


## Benchmarks

```go
name                 time/op      ops/s        allocs/op    vs ref
-------------------- ------------ ------------ ------------ ------------------
set 1K (seq)         22.8 ns      43.8M        0            ❌ -5% (p=0.000)
set 1K (rnd)         30.9 ns      32.4M        0            ✅ +6% (p=0.000)
set 1K (sps)         19.0 ns      52.7M        0            ✅ +17% (p=0.000)
set 1K (dns)         19.9 ns      50.2M        0            ✅ +14% (p=0.000)
set 1M (seq)         12.5 ns      80.3M        0            ✅ +5% (p=0.000)
set 1M (rnd)         19.6 ns      51.0M        0            ✅ +3% (p=0.000)
set 1M (sps)         32.7 ns      30.6M        0            ✅ +26% (p=0.000)
set 1M (dns)         12.0 ns      83.1M        0            ✅ +12% (p=0.000)
has 1K (seq)         17.6 ns      56.7M        0            ✅ +13% (p=0.000)
has 1K (rnd)         22.4 ns      44.6M        0            ✅ +28% (p=0.000)
has 1K (sps)         12.9 ns      77.2M        0            ✅ +55% (p=0.000)
has 1K (dns)         15.7 ns      63.5M        0            ✅ +35% (p=0.000)
has 1M (seq)         9.1 ns       110.2M       0            ✅ +10% (p=0.000)
has 1M (rnd)         19.0 ns      52.7M        0            🟰 similar
has 1M (sps)         28.8 ns      34.8M        0            ✅ +23% (p=0.000)
has 1M (dns)         11.2 ns      89.6M        0            ❌ -1% (p=0.000)
del 1K (seq)         6.7 ns       149.1M       0            ❌ -6% (p=0.000)
del 1K (rnd)         6.4 ns       155.1M       0            ❌ -3% (p=0.000)
del 1K (sps)         6.4 ns       155.9M       0            ❌ -2% (p=0.000)
del 1K (dns)         6.4 ns       156.6M       0            ❌ -2% (p=0.000)
del 1M (seq)         6.5 ns       153.8M       0            🟰 similar
del 1M (rnd)         19.7 ns      50.7M        0            ✅ +10% (p=0.000)
del 1M (sps)         9.3 ns       107.6M       0            🟰 +49% (p=0.001)
del 1M (dns)         38.8 ns      25.8M        0            ✅ +20% (p=0.000)
and 1K (seq)         785.8 ns     1.3M         4            ✅ +82% (p=0.000)
and 1K (rnd)         582.4 ns     1.7M         4            🟰 similar
and 1K (sps)         1.2 µs       854.6K       19           ✅ +14% (p=0.000)
and 1K (dns)         176.3 ns     5.7M         4            ❌ -22% (p=0.000)
and 1M (seq)         24.3 µs      41.2K        19           ✅ +24% (p=0.000)
and 1M (rnd)         24.5 µs      40.8K        19           ✅ +25% (p=0.000)
and 1M (sps)         3.7 ms       268          15.3K        ❌ -4% (p=0.000)
and 1M (dns)         2.8 µs       354.0K       5            ✅ +128% (p=0.000)
or 1K (seq)          1.8 µs       548.9K       16           ❌ -15% (p=0.000)
or 1K (rnd)          1.6 µs       631.3K       16           ❌ -35% (p=0.000)
or 1K (sps)          1.9 µs       519.8K       32           ✅ +13% (p=0.000)
or 1K (dns)          354.4 ns     2.8M         12           ✅ +14% (p=0.000)
or 1M (seq)          26.0 µs      38.5K        27           ✅ +13% (p=0.000)
or 1M (rnd)          26.3 µs      38.0K        27           ✅ +14% (p=0.000)
or 1M (sps)          4.1 ms       245          15.3K        ✅ +6% (p=0.000)
or 1M (dns)          2.9 µs       346.2K       8            ✅ +24025% (p=0.000)
xor 1K (seq)         1.3 µs       783.4K       14           ✅ +315% (p=0.000)
xor 1K (rnd)         942.1 ns     1.1M         13           ❌ -13% (p=0.000)
xor 1K (sps)         1.8 µs       550.3K       32           ✅ +17% (p=0.000)
xor 1K (dns)         169.3 ns     5.9M         4            ✅ +1805% (p=0.000)
xor 1M (seq)         26.9 µs      37.1K        27           ✅ +86% (p=0.000)
xor 1M (rnd)         26.3 µs      38.1K        27           ✅ +93% (p=0.000)
xor 1M (sps)         3.9 ms       253          15.3K        ✅ +5% (p=0.000)
xor 1M (dns)         2.9 µs       348.3K       8            ✅ +628% (p=0.000)
andnot 1K (seq)      969.3 ns     1.0M         4            ✅ +1% (p=0.000)
andnot 1K (rnd)      717.6 ns     1.4M         4            ✅ +4% (p=0.000)
andnot 1K (sps)      1.4 µs       714.7K       19           ✅ +11% (p=0.000)
andnot 1K (dns)      166.5 ns     6.0M         4            ✅ +3002% (p=0.000)
andnot 1M (seq)      24.8 µs      40.4K        19           ✅ +29% (p=0.000)
andnot 1M (rnd)      24.9 µs      40.1K        19           ✅ +39% (p=0.000)
andnot 1M (sps)      3.9 ms       258          15.3K        ❌ -6% (p=0.000)
andnot 1M (dns)      2.9 µs       346.1K       5            ✅ +863% (p=0.000)
range 1K (seq)       535.6 ns     1.9M         0            🟰 similar
range 1K (rnd)       421.8 ns     2.4M         0            🟰 similar
range 1K (sps)       581.8 ns     1.7M         0            ✅ +23% (p=0.000)
range 1K (dns)       112.6 ns     8.9M         0            ✅ +1% (p=0.000)
range 1M (seq)       2.4 ms       421          0            ❌ -33% (p=0.000)
range 1M (rnd)       2.2 ms       464          0            ❌ -40% (p=0.000)
range 1M (sps)       579.1 µs     1.7K         0            ✅ +8% (p=0.000)
range 1M (dns)       212.3 µs     4.7K         0            ✅ +49% (p=0.000)
```


## About

Bench is MIT licensed and maintained by [@kelindar](https://github.com/kelindar). PRs and issues welcome! 
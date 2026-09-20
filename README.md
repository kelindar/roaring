<p align="center">
<img width="300" height="100" src=".github/logo.png" border="0" alt="kelindar/roaring">
<br>
<img src="https://img.shields.io/github/go-mod/go-version/kelindar/roaring" alt="Go Version">
<a href="https://pkg.go.dev/github.com/kelindar/roaring"><img src="https://pkg.go.dev/badge/github.com/kelindar/roaring" alt="PkgGoDev"></a>
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
set 1K (seq)         18.5 ns      54.1M        0             ✅ +7%
set 1K (rnd)         27.3 ns      36.7M        0             ✅ +8%
set 1K (sps)         19.4 ns      51.4M        0             ✅ +8%
set 1K (dns)         18.5 ns      54.0M        0             ❔ uncertain
set 1M (seq)         11.1 ns      90.3M        0             ✅ +53%
set 1M (rnd)         19.5 ns      51.4M        0             ✅ +10%
set 1M (sps)         33.3 ns      30.0M        0             ✅ +15%
set 1M (dns)         12.4 ns      80.5M        0             ✅ +9%
has 1K (seq)         14.7 ns      68.1M        0             ✅ +17%
has 1K (rnd)         19.3 ns      51.8M        0             ✅ +22%
has 1K (sps)         14.9 ns      67.0M        0             ✅ +14%
has 1K (dns)         13.8 ns      72.5M        0             ✅ +27%
has 1M (seq)         10.3 ns      96.7M        0             ❔ uncertain
has 1M (rnd)         19.2 ns      52.2M        0             ❔ uncertain
has 1M (sps)         27.5 ns      36.3M        0             ✅ +25%
has 1M (dns)         11.6 ns      86.0M        0             🟰 similar
del 1K (seq)         6.6 ns       152.0M       0             ✅ +11%
del 1K (rnd)         6.6 ns       151.6M       0             ❔ uncertain
del 1K (sps)         6.5 ns       152.9M       0             ✅ +12%
del 1K (dns)         6.6 ns       152.6M       0             ✅ +13%
del 1M (seq)         6.7 ns       150.1M       0             ✅ +11%
del 1M (rnd)         6.7 ns       149.6M       0             ✅ +10%
del 1M (sps)         6.6 ns       150.9M       0             ✅ +11%
del 1M (dns)         6.6 ns       151.2M       0             ✅ +11%
and 1K (seq)         806.8 ns     1.2M         4             ❔ uncertain
and 1K (rnd)         579.9 ns     1.7M         4             ❔ uncertain
and 1K (sps)         1.1 µs       874.3K       4             ❔ uncertain
and 1K (dns)         83.4 ns      12.0M         4             ✅ +55%
and 1M (seq)         23.9 µs      41.8K         4             ✅ +57%
and 1M (rnd)         24.3 µs      41.1K         4             ✅ +51%
and 1M (sps)         1.8 ms       561           4             ✅ +85%
and 1M (dns)         2.8 µs       352.6K        4             ✅ +3.1x
or 1K (seq)          1.4 µs       702.1K        5             ❔ uncertain
or 1K (rnd)          1.1 µs       935.5K        5             ❔ uncertain
or 1K (sps)          1.6 µs       629.3K        5             ✅ +42%
or 1K (dns)          104.1 ns     9.6M          5             ✅ +4.6x
or 1M (seq)          24.9 µs      40.2K         4             ✅ +45%
or 1M (rnd)          25.3 µs      39.5K         4             ✅ +44%
or 1M (sps)          2.0 ms       489           5             ❔ uncertain
or 1M (dns)          2.8 µs       357.4K        4             ✅ +266x
xor 1K (seq)         1.2 µs       825.1K        5             ✅ +5.5x
xor 1K (rnd)         971.9 ns     1.0M          5             ❔ uncertain
xor 1K (sps)         1.5 µs       668.1K        5             ✅ +26%
xor 1K (dns)         107.1 ns     9.3M          5             ✅ +38x
xor 1M (seq)         26.2 µs      38.2K         4             ✅ +47%
xor 1M (rnd)         26.5 µs      37.8K         4             ✅ +43%
xor 1M (sps)         2.1 ms       482           5             ✅ +92%
xor 1M (dns)         3.1 µs       320.1K        4             ✅ +8.3x
andnot 1K (seq)      784.3 ns     1.3M          4             ✅ +16%
andnot 1K (rnd)      644.8 ns     1.6M          4             ❔ uncertain
andnot 1K (sps)      1.2 µs       841.3K        4             ❔ uncertain
andnot 1K (dns)      86.4 ns      11.6M         4             ✅ +33%
andnot 1M (seq)      25.2 µs      39.8K         4             ✅ +54%
andnot 1M (rnd)      25.0 µs      40.0K         4             ✅ +71%
andnot 1M (sps)      1.8 ms       552           4             ✅ +86%
andnot 1M (dns)      3.2 µs       310.4K        4             ✅ +21x
range 1K (seq)       538.3 ns     1.9M          0             ❔ uncertain
range 1K (rnd)       408.9 ns     2.4M          0             ❌ -7%
range 1K (sps)       583.4 ns     1.7M          0             ❔ uncertain
range 1K (dns)       116.3 ns     8.6M          0             ❔ uncertain
range 1M (seq)       552.0 µs     1.8K          0             ✅ +2.7x
range 1M (rnd)       452.2 µs     2.2K          0             ✅ +2.6x
range 1M (sps)       574.2 µs     1.7K          0             ✅ +10%
range 1M (dns)       98.8 µs      10.1K         0             ✅ +2.9x
```


## About

Bench is MIT licensed and maintained by [@kelindar](https://github.com/kelindar). PRs and issues welcome!

<p align="center">
<img width="330" height="110" src=".github/logo.png" border="0" alt="kelindar/roaring">
<br>
<img src="https://img.shields.io/github/go-mod/go-version/kelindar/roaring" alt="Go Version">
<a href="https://pkg.go.dev/github.com/kelindar/roaring"><img src="https://pkg.go.dev/badge/github.com/kelindar/roaring" alt="PkgGoDev"></a>
<a href="https://goreportcard.com/report/github.com/kelindar/roaring"><img src="https://goreportcard.com/badge/github.com/kelindar/roaring" alt="Go Report Card"></a>
<a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License"></a>
<a href="https://coveralls.io/github/kelindar/roaring"><img src="https://coveralls.io/repos/github/kelindar/roaring/badge.svg" alt="Coverage"></a>
</p>

**High-Performance Roaring Bitmaps for Go**

This library provides a fast, memory-efficient, and idiomatic Go implementation of [Roaring Bitmaps](https://roaringbitmap.org/), a compressed bitmap data structure for sets of 32-bit integers. It is designed for high-throughput analytics, set operations, and efficient serialization. While most of you should probably use [the original, well maintained implementation](https://github.com/RoaringBitmap/roaring), this implementation uses [https://github.com/kelindar/bitmap](kelindar/bitmap) for its dense implementation, and tries to optimize `AND`/`AND NOT`/`OR`,`XOR` operations. 

- **High Performance:** Optimized for fast set operations (AND, OR, XOR, AND NOT) and iteration.
- **Memory Efficient:** Uses containerization and compression for sparse and dense data.
- **Go Idioms:** Clean, concise API with Go-style patterns and minimal dependencies.
- **Serialization:** Efficient streaming and byte-level serialization/deserialization.

## Use When

- ✅ You need to store and manipulate large sets of 32-bit integers efficiently.
- ✅ You want fast set operations (union, intersection, difference, symmetric difference).
- ✅ You need to serialize/deserialize bitmaps for storage or network transfer.
- ✅ You want a dependency-free, pure Go implementation.

**Not For:**

- ❌ Sets of non-integer or non-uint32 data.
- ❌ Use cases requiring persistent storage (use a DB or file format for that).

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

name                 time/op      ops/s        allocs/op    vs original
-------------------- ------------ ------------ ------------ ------------------
set 1K (seq)         42.5 ns      23.5M        0            ~ similar
set 1K (rnd)         40.4 ns      24.7M        0            ~ similar
set 1K (sps)         37.8 ns      26.5M        0            ✅ +4% (p=0.000)
set 1K (dns)         31.3 ns      31.9M        0            ✅ +2% (p=0.000)
set 1M (seq)         39.9 ns      25.1M        0            ❌ -15% (p=0.000)
set 1M (rnd)         32.8 ns      30.5M        0            ~ +2% (p=0.359)
set 1M (sps)         120.6 ns     8.3M         0            ~ +3% (p=0.012)
set 1M (dns)         18.7 ns      53.4M        0            ~ +4% (p=0.011)
has 1K (seq)         37.3 ns      26.8M        0            ✅ +5% (p=0.000)
has 1K (rnd)         36.9 ns      27.1M        0            ~ +2% (p=0.002)
has 1K (sps)         33.5 ns      29.8M        0            ✅ +10% (p=0.000)
has 1K (dns)         30.2 ns      33.2M        0            ✅ +1% (p=0.000)
has 1M (seq)         26.7 ns      37.5M        0            ✅ +3% (p=0.000)
has 1M (rnd)         26.7 ns      37.4M        0            ✅ +3% (p=0.000)
has 1M (sps)         101.3 ns     9.9M         0            ✅ +11% (p=0.000)
has 1M (dns)         17.3 ns      57.8M        0            ~ similar
del 1K (seq)         9.0 ns       111.1M       0            ❌ -3% (p=0.000)
del 1K (rnd)         8.9 ns       112.5M       0            ❌ -2% (p=0.000)
del 1K (sps)         8.9 ns       111.9M       0            ❌ -3% (p=0.000)
del 1K (dns)         8.9 ns       112.0M       0            ❌ -3% (p=0.000)
del 1M (seq)         14.6 ns      68.4M        0            ~ +14% (p=0.122)
del 1M (rnd)         13.8 ns      72.3M        0            ~ +19% (p=0.033)
del 1M (sps)         26.8 ns      37.3M        0            ~ +9% (p=0.546)
del 1M (dns)         11.0 ns      91.1M        0            ~ similar
and 1K (seq)         782.3 ns     1.3M         4            ✅ +321% (p=0.000)
and 1K (rnd)         573.2 ns     1.7M         4            ✅ +23% (p=0.000)
and 1K (sps)         2.2 µs       453.8K       19           ✅ +18% (p=0.000)
and 1K (dns)         323.4 ns     3.1M         4            ~ -10% (p=0.002)
and 1M (seq)         46.7 µs      21.4K        19           ✅ +35% (p=0.000)
and 1M (rnd)         48.7 µs      20.5K        19           ✅ +33% (p=0.000)
and 1M (sps)         5.5 ms       182          15.3K        ~ +3% (p=0.077)
and 1M (dns)         6.4 µs       155.7K       5            ✅ +135% (p=0.000)
or 1K (seq)          3.0 µs       337.9K       15           ✅ +413% (p=0.000)
or 1K (rnd)          2.6 µs       386.9K       16           ❌ -16% (p=0.000)
or 1K (sps)          3.9 µs       253.9K       32           ~ +8% (p=0.019)
or 1K (dns)          684.9 ns     1.5M         12           ✅ +28% (p=0.000)
or 1M (seq)          47.8 µs      20.9K        27           ✅ +20% (p=0.000)
or 1M (rnd)          47.9 µs      20.9K        27           ✅ +17% (p=0.000)
or 1M (sps)          7.1 ms       141          15.3K        ✅ +8% (p=0.000)
or 1M (dns)          6.0 µs       168.0K       8            ✅ +23919% (p=0.000)
xor 1K (seq)         2.1 µs       468.4K       14           ~ +5% (p=0.125)
xor 1K (rnd)         1.9 µs       523.2K       14           ❌ -10% (p=0.000)
xor 1K (sps)         3.6 µs       275.9K       32           ✅ +13% (p=0.000)
xor 1K (dns)         426.2 ns     2.3M         7            ✅ +1420% (p=0.000)
xor 1M (seq)         46.1 µs      21.7K        27           ✅ +115% (p=0.000)
xor 1M (rnd)         47.4 µs      21.1K        27           ✅ +112% (p=0.000)
xor 1M (sps)         7.0 ms       144          15.3K        ~ +4% (p=0.005)
xor 1M (dns)         6.0 µs       165.4K       8            ✅ +699% (p=0.000)
andnot 1K (seq)      1.5 µs       658.4K       4            ~ +2% (p=0.163)
andnot 1K (rnd)      1.2 µs       851.3K       4            ✅ +10% (p=0.000)
andnot 1K (sps)      2.3 µs       427.8K       19           ✅ +16% (p=0.000)
andnot 1K (dns)      344.5 ns     2.9M         5            ✅ +2044% (p=0.000)
andnot 1M (seq)      45.1 µs      22.2K        19           ✅ +40% (p=0.000)
andnot 1M (rnd)      145.1 µs     6.9K         19           ❌ -50% (p=0.000)
andnot 1M (sps)      5.5 ms       182          15.3K        ~ +2% (p=0.073)
andnot 1M (dns)      5.9 µs       170.2K       5            ✅ +836% (p=0.000)
range 1K (seq)       762.2 ns     1.3M         0            ~ similar
range 1K (rnd)       680.1 ns     1.5M         0            ~ similar
range 1K (sps)       910.3 ns     1.1M         0            ~ +8% (p=0.025)
range 1K (dns)       160.8 ns     6.2M         0            ~ +7% (p=0.085)
range 1M (seq)       3.2 ms       315          0            ❌ -43% (p=0.000)
range 1M (rnd)       2.9 ms       349          0            ❌ -46% (p=0.000)
range 1M (sps)       892.3 µs     1.1K         0            ✅ +15% (p=0.000)
range 1M (dns)       249.4 µs     4.0K         0            ✅ +44% (p=0.000)
```


## About

Bench is MIT licensed and maintained by [@kelindar](https://github.com/kelindar). PRs and issues welcome! 
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
set 1K (seq)         21.6 ns      46.2M        0             🟰 similar
set 1K (rnd)         29.7 ns      33.7M        0             ✅ +12%
set 1K (sps)         17.8 ns      56.1M        0             ✅ +29%
set 1K (dns)         19.0 ns      52.6M        0             ✅ +16%
set 1M (seq)         12.0 ns      83.1M        0             ✅ +20%
set 1M (rnd)         18.6 ns      53.8M        0             🟰 similar
set 1M (sps)         31.8 ns      31.5M        0             ✅ +27%
set 1M (dns)         12.0 ns      83.4M        0             🟰 similar
has 1K (seq)         17.6 ns      57.0M        0             ✅ +12%
has 1K (rnd)         21.9 ns      45.7M        0             ✅ +25%
has 1K (sps)         12.7 ns      78.8M        0             ✅ +57%
has 1K (dns)         14.8 ns      67.4M        0             ✅ +39%
has 1M (seq)         9.6 ns       104.7M       0             🟰 similar
has 1M (rnd)         18.1 ns      55.2M        0             🟰 similar
has 1M (sps)         27.9 ns      35.8M        0             ✅ +24%
has 1M (dns)         10.8 ns      92.2M        0             🟰 similar
del 1K (seq)         6.4 ns       155.2M       0             ❌ -7%
del 1K (rnd)         6.5 ns       154.8M       0             ❌ -7%
del 1K (sps)         6.4 ns       155.7M       0             🟰 similar
del 1K (dns)         6.4 ns       156.5M       0             🟰 similar
del 1M (seq)         6.5 ns       153.6M       0             🟰 similar
del 1M (rnd)         19.3 ns      51.9M        0             ✅ +8%
del 1M (sps)         12.1 ns      82.7M        0             ✅ +89%
del 1M (dns)         37.8 ns      26.4M        0             ✅ +17%
and 1K (seq)         802.5 ns     1.2M         4             🟰 similar
and 1K (rnd)         600.8 ns     1.7M         4             🟰 similar
and 1K (sps)         1.1 µs       886.6K       19            ✅ +14%
and 1K (dns)         172.9 ns     5.8M         4             ❌ -23%
and 1M (seq)         24.5 µs      40.8K        19            ✅ +24%
and 1M (rnd)         24.5 µs      40.8K        19            ✅ +23%
and 1M (sps)         3.7 ms       272          15.3K         🟰 similar
and 1M (dns)         2.8 µs       361.0K       5             ✅ +2.7x
or 1K (seq)          1.8 µs       544.6K       16            ❌ -14%
or 1K (rnd)          1.6 µs       630.1K       16            ❌ -32%
or 1K (sps)          2.0 µs       503.5K       32            ✅ +14%
or 1K (dns)          354.3 ns     2.8M         12            ✅ +19%
or 1M (seq)          25.6 µs      39.1K        27            ✅ +14%
or 1M (rnd)          25.4 µs      39.3K        27            ✅ +18%
or 1M (sps)          4.0 ms       250          15.3K         🟰 similar
or 1M (dns)          2.8 µs       351.9K       8             ✅ +248x
xor 1K (seq)         1.2 µs       811.6K       14            🟰 similar
xor 1K (rnd)         1.1 µs       917.8K       14            ❌ -17%
xor 1K (sps)         1.8 µs       556.7K       32            ✅ +19%
xor 1K (dns)         210.2 ns     4.8M         7             ✅ +15x
xor 1M (seq)         25.4 µs      39.4K        27            ✅ +93%
xor 1M (rnd)         25.3 µs      39.5K        27            ✅ +97%
xor 1M (sps)         3.9 ms       257          15.3K         ✅ +9%
xor 1M (dns)         2.9 µs       345.8K       8             ✅ +7.1x
andnot 1K (seq)      976.8 ns     1.0M         4             ❌ -7%
andnot 1K (rnd)      685.7 ns     1.5M         4             ❌ -7%
andnot 1K (sps)      1.4 µs       727.7K       19            🟰 similar
andnot 1K (dns)      165.1 ns     6.1M         4             ✅ +30x
andnot 1M (seq)      24.3 µs      41.1K        19            ✅ +30%
andnot 1M (rnd)      24.4 µs      41.0K        19            ✅ +44%
andnot 1M (sps)      3.7 ms       269          15.3K         🟰 similar
andnot 1M (dns)      2.8 µs       362.7K       5             ✅ +9.6x
range 1K (seq)       485.7 ns     2.1M         0             🟰 similar
range 1K (rnd)       383.6 ns     2.6M         0             🟰 similar
range 1K (sps)       542.2 ns     1.8M         0             ✅ +25%
range 1K (dns)       106.6 ns     9.4M         0             🟰 similar
range 1M (seq)       2.3 ms       441          0             ❌ -35%
range 1M (rnd)       2.1 ms       482          0             ❌ -43%
range 1M (sps)       558.6 µs     1.8K         0             ✅ +12%
range 1M (dns)       203.9 µs     4.9K         0             ✅ +45%
```


## About

Bench is MIT licensed and maintained by [@kelindar](https:/r). PRs and issues welcome! 
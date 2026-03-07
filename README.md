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
set 1K (seq)         18.9 ns      52.9M        0             🟰 similar
set 1K (rnd)         30.4 ns      32.9M        0             🟰 similar
set 1K (sps)         20.0 ns      49.9M        0             🟰 similar
set 1K (dns)         20.7 ns      48.4M        0             🟰 similar
set 1M (seq)         12.0 ns      83.5M        0             ✅ +12%
set 1M (rnd)         19.4 ns      51.6M        0             ✅ +7%
set 1M (sps)         30.1 ns      33.2M        0             ✅ +31%
set 1M (dns)         13.1 ns      76.2M        0             ✅ +9%
has 1K (seq)         17.4 ns      57.5M        0             🟰 similar
has 1K (rnd)         25.6 ns      39.0M        0             🟰 similar
has 1K (sps)         13.3 ns      75.5M        0             ✅ +35%
has 1K (dns)         20.0 ns      50.0M        0             🟰 similar
has 1M (seq)         8.7 ns       115.2M       0             ✅ +22%
has 1M (rnd)         18.3 ns      54.7M        0             🟰 similar
has 1M (sps)         26.4 ns      37.9M        0             ✅ +31%
has 1M (dns)         12.2 ns      82.2M        0             🟰 similar
del 1K (seq)         6.8 ns       146.6M       0             ❌ -13%
del 1K (rnd)         6.8 ns       146.8M       0             ❌ -10%
del 1K (sps)         6.7 ns       148.2M       0             ❌ -9%
del 1K (dns)         6.8 ns       147.8M       0             ❌ -10%
del 1M (seq)         6.8 ns       146.4M       0             🟰 similar
del 1M (rnd)         20.4 ns      49.1M        0             ✅ +12%
del 1M (sps)         12.1 ns      82.6M        0             ✅ +86%
del 1M (dns)         38.7 ns      25.8M        0             ✅ +34%
and 1K (seq)         888.3 ns     1.1M         4             ❌ -7%
and 1K (rnd)         633.3 ns     1.6M         4             🟰 similar
and 1K (sps)         1.4 µs       710.9K       19            🟰 similar
and 1K (dns)         181.9 ns     5.5M         4             ❌ -10%
and 1M (seq)         24.6 µs      40.7K        19            ✅ +28%
and 1M (rnd)         23.5 µs      42.5K        19            ✅ +27%
and 1M (sps)         3.8 ms       265          15.3K         🟰 similar
and 1M (dns)         3.8 µs       261.8K       5             ✅ +2.2x
or 1K (seq)          1.7 µs       589.4K       16            ❌ -10%
or 1K (rnd)          1.5 µs       661.2K       16            ❌ -30%
or 1K (sps)          1.8 µs       552.7K       32            ✅ +16%
or 1K (dns)          345.9 ns     2.9M         12            ✅ +17%
or 1M (seq)          25.7 µs      39.0K        27            ✅ +17%
or 1M (rnd)          24.0 µs      41.7K        27            ✅ +17%
or 1M (sps)          3.9 ms       258          15.3K         ✅ +9%
or 1M (dns)          2.9 µs       344.9K       8             ✅ +273x
xor 1K (seq)         1.6 µs       613.1K       16            ❌ -37%
xor 1K (rnd)         1.1 µs       947.1K       14            ❌ -25%
xor 1K (sps)         1.8 µs       566.2K       32            ✅ +8%
xor 1K (dns)         219.6 ns     4.6M         7             ✅ +14x
xor 1M (seq)         24.6 µs      40.7K        27            ✅ +97%
xor 1M (rnd)         26.5 µs      37.7K        27            ✅ +92%
xor 1M (sps)         4.0 ms       251          15.3K         🟰 similar
xor 1M (dns)         2.9 µs       339.7K       8             ✅ +9.1x
andnot 1K (seq)      898.5 ns     1.1M         4             🟰 similar
andnot 1K (rnd)      697.9 ns     1.4M         4             🟰 similar
andnot 1K (sps)      1.4 µs       727.6K       19            ✅ +14%
andnot 1K (dns)      161.0 ns     6.2M         4             ✅ +30x
andnot 1M (seq)      23.8 µs      42.0K        19            ✅ +32%
andnot 1M (rnd)      85.3 µs      11.7K        19            ❌ -61%
andnot 1M (sps)      3.7 ms       270          15.3K         🟰 similar
andnot 1M (dns)      2.8 µs       354.3K       5             ✅ +10.1x
range 1K (seq)       501.4 ns     2.0M         0             🟰 similar
range 1K (rnd)       429.1 ns     2.3M         0             🟰 similar
range 1K (sps)       583.6 ns     1.7M         0             ✅ +22%
range 1K (dns)       114.5 ns     8.7M         0             🟰 similar
range 1M (seq)       2.4 ms       413          0             ❌ -37%
range 1M (rnd)       2.2 ms       461          0             ❌ -41%
range 1M (sps)       554.2 µs     1.8K         0             ✅ +12%
range 1M (dns)       218.2 µs     4.6K         0             ✅ +39%
```


## About

Bench is MIT licensed and maintained by [@kelindar](https://github.com/kelindar). PRs and issues welcome! 
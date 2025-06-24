# Roaring Bitmap Benchmarks

This repository includes comprehensive benchmarks for the core Roaring Bitmap operations: **Set**, **Remove**, and **Contains**. These benchmarks include direct comparisons with the reference @RoaringBitmap/roaring implementation.

## Running Benchmarks

### Basic Usage

Run all benchmarks:
```bash
go test -bench=.
```

Run benchmarks with memory allocation stats:
```bash
go test -bench=. -benchmem
```

Run specific operation benchmarks:
```bash
# Set operations only
go test -bench=BenchmarkSet

# Remove operations only  
go test -bench=BenchmarkRemove

# Contains operations only
go test -bench=BenchmarkContains
```

### Comparison Benchmarks

Run direct comparisons with @RoaringBitmap/roaring:
```bash
# Run only comparison benchmarks
go test -bench=BenchmarkComparison

# Run comparison benchmarks with memory stats
go test -bench=BenchmarkComparison -benchmem

# Compare specific operations
go test -bench=BenchmarkComparisonSet      # Set operation comparisons
go test -bench=BenchmarkComparisonContains # Contains operation comparisons  
go test -bench=BenchmarkComparisonRemove   # Remove operation comparisons
```

### Advanced Options

Control benchmark duration:
```bash
# Run each benchmark for 5 seconds
go test -bench=. -benchtime=5s

# Run each benchmark for 1000 iterations
go test -bench=. -benchtime=1000x
```

Run benchmarks multiple times for statistical accuracy:
```bash
go test -bench=. -count=5
```

Save benchmark results for comparison:
```bash
go test -bench=. -benchmem > benchmark_results.txt
```

## Benchmark Categories

### Data Patterns

The benchmarks test different data patterns that affect Roaring Bitmap performance:

1. **Sequential**: Consecutive integers (e.g., 1, 2, 3, 4, ...)
   - Optimizes to run containers
   - Best case performance for most operations

2. **Random**: Randomly distributed integers
   - Mixed container types
   - Representative of real-world sparse data

3. **Sparse**: Large gaps between values (e.g., 0, 1000, 2000, ...)
   - Tests array container performance
   - Minimal memory usage

4. **Dense**: High probability of duplicates in small range
   - Tests bitmap container performance
   - High cardinality in small space

5. **Container Boundary**: Values crossing 16-bit container boundaries
   - Tests multi-container operations
   - Important for large datasets

### Dataset Sizes

- **Small**: 1,000 elements
- **Medium**: 10,000 elements  
- **Large**: 100,000 elements
- **XLarge**: 1,000,000 elements (used in some benchmarks)

### Operation Types

#### Set Operations
- `BenchmarkSetSequential*`: Adding consecutive values
- `BenchmarkSetRandom*`: Adding random values
- `BenchmarkSetSparse*`: Adding sparse values
- `BenchmarkSetDense*`: Adding dense values
- `BenchmarkSingleSet*`: Individual set operations

#### Remove Operations
- `BenchmarkRemoveSequential*`: Removing consecutive values
- `BenchmarkRemoveRandom*`: Removing random values
- `BenchmarkRemoveSparse*`: Removing sparse values
- `BenchmarkRemoveDense*`: Removing dense values
- `BenchmarkSingleRemove`: Individual remove operations

#### Contains Operations
- `BenchmarkContainsSequential*`: Checking consecutive values
- `BenchmarkContainsRandom*`: Checking random values
- `BenchmarkContainsSparse*`: Checking sparse values
- `BenchmarkContainsDense*`: Checking dense values
- `BenchmarkSingleContains*`: Individual contains operations

#### Mixed Operations
- `BenchmarkMixedOperations*`: Combined Set/Contains/Remove operations

#### Comparison Benchmarks
- `BenchmarkComparison*_This`: This implementation's performance
- `BenchmarkComparison*_Reference`: @RoaringBitmap/roaring reference implementation
- Direct side-by-side comparison using identical data patterns and sizes

## Interpreting Results

### Performance Metrics

Benchmark output format:
```
BenchmarkSetSequentialSmall-2    6812    33443 ns/op    3368 B/op    12 allocs/op
```

- `6812`: Number of iterations run
- `33443 ns/op`: Nanoseconds per operation
- `3368 B/op`: Bytes allocated per operation  
- `12 allocs/op`: Number of allocations per operation

### Performance Analysis

**Good Performance Indicators:**
- Low ns/op for operations
- Minimal allocations for Contains operations
- Reasonable memory usage growth with dataset size

**Container Type Performance:**
- **Run containers** (sequential data): Fastest for all operations
- **Array containers** (sparse data): Good for small datasets
- **Bitmap containers** (dense data): Consistent performance regardless of cardinality

### Direct Comparison with @RoaringBitmap/roaring

This benchmark suite includes direct performance comparisons with the reference @RoaringBitmap/roaring implementation using identical data patterns and test conditions.

**Run comparison benchmarks:**
```bash
# All comparison benchmarks
go test -bench=BenchmarkComparison -benchmem

# Compare and save results for analysis
go test -bench=BenchmarkComparison -benchmem > comparison_results.txt
```

**Analyze results:**
Look for benchmark pairs like:
```
BenchmarkComparisonSetSequentialSmall_This-2        5000    250000 ns/op    8000 B/op    20 allocs/op
BenchmarkComparisonSetSequentialSmall_Reference-2   4500    280000 ns/op    9000 B/op    25 allocs/op
```

**Key comparison metrics:**
- **Performance ratio**: Compare ns/op between implementations
- **Memory efficiency**: Compare B/op and allocs/op
- **Scalability**: How performance changes with data size
- **Data pattern sensitivity**: Which patterns favor each implementation

### Expected Performance Characteristics

- **Set operations**: Should show good performance for sequential data, moderate for random
- **Contains operations**: Should be very fast with minimal allocations
- **Remove operations**: Performance depends on container optimization after removal

## Example Output

```
BenchmarkSetSequentialSmall-2          6812     33443 ns/op    3368 B/op      12 allocs/op
BenchmarkSetRandomMedium-2              163   1645096 ns/op  733868 B/op     261 allocs/op
BenchmarkContainsSequentialLarge-2      100   2045457 ns/op       0 B/op       0 allocs/op
BenchmarkRemoveRandomSmall-2           2304    100321 ns/op   11632 B/op      19 allocs/op
```

This shows:
- Sequential sets are very efficient
- Random operations require more memory due to container creation
- Contains operations have zero allocations (good!)
- Remove operations may trigger container optimizations
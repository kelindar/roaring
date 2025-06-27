# tinybench

A lightweight, statistical benchmarking library for Go that provides:

- **Statistical analysis** using Welch's t-test for significance testing
- **JSON persistence** with incremental saving for interrupt resilience  
- **Delta comparison** between runs with p-values
- **Reference implementation comparison** 
- **Clean table formatting** with configurable output
- **Filtering** by benchmark name prefix
- **Configurable sampling** parameters

## Quick Start

```go
package main

import "github.com/kelindar/roaring/tinybench"

func main() {
    runner := tinybench.New("results.json").
        Filter("set").  // optional: only run benchmarks starting with "set"
        Start()

    // Simple benchmark
    runner.Run("benchmark name", func() {
        // code to benchmark
    })

    // Benchmark with reference comparison
    runner.Run("benchmark vs ref", 
        func() { /* our implementation */ },
        func() { /* reference implementation */ })

    runner.Finish()
}
```

## API Reference

### Creating a Runner

```go
runner := tinybench.New("results.json")
```

### Configuration (all optional)

```go
runner.Filter("prefix").           // Only run benchmarks matching prefix
       Samples(50).               // Number of samples per benchmark (default: 100)
       Duration(5*time.Millisecond). // Duration per sample (default: 10ms)
       ShowReference(false).      // Hide reference comparison column
       TableFormat("%-25s %-10s %-10s %-15s %-15s\n") // Custom table format
```

### Running Benchmarks

```go
// Start the session (prints table header)
runner.Start()

// Simple benchmark
runner.Run("name", benchmarkFunc)

// Benchmark with reference comparison  
runner.Run("name", ourFunc, refFunc)

// Finish the session
runner.Finish()
```

## Output Format

The library outputs a formatted table with these columns:

- **name**: Benchmark name
- **time/op**: Time per operation (ns or ms)
- **ops/s**: Operations per second (with K/M suffixes)  
- **vs prev**: Comparison with previous run (with p-value)
- **vs ref**: Comparison with reference implementation (with p-value)

Statistical significance indicators:
- ✅ Statistically significant improvement
- ❌ Statistically significant regression  
- ~ No statistically significant difference

## Features

### Incremental Saving
Results are saved after each benchmark completes, so interrupting mid-flight doesn't lose progress.

### Statistical Analysis
Uses Welch's t-test with 99% confidence intervals to determine statistical significance.

### JSON Persistence
Results are stored in a simple JSON format for easy analysis and tracking over time.

### Filtering
Use `-bench prefix` pattern to run only specific benchmarks during development.

## Example Output

```
name                 time/op      ops/s        vs prev            vs ref
-------------------- ------------ ------------ ------------------ ------------------
set (seq)            40.7ns       24.6M        new                ✅ 1.03x (p=0.000)
set (rnd)            38.9ns       25.7M        ✅ 2.47x (p=0.000) ✅ 1.02x (p=0.000)
``` 
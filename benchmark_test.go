package roaring

import (
	"math/rand/v2"
	"testing"
	
	reference "github.com/RoaringBitmap/roaring"
)

// Benchmark data sizes for different test scenarios
const (
	benchmarkSizeSmall  = 1000
	benchmarkSizeMedium = 10000
	benchmarkSizeLarge  = 100000
	benchmarkSizeXLarge = 1000000
)

// generateSequentialData creates consecutive integers starting from offset
func generateSequentialData(size int, offset uint32) []uint32 {
	data := make([]uint32, size)
	for i := 0; i < size; i++ {
		data[i] = offset + uint32(i)
	}
	return data
}

// generateRandomData creates random integers within a range
func generateRandomData(size int, maxVal uint32) []uint32 {
	data := make([]uint32, size)
	for i := 0; i < size; i++ {
		data[i] = uint32(rand.IntN(int(maxVal)))
	}
	return data
}

// generateSparseData creates sparse integers (large gaps between values)
func generateSparseData(size int) []uint32 {
	data := make([]uint32, size)
	for i := 0; i < size; i++ {
		data[i] = uint32(i * 1000) // Large gaps between values
	}
	return data
}

// generateDenseData creates dense integers in a small range
func generateDenseData(size int) []uint32 {
	data := make([]uint32, size)
	for i := 0; i < size; i++ {
		data[i] = uint32(rand.IntN(size/10)) // High probability of duplicates
	}
	return data
}

// generateContainerBoundaryData creates values that cross container boundaries
func generateContainerBoundaryData(size int) []uint32 {
	data := make([]uint32, size)
	for i := 0; i < size; i++ {
		// Mix values around container boundaries (65536 multiples)
		container := uint32(i % 10)
		offset := uint32(rand.IntN(200)) - 100 // -100 to +99 around boundary
		data[i] = container*65536 + 65536/2 + offset
	}
	return data
}

// SET OPERATION BENCHMARKS

func BenchmarkSetSequentialSmall(b *testing.B) {
	data := generateSequentialData(benchmarkSizeSmall, 0)
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		rb := New()
		for _, val := range data {
			rb.Set(val)
		}
	}
}

func BenchmarkSetSequentialMedium(b *testing.B) {
	data := generateSequentialData(benchmarkSizeMedium, 0)
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		rb := New()
		for _, val := range data {
			rb.Set(val)
		}
	}
}

func BenchmarkSetSequentialLarge(b *testing.B) {
	data := generateSequentialData(benchmarkSizeLarge, 0)
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		rb := New()
		for _, val := range data {
			rb.Set(val)
		}
	}
}

func BenchmarkSetRandomSmall(b *testing.B) {
	data := generateRandomData(benchmarkSizeSmall, benchmarkSizeSmall*10)
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		rb := New()
		for _, val := range data {
			rb.Set(val)
		}
	}
}

func BenchmarkSetRandomMedium(b *testing.B) {
	data := generateRandomData(benchmarkSizeMedium, benchmarkSizeMedium*10)
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		rb := New()
		for _, val := range data {
			rb.Set(val)
		}
	}
}

func BenchmarkSetRandomLarge(b *testing.B) {
	data := generateRandomData(benchmarkSizeLarge, benchmarkSizeLarge*10)
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		rb := New()
		for _, val := range data {
			rb.Set(val)
		}
	}
}

func BenchmarkSetSparse(b *testing.B) {
	data := generateSparseData(benchmarkSizeMedium)
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		rb := New()
		for _, val := range data {
			rb.Set(val)
		}
	}
}

func BenchmarkSetDense(b *testing.B) {
	data := generateDenseData(benchmarkSizeMedium)
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		rb := New()
		for _, val := range data {
			rb.Set(val)
		}
	}
}

func BenchmarkSetContainerBoundary(b *testing.B) {
	data := generateContainerBoundaryData(benchmarkSizeMedium)
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		rb := New()
		for _, val := range data {
			rb.Set(val)
		}
	}
}

// REMOVE OPERATION BENCHMARKS

func BenchmarkRemoveSequentialSmall(b *testing.B) {
	data := generateSequentialData(benchmarkSizeSmall, 0)
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		rb := New()
		// Pre-populate the bitmap
		for _, val := range data {
			rb.Set(val)
		}
		b.StartTimer()
		
		// Remove all values
		for _, val := range data {
			rb.Remove(val)
		}
	}
}

func BenchmarkRemoveSequentialMedium(b *testing.B) {
	data := generateSequentialData(benchmarkSizeMedium, 0)
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		rb := New()
		for _, val := range data {
			rb.Set(val)
		}
		b.StartTimer()
		
		for _, val := range data {
			rb.Remove(val)
		}
	}
}

func BenchmarkRemoveSequentialLarge(b *testing.B) {
	data := generateSequentialData(benchmarkSizeLarge, 0)
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		rb := New()
		for _, val := range data {
			rb.Set(val)
		}
		b.StartTimer()
		
		for _, val := range data {
			rb.Remove(val)
		}
	}
}

func BenchmarkRemoveRandomSmall(b *testing.B) {
	data := generateRandomData(benchmarkSizeSmall, benchmarkSizeSmall*10)
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		rb := New()
		for _, val := range data {
			rb.Set(val)
		}
		b.StartTimer()
		
		for _, val := range data {
			rb.Remove(val)
		}
	}
}

func BenchmarkRemoveRandomMedium(b *testing.B) {
	data := generateRandomData(benchmarkSizeMedium, benchmarkSizeMedium*10)
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		rb := New()
		for _, val := range data {
			rb.Set(val)
		}
		b.StartTimer()
		
		for _, val := range data {
			rb.Remove(val)
		}
	}
}

func BenchmarkRemoveRandomLarge(b *testing.B) {
	data := generateRandomData(benchmarkSizeLarge, benchmarkSizeLarge*10)
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		rb := New()
		for _, val := range data {
			rb.Set(val)
		}
		b.StartTimer()
		
		for _, val := range data {
			rb.Remove(val)
		}
	}
}

func BenchmarkRemoveSparse(b *testing.B) {
	data := generateSparseData(benchmarkSizeMedium)
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		rb := New()
		for _, val := range data {
			rb.Set(val)
		}
		b.StartTimer()
		
		for _, val := range data {
			rb.Remove(val)
		}
	}
}

func BenchmarkRemoveDense(b *testing.B) {
	data := generateDenseData(benchmarkSizeMedium)
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		rb := New()
		for _, val := range data {
			rb.Set(val)
		}
		b.StartTimer()
		
		for _, val := range data {
			rb.Remove(val)
		}
	}
}

// CONTAINS OPERATION BENCHMARKS

func BenchmarkContainsSequentialSmall(b *testing.B) {
	data := generateSequentialData(benchmarkSizeSmall, 0)
	rb := New()
	for _, val := range data {
		rb.Set(val)
	}
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		for _, val := range data {
			rb.Contains(val)
		}
	}
}

func BenchmarkContainsSequentialMedium(b *testing.B) {
	data := generateSequentialData(benchmarkSizeMedium, 0)
	rb := New()
	for _, val := range data {
		rb.Set(val)
	}
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		for _, val := range data {
			rb.Contains(val)
		}
	}
}

func BenchmarkContainsSequentialLarge(b *testing.B) {
	data := generateSequentialData(benchmarkSizeLarge, 0)
	rb := New()
	for _, val := range data {
		rb.Set(val)
	}
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		for _, val := range data {
			rb.Contains(val)
		}
	}
}

func BenchmarkContainsRandomSmall(b *testing.B) {
	data := generateRandomData(benchmarkSizeSmall, benchmarkSizeSmall*10)
	rb := New()
	for _, val := range data {
		rb.Set(val)
	}
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		for _, val := range data {
			rb.Contains(val)
		}
	}
}

func BenchmarkContainsRandomMedium(b *testing.B) {
	data := generateRandomData(benchmarkSizeMedium, benchmarkSizeMedium*10)
	rb := New()
	for _, val := range data {
		rb.Set(val)
	}
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		for _, val := range data {
			rb.Contains(val)
		}
	}
}

func BenchmarkContainsRandomLarge(b *testing.B) {
	data := generateRandomData(benchmarkSizeLarge, benchmarkSizeLarge*10)
	rb := New()
	for _, val := range data {
		rb.Set(val)
	}
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		for _, val := range data {
			rb.Contains(val)
		}
	}
}

func BenchmarkContainsSparse(b *testing.B) {
	data := generateSparseData(benchmarkSizeMedium)
	rb := New()
	for _, val := range data {
		rb.Set(val)
	}
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		for _, val := range data {
			rb.Contains(val)
		}
	}
}

func BenchmarkContainsDense(b *testing.B) {
	data := generateDenseData(benchmarkSizeMedium)
	rb := New()
	for _, val := range data {
		rb.Set(val)
	}
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		for _, val := range data {
			rb.Contains(val)
		}
	}
}

// MIXED OPERATION BENCHMARKS

func BenchmarkMixedOperationsSequential(b *testing.B) {
	data := generateSequentialData(benchmarkSizeMedium, 0)
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		rb := New()
		
		// Set all values
		for _, val := range data {
			rb.Set(val)
		}
		
		// Check all values
		for _, val := range data {
			rb.Contains(val)
		}
		
		// Remove half the values
		for i, val := range data {
			if i%2 == 0 {
				rb.Remove(val)
			}
		}
	}
}

func BenchmarkMixedOperationsRandom(b *testing.B) {
	data := generateRandomData(benchmarkSizeMedium, benchmarkSizeMedium*10)
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		rb := New()
		
		// Set all values
		for _, val := range data {
			rb.Set(val)
		}
		
		// Check all values
		for _, val := range data {
			rb.Contains(val)
		}
		
		// Remove half the values
		for i, val := range data {
			if i%2 == 0 {
				rb.Remove(val)
			}
		}
	}
}

// SINGLE OPERATION BENCHMARKS (for measuring per-operation performance)

func BenchmarkSingleSetSequential(b *testing.B) {
	rb := New()
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		rb.Set(uint32(i))
	}
}

func BenchmarkSingleSetRandom(b *testing.B) {
	rb := New()
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		rb.Set(uint32(rand.IntN(b.N * 10)))
	}
}

func BenchmarkSingleContainsHit(b *testing.B) {
	rb := New()
	// Pre-populate with sequential values
	for i := 0; i < b.N; i++ {
		rb.Set(uint32(i))
	}
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		rb.Contains(uint32(i % b.N))
	}
}

func BenchmarkSingleContainsMiss(b *testing.B) {
	rb := New()
	// Pre-populate with sequential values
	for i := 0; i < 1000; i++ {
		rb.Set(uint32(i))
	}
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		rb.Contains(uint32(i + 10000)) // Values not in bitmap
	}
}

func BenchmarkSingleRemove(b *testing.B) {
	b.StopTimer()
	rb := New()
	// Pre-populate with values
	for i := 0; i < b.N*2; i++ {
		rb.Set(uint32(i))
	}
	b.StartTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		rb.Remove(uint32(i))
	}
}

// COMPARISON BENCHMARKS WITH REFERENCE IMPLEMENTATION
// These benchmarks compare this implementation with github.com/RoaringBitmap/roaring

func BenchmarkComparisonSetSequentialSmall_This(b *testing.B) {
	data := generateSequentialData(benchmarkSizeSmall, 0)
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		rb := New()
		for _, val := range data {
			rb.Set(val)
		}
	}
}

func BenchmarkComparisonSetSequentialSmall_Reference(b *testing.B) {
	data := generateSequentialData(benchmarkSizeSmall, 0)
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		rb := reference.New()
		for _, val := range data {
			rb.Add(val)
		}
	}
}

func BenchmarkComparisonSetRandomMedium_This(b *testing.B) {
	data := generateRandomData(benchmarkSizeMedium, benchmarkSizeMedium*10)
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		rb := New()
		for _, val := range data {
			rb.Set(val)
		}
	}
}

func BenchmarkComparisonSetRandomMedium_Reference(b *testing.B) {
	data := generateRandomData(benchmarkSizeMedium, benchmarkSizeMedium*10)
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		rb := reference.New()
		for _, val := range data {
			rb.Add(val)
		}
	}
}

func BenchmarkComparisonSetSparse_This(b *testing.B) {
	data := generateSparseData(benchmarkSizeMedium)
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		rb := New()
		for _, val := range data {
			rb.Set(val)
		}
	}
}

func BenchmarkComparisonSetSparse_Reference(b *testing.B) {
	data := generateSparseData(benchmarkSizeMedium)
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		rb := reference.New()
		for _, val := range data {
			rb.Add(val)
		}
	}
}

func BenchmarkComparisonContainsSequentialMedium_This(b *testing.B) {
	data := generateSequentialData(benchmarkSizeMedium, 0)
	rb := New()
	for _, val := range data {
		rb.Set(val)
	}
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		for _, val := range data {
			rb.Contains(val)
		}
	}
}

func BenchmarkComparisonContainsSequentialMedium_Reference(b *testing.B) {
	data := generateSequentialData(benchmarkSizeMedium, 0)
	rb := reference.New()
	for _, val := range data {
		rb.Add(val)
	}
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		for _, val := range data {
			rb.Contains(val)
		}
	}
}

func BenchmarkComparisonContainsRandomMedium_This(b *testing.B) {
	data := generateRandomData(benchmarkSizeMedium, benchmarkSizeMedium*10)
	rb := New()
	for _, val := range data {
		rb.Set(val)
	}
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		for _, val := range data {
			rb.Contains(val)
		}
	}
}

func BenchmarkComparisonContainsRandomMedium_Reference(b *testing.B) {
	data := generateRandomData(benchmarkSizeMedium, benchmarkSizeMedium*10)
	rb := reference.New()
	for _, val := range data {
		rb.Add(val)
	}
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		for _, val := range data {
			rb.Contains(val)
		}
	}
}

func BenchmarkComparisonRemoveSequentialMedium_This(b *testing.B) {
	data := generateSequentialData(benchmarkSizeMedium, 0)
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		rb := New()
		for _, val := range data {
			rb.Set(val)
		}
		b.StartTimer()
		
		for _, val := range data {
			rb.Remove(val)
		}
	}
}

func BenchmarkComparisonRemoveSequentialMedium_Reference(b *testing.B) {
	data := generateSequentialData(benchmarkSizeMedium, 0)
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		rb := reference.New()
		for _, val := range data {
			rb.Add(val)
		}
		b.StartTimer()
		
		for _, val := range data {
			rb.Remove(val)
		}
	}
}

func BenchmarkComparisonRemoveRandomMedium_This(b *testing.B) {
	data := generateRandomData(benchmarkSizeMedium, benchmarkSizeMedium*10)
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		rb := New()
		for _, val := range data {
			rb.Set(val)
		}
		b.StartTimer()
		
		for _, val := range data {
			rb.Remove(val)
		}
	}
}

func BenchmarkComparisonRemoveRandomMedium_Reference(b *testing.B) {
	data := generateRandomData(benchmarkSizeMedium, benchmarkSizeMedium*10)
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		rb := reference.New()
		for _, val := range data {
			rb.Add(val)
		}
		b.StartTimer()
		
		for _, val := range data {
			rb.Remove(val)
		}
	}
}
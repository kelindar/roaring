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

// Helper functions for benchmarking different operations

// benchmarkSet runs Set operation benchmarks with the given data generator
func benchmarkSet(b *testing.B, name string, dataGen func() []uint32) {
	b.Run(name, func(b *testing.B) {
		data := dataGen()
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			rb := New()
			for _, val := range data {
				rb.Set(val)
			}
		}
	})
}

// benchmarkRemove runs Remove operation benchmarks with the given data generator
func benchmarkRemove(b *testing.B, name string, dataGen func() []uint32) {
	b.Run(name, func(b *testing.B) {
		data := dataGen()
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
	})
}

// benchmarkContains runs Contains operation benchmarks with the given data generator
func benchmarkContains(b *testing.B, name string, dataGen func() []uint32) {
	b.Run(name, func(b *testing.B) {
		data := dataGen()
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
	})
}

// benchmarkMixed runs mixed operation benchmarks with the given data generator
func benchmarkMixed(b *testing.B, name string, dataGen func() []uint32) {
	b.Run(name, func(b *testing.B) {
		data := dataGen()
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
	})
}

// benchmarkComparison runs comparison benchmarks between this implementation and reference
func benchmarkComparison(b *testing.B, name string, dataGen func() []uint32, operation string) {
	data := dataGen()
	
	// This implementation
	b.Run(name+"_This", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		switch operation {
		case "Set":
			for i := 0; i < b.N; i++ {
				rb := New()
				for _, val := range data {
					rb.Set(val)
				}
			}
		case "Contains":
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
		case "Remove":
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
	})
	
	// Reference implementation
	b.Run(name+"_Reference", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		switch operation {
		case "Set":
			for i := 0; i < b.N; i++ {
				rb := reference.New()
				for _, val := range data {
					rb.Add(val)
				}
			}
		case "Contains":
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
		case "Remove":
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
	})
}

// SET OPERATION BENCHMARKS

func BenchmarkSet(b *testing.B) {
	// Sequential benchmarks
	benchmarkSet(b, "SequentialSmall", func() []uint32 {
		return generateSequentialData(benchmarkSizeSmall, 0)
	})
	benchmarkSet(b, "SequentialMedium", func() []uint32 {
		return generateSequentialData(benchmarkSizeMedium, 0)
	})
	benchmarkSet(b, "SequentialLarge", func() []uint32 {
		return generateSequentialData(benchmarkSizeLarge, 0)
	})
	
	// Random benchmarks
	benchmarkSet(b, "RandomSmall", func() []uint32 {
		return generateRandomData(benchmarkSizeSmall, benchmarkSizeSmall*10)
	})
	benchmarkSet(b, "RandomMedium", func() []uint32 {
		return generateRandomData(benchmarkSizeMedium, benchmarkSizeMedium*10)
	})
	benchmarkSet(b, "RandomLarge", func() []uint32 {
		return generateRandomData(benchmarkSizeLarge, benchmarkSizeLarge*10)
	})
	
	// Special pattern benchmarks
	benchmarkSet(b, "Sparse", func() []uint32 {
		return generateSparseData(benchmarkSizeMedium)
	})
	benchmarkSet(b, "Dense", func() []uint32 {
		return generateDenseData(benchmarkSizeMedium)
	})
	benchmarkSet(b, "ContainerBoundary", func() []uint32 {
		return generateContainerBoundaryData(benchmarkSizeMedium)
	})
}

// Individual benchmarks for backward compatibility
func BenchmarkSetSequentialSmall(b *testing.B) {
	benchmarkSet(b, "SequentialSmall", func() []uint32 {
		return generateSequentialData(benchmarkSizeSmall, 0)
	})
}

func BenchmarkSetSequentialMedium(b *testing.B) {
	benchmarkSet(b, "SequentialMedium", func() []uint32 {
		return generateSequentialData(benchmarkSizeMedium, 0)
	})
}

func BenchmarkSetSequentialLarge(b *testing.B) {
	benchmarkSet(b, "SequentialLarge", func() []uint32 {
		return generateSequentialData(benchmarkSizeLarge, 0)
	})
}

func BenchmarkSetRandomSmall(b *testing.B) {
	benchmarkSet(b, "RandomSmall", func() []uint32 {
		return generateRandomData(benchmarkSizeSmall, benchmarkSizeSmall*10)
	})
}

func BenchmarkSetRandomMedium(b *testing.B) {
	benchmarkSet(b, "RandomMedium", func() []uint32 {
		return generateRandomData(benchmarkSizeMedium, benchmarkSizeMedium*10)
	})
}

func BenchmarkSetRandomLarge(b *testing.B) {
	benchmarkSet(b, "RandomLarge", func() []uint32 {
		return generateRandomData(benchmarkSizeLarge, benchmarkSizeLarge*10)
	})
}

func BenchmarkSetSparse(b *testing.B) {
	benchmarkSet(b, "Sparse", func() []uint32 {
		return generateSparseData(benchmarkSizeMedium)
	})
}

func BenchmarkSetDense(b *testing.B) {
	benchmarkSet(b, "Dense", func() []uint32 {
		return generateDenseData(benchmarkSizeMedium)
	})
}

func BenchmarkSetContainerBoundary(b *testing.B) {
	benchmarkSet(b, "ContainerBoundary", func() []uint32 {
		return generateContainerBoundaryData(benchmarkSizeMedium)
	})
}

// REMOVE OPERATION BENCHMARKS

func BenchmarkRemove(b *testing.B) {
	// Sequential benchmarks
	benchmarkRemove(b, "SequentialSmall", func() []uint32 {
		return generateSequentialData(benchmarkSizeSmall, 0)
	})
	benchmarkRemove(b, "SequentialMedium", func() []uint32 {
		return generateSequentialData(benchmarkSizeMedium, 0)
	})
	benchmarkRemove(b, "SequentialLarge", func() []uint32 {
		return generateSequentialData(benchmarkSizeLarge, 0)
	})
	
	// Random benchmarks
	benchmarkRemove(b, "RandomSmall", func() []uint32 {
		return generateRandomData(benchmarkSizeSmall, benchmarkSizeSmall*10)
	})
	benchmarkRemove(b, "RandomMedium", func() []uint32 {
		return generateRandomData(benchmarkSizeMedium, benchmarkSizeMedium*10)
	})
	benchmarkRemove(b, "RandomLarge", func() []uint32 {
		return generateRandomData(benchmarkSizeLarge, benchmarkSizeLarge*10)
	})
	
	// Special pattern benchmarks
	benchmarkRemove(b, "Sparse", func() []uint32 {
		return generateSparseData(benchmarkSizeMedium)
	})
	benchmarkRemove(b, "Dense", func() []uint32 {
		return generateDenseData(benchmarkSizeMedium)
	})
}

// Individual benchmarks for backward compatibility
func BenchmarkRemoveSequentialSmall(b *testing.B) {
	benchmarkRemove(b, "SequentialSmall", func() []uint32 {
		return generateSequentialData(benchmarkSizeSmall, 0)
	})
}

func BenchmarkRemoveSequentialMedium(b *testing.B) {
	benchmarkRemove(b, "SequentialMedium", func() []uint32 {
		return generateSequentialData(benchmarkSizeMedium, 0)
	})
}

func BenchmarkRemoveSequentialLarge(b *testing.B) {
	benchmarkRemove(b, "SequentialLarge", func() []uint32 {
		return generateSequentialData(benchmarkSizeLarge, 0)
	})
}

func BenchmarkRemoveRandomSmall(b *testing.B) {
	benchmarkRemove(b, "RandomSmall", func() []uint32 {
		return generateRandomData(benchmarkSizeSmall, benchmarkSizeSmall*10)
	})
}

func BenchmarkRemoveRandomMedium(b *testing.B) {
	benchmarkRemove(b, "RandomMedium", func() []uint32 {
		return generateRandomData(benchmarkSizeMedium, benchmarkSizeMedium*10)
	})
}

func BenchmarkRemoveRandomLarge(b *testing.B) {
	benchmarkRemove(b, "RandomLarge", func() []uint32 {
		return generateRandomData(benchmarkSizeLarge, benchmarkSizeLarge*10)
	})
}

func BenchmarkRemoveSparse(b *testing.B) {
	benchmarkRemove(b, "Sparse", func() []uint32 {
		return generateSparseData(benchmarkSizeMedium)
	})
}

func BenchmarkRemoveDense(b *testing.B) {
	benchmarkRemove(b, "Dense", func() []uint32 {
		return generateDenseData(benchmarkSizeMedium)
	})
}

// CONTAINS OPERATION BENCHMARKS

func BenchmarkContains(b *testing.B) {
	// Sequential benchmarks
	benchmarkContains(b, "SequentialSmall", func() []uint32 {
		return generateSequentialData(benchmarkSizeSmall, 0)
	})
	benchmarkContains(b, "SequentialMedium", func() []uint32 {
		return generateSequentialData(benchmarkSizeMedium, 0)
	})
	benchmarkContains(b, "SequentialLarge", func() []uint32 {
		return generateSequentialData(benchmarkSizeLarge, 0)
	})
	
	// Random benchmarks
	benchmarkContains(b, "RandomSmall", func() []uint32 {
		return generateRandomData(benchmarkSizeSmall, benchmarkSizeSmall*10)
	})
	benchmarkContains(b, "RandomMedium", func() []uint32 {
		return generateRandomData(benchmarkSizeMedium, benchmarkSizeMedium*10)
	})
	benchmarkContains(b, "RandomLarge", func() []uint32 {
		return generateRandomData(benchmarkSizeLarge, benchmarkSizeLarge*10)
	})
	
	// Special pattern benchmarks
	benchmarkContains(b, "Sparse", func() []uint32 {
		return generateSparseData(benchmarkSizeMedium)
	})
	benchmarkContains(b, "Dense", func() []uint32 {
		return generateDenseData(benchmarkSizeMedium)
	})
}

// Individual benchmarks for backward compatibility
func BenchmarkContainsSequentialSmall(b *testing.B) {
	benchmarkContains(b, "SequentialSmall", func() []uint32 {
		return generateSequentialData(benchmarkSizeSmall, 0)
	})
}

func BenchmarkContainsSequentialMedium(b *testing.B) {
	benchmarkContains(b, "SequentialMedium", func() []uint32 {
		return generateSequentialData(benchmarkSizeMedium, 0)
	})
}

func BenchmarkContainsSequentialLarge(b *testing.B) {
	benchmarkContains(b, "SequentialLarge", func() []uint32 {
		return generateSequentialData(benchmarkSizeLarge, 0)
	})
}

func BenchmarkContainsRandomSmall(b *testing.B) {
	benchmarkContains(b, "RandomSmall", func() []uint32 {
		return generateRandomData(benchmarkSizeSmall, benchmarkSizeSmall*10)
	})
}

func BenchmarkContainsRandomMedium(b *testing.B) {
	benchmarkContains(b, "RandomMedium", func() []uint32 {
		return generateRandomData(benchmarkSizeMedium, benchmarkSizeMedium*10)
	})
}

func BenchmarkContainsRandomLarge(b *testing.B) {
	benchmarkContains(b, "RandomLarge", func() []uint32 {
		return generateRandomData(benchmarkSizeLarge, benchmarkSizeLarge*10)
	})
}

func BenchmarkContainsSparse(b *testing.B) {
	benchmarkContains(b, "Sparse", func() []uint32 {
		return generateSparseData(benchmarkSizeMedium)
	})
}

func BenchmarkContainsDense(b *testing.B) {
	benchmarkContains(b, "Dense", func() []uint32 {
		return generateDenseData(benchmarkSizeMedium)
	})
}

// MIXED OPERATION BENCHMARKS

func BenchmarkMixedOperations(b *testing.B) {
	benchmarkMixed(b, "Sequential", func() []uint32 {
		return generateSequentialData(benchmarkSizeMedium, 0)
	})
	benchmarkMixed(b, "Random", func() []uint32 {
		return generateRandomData(benchmarkSizeMedium, benchmarkSizeMedium*10)
	})
}

// Individual benchmarks for backward compatibility
func BenchmarkMixedOperationsSequential(b *testing.B) {
	benchmarkMixed(b, "Sequential", func() []uint32 {
		return generateSequentialData(benchmarkSizeMedium, 0)
	})
}

func BenchmarkMixedOperationsRandom(b *testing.B) {
	benchmarkMixed(b, "Random", func() []uint32 {
		return generateRandomData(benchmarkSizeMedium, benchmarkSizeMedium*10)
	})
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

func BenchmarkComparison(b *testing.B) {
	// Set operation comparisons
	benchmarkComparison(b, "SetSequentialSmall", func() []uint32 {
		return generateSequentialData(benchmarkSizeSmall, 0)
	}, "Set")
	benchmarkComparison(b, "SetRandomMedium", func() []uint32 {
		return generateRandomData(benchmarkSizeMedium, benchmarkSizeMedium*10)
	}, "Set")
	benchmarkComparison(b, "SetSparse", func() []uint32 {
		return generateSparseData(benchmarkSizeMedium)
	}, "Set")
	
	// Contains operation comparisons
	benchmarkComparison(b, "ContainsSequentialMedium", func() []uint32 {
		return generateSequentialData(benchmarkSizeMedium, 0)
	}, "Contains")
	benchmarkComparison(b, "ContainsRandomMedium", func() []uint32 {
		return generateRandomData(benchmarkSizeMedium, benchmarkSizeMedium*10)
	}, "Contains")
	
	// Remove operation comparisons
	benchmarkComparison(b, "RemoveSequentialMedium", func() []uint32 {
		return generateSequentialData(benchmarkSizeMedium, 0)
	}, "Remove")
	benchmarkComparison(b, "RemoveRandomMedium", func() []uint32 {
		return generateRandomData(benchmarkSizeMedium, benchmarkSizeMedium*10)
	}, "Remove")
}

// Individual comparison benchmarks for backward compatibility
func BenchmarkComparisonSetSequentialSmall_This(b *testing.B) {
	benchmarkComparison(b, "SetSequentialSmall", func() []uint32 {
		return generateSequentialData(benchmarkSizeSmall, 0)
	}, "Set")
}

func BenchmarkComparisonSetSequentialSmall_Reference(b *testing.B) {
	// This will be handled by the benchmarkComparison function
	b.Skip("Use BenchmarkComparison instead")
}

func BenchmarkComparisonSetRandomMedium_This(b *testing.B) {
	benchmarkComparison(b, "SetRandomMedium", func() []uint32 {
		return generateRandomData(benchmarkSizeMedium, benchmarkSizeMedium*10)
	}, "Set")
}

func BenchmarkComparisonSetRandomMedium_Reference(b *testing.B) {
	b.Skip("Use BenchmarkComparison instead")
}

func BenchmarkComparisonSetSparse_This(b *testing.B) {
	benchmarkComparison(b, "SetSparse", func() []uint32 {
		return generateSparseData(benchmarkSizeMedium)
	}, "Set")
}

func BenchmarkComparisonSetSparse_Reference(b *testing.B) {
	b.Skip("Use BenchmarkComparison instead")
}

func BenchmarkComparisonContainsSequentialMedium_This(b *testing.B) {
	benchmarkComparison(b, "ContainsSequentialMedium", func() []uint32 {
		return generateSequentialData(benchmarkSizeMedium, 0)
	}, "Contains")
}

func BenchmarkComparisonContainsSequentialMedium_Reference(b *testing.B) {
	b.Skip("Use BenchmarkComparison instead")
}

func BenchmarkComparisonContainsRandomMedium_This(b *testing.B) {
	benchmarkComparison(b, "ContainsRandomMedium", func() []uint32 {
		return generateRandomData(benchmarkSizeMedium, benchmarkSizeMedium*10)
	}, "Contains")
}

func BenchmarkComparisonContainsRandomMedium_Reference(b *testing.B) {
	b.Skip("Use BenchmarkComparison instead")
}

func BenchmarkComparisonRemoveSequentialMedium_This(b *testing.B) {
	benchmarkComparison(b, "RemoveSequentialMedium", func() []uint32 {
		return generateSequentialData(benchmarkSizeMedium, 0)
	}, "Remove")
}

func BenchmarkComparisonRemoveSequentialMedium_Reference(b *testing.B) {
	b.Skip("Use BenchmarkComparison instead")
}

func BenchmarkComparisonRemoveRandomMedium_This(b *testing.B) {
	benchmarkComparison(b, "RemoveRandomMedium", func() []uint32 {
		return generateRandomData(benchmarkSizeMedium, benchmarkSizeMedium*10)
	}, "Remove")
}

func BenchmarkComparisonRemoveRandomMedium_Reference(b *testing.B) {
	b.Skip("Use BenchmarkComparison instead")
}
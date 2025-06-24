package roaring

import (
	"math/rand/v2"
	"testing"
)

// BenchmarkRunContainerOps benchmarks specific run container operations
func BenchmarkRunContainerOps(b *testing.B) {
	// Test different run container scenarios
	scenarios := []struct {
		name string
		size int
		gen  func(size int) []uint16
	}{
		{
			name: "sequential",
			size: 1000,
			gen: func(size int) []uint16 {
				data := make([]uint16, size)
				for i := 0; i < size; i++ {
					data[i] = uint16(i)
				}
				return data
			},
		},
		{
			name: "sparse", 
			size: 1000,
			gen: func(size int) []uint16 {
				data := make([]uint16, size)
				for i := 0; i < size; i++ {
					data[i] = uint16(i * 10) // Large gaps
				}
				return data
			},
		},
		{
			name: "random",
			size: 1000,
			gen: func(size int) []uint16 {
				data := make([]uint16, size)
				for i := 0; i < size; i++ {
					data[i] = uint16(rand.IntN(65536))
				}
				return data
			},
		},
	}

	for _, scenario := range scenarios {
		data := scenario.gen(scenario.size)
		
		// Benchmark runSet operations
		b.Run("runSet-"+scenario.name, func(b *testing.B) {
			c := &container{Type: typeRun, Size: 0, Data: nil}
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				for _, value := range data {
					c.runSet(value)
				}
				// Reset container for next iteration
				c.Size = 0
				c.Data = nil
			}
		})

		// Benchmark runHas operations on populated container
		b.Run("runHas-"+scenario.name, func(b *testing.B) {
			c := &container{Type: typeRun, Size: 0, Data: nil}
			// Populate container
			for _, value := range data {
				c.runSet(value)
			}
			
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				for _, value := range data {
					c.runHas(value)
				}
			}
		})

		// Benchmark runDel operations  
		b.Run("runDel-"+scenario.name, func(b *testing.B) {
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				// Populate container
				c := &container{Type: typeRun, Size: 0, Data: nil}
				for _, value := range data {
					c.runSet(value)
				}
				
				b.StartTimer()
				for _, value := range data {
					c.runDel(value)
				}
				b.StopTimer()
			}
		})
	}
}

// BenchmarkRunArrayOps benchmarks run array manipulation operations
func BenchmarkRunArrayOps(b *testing.B) {
	// Create a container with multiple runs
	c := &container{Type: typeRun, Size: 0, Data: nil}
	
	// Add some runs with gaps to create a realistic scenario
	values := []uint16{1, 2, 3, 10, 11, 12, 20, 21, 30, 31, 32, 33}
	for _, v := range values {
		c.runSet(v)
	}
	
	b.Run("runInsertRunAt", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// Save original state
			originalData := make([]uint16, len(c.Data))
			copy(originalData, c.Data)
			originalSize := c.Size
			
			// Insert a new run in the middle
			c.runInsertRunAt(2, run{15, 16})
			
			// Restore original state for next iteration
			c.Data = originalData
			c.Size = originalSize
		}
	})
	
	b.Run("runRemoveRunAt", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// Save original state
			originalData := make([]uint16, len(c.Data))
			copy(originalData, c.Data)
			originalSize := c.Size
			
			// Remove a run from the middle
			c.runRemoveRunAt(1)
			
			// Restore original state for next iteration
			c.Data = originalData
			c.Size = originalSize
		}
	})
}

// BenchmarkRunConversions benchmarks container type conversions
func BenchmarkRunConversions(b *testing.B) {
	// Test different patterns that might convert to run containers
	
	b.Run("arrayToRun-sequential", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			c := &container{Type: typeArray, Size: 0, Data: make([]uint16, 0, 100)}
			// Add sequential values that should convert to runs
			for j := 0; j < 100; j++ {
				c.arrSet(uint16(j))
			}
			
			b.StartTimer()
			c.arrToRun()
			b.StopTimer()
		}
	})
	
	b.Run("runToArray-sparse", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			c := &container{Type: typeRun, Size: 0, Data: nil}
			// Add sparse values that should convert to array
			for j := 0; j < 50; j++ {
				c.runSet(uint16(j * 100)) // Very sparse
			}
			
			b.StartTimer()
			c.runToArray()
			b.StopTimer()
		}
	})
}
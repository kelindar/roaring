package roaring

import (
	"testing"
	"math/rand/v2"
	"github.com/RoaringBitmap/roaring"
)

// BenchmarkRunContainerVsReference compares optimized run container operations against reference implementation
func BenchmarkRunContainerVsReference(b *testing.B) {
	// Test scenarios that specifically exercise run containers
	scenarios := []struct {
		name   string
		values []uint32
	}{
		{
			name: "dense-runs",
			values: func() []uint32 {
				// Create dense consecutive runs that should stay as run containers
				var vals []uint32
				for i := uint32(0); i < 100; i++ {
					for j := uint32(0); j < 50; j++ {
						vals = append(vals, i*1000+j) // 50 consecutive values, then gap
					}
				}
				return vals
			}(),
		},
		{
			name: "sparse-runs", 
			values: func() []uint32 {
				// Create sparse patterns that benefit from run representation
				var vals []uint32
				for i := uint32(0); i < 1000; i++ {
					vals = append(vals, i*100) // Single values with large gaps
				}
				return vals
			}(),
		},
		{
			name: "mixed-runs",
			values: func() []uint32 {
				// Mix of small runs and single values
				var vals []uint32
				for i := uint32(0); i < 200; i++ {
					if i%3 == 0 {
						// Add small run
						for j := uint32(0); j < 3; j++ {
							vals = append(vals, i*50+j)
						}
					} else {
						// Add single value
						vals = append(vals, i*50)
					}
				}
				return vals
			}(),
		},
	}

	for _, scenario := range scenarios {
		b.Run("set-"+scenario.name, func(b *testing.B) {
			// Force both implementations to use similar container types by testing in isolation
			b.Run("optimized", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					rb := New()
					for _, v := range scenario.values {
						rb.Set(v)
					}
				}
			})
			
			b.Run("reference", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					rb := roaring.New()
					for _, v := range scenario.values {
						rb.Add(v)
					}
				}
			})
		})

		b.Run("contains-"+scenario.name, func(b *testing.B) {
			// Pre-populate bitmaps
			rbOptimized := New()
			rbReference := roaring.New()
			for _, v := range scenario.values {
				rbOptimized.Set(v)
				rbReference.Add(v)
			}

			b.Run("optimized", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					for _, v := range scenario.values {
						rbOptimized.Contains(v)
					}
				}
			})
			
			b.Run("reference", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					for _, v := range scenario.values {
						rbReference.Contains(v)
					}
				}
			})
		})

		b.Run("remove-"+scenario.name, func(b *testing.B) {
			b.Run("optimized", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					rb := New()
					// Populate
					for _, v := range scenario.values {
						rb.Set(v)
					}
					// Remove
					b.StartTimer()
					for _, v := range scenario.values {
						rb.Remove(v)
					}
					b.StopTimer()
				}
			})
			
			b.Run("reference", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					rb := roaring.New()
					// Populate
					for _, v := range scenario.values {
						rb.Add(v)
					}
					// Remove
					b.StartTimer()
					for _, v := range scenario.values {
						rb.Remove(v)
					}
					b.StopTimer()
				}
			})
		})
	}
}

// BenchmarkRunContainerSpecific tests operations that specifically exercise run container code paths
func BenchmarkRunContainerSpecific(b *testing.B) {
	// Create scenarios that will definitely use run containers
	b.Run("pure-run-operations", func(b *testing.B) {
		// Test pure run container operations by directly creating run containers
		b.Run("sequential-insertions", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				c := &container{Type: typeRun, Size: 0, Data: nil}
				// Insert sequential values that will create consecutive runs
				for j := uint16(0); j < 1000; j++ {
					c.runSet(j)
				}
			}
		})

		b.Run("sparse-insertions", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				c := &container{Type: typeRun, Size: 0, Data: nil}
				// Insert sparse values that will create many separate runs
				for j := uint16(0); j < 1000; j++ {
					c.runSet(j * 10) // Large gaps between values
				}
			}
		})

		b.Run("random-access", func(b *testing.B) {
			// Pre-populate a run container
			c := &container{Type: typeRun, Size: 0, Data: nil}
			values := make([]uint16, 1000)
			for i := 0; i < 1000; i++ {
				values[i] = uint16(rand.IntN(65536))
				c.runSet(values[i])
			}
			
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				for _, v := range values {
					c.runHas(v)
				}
			}
		})

		b.Run("mixed-operations", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				c := &container{Type: typeRun, Size: 0, Data: nil}
				
				// Mix of set and delete operations
				for j := uint16(0); j < 500; j++ {
					c.runSet(j * 2)     // Set even numbers
				}
				for j := uint16(0); j < 250; j++ {
					c.runDel(j * 4)     // Delete every 4th number
				}
				for j := uint16(1); j < 500; j += 2 {
					c.runSet(j)         // Set odd numbers
				}
			}
		})
	})
}
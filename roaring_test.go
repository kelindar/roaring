package roaring

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasicOperations(t *testing.T) {
	tests := []struct {
		name string
		gen  dataGen
	}{
		{"sequential", genSeq(100, 0)},
		{"random", genRand(100, 10000)},
		{"sparse", genSparse(50)},
		{"boundary", genBoundary()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, _ := tt.gen()
			our, ref := testPair(data)

			// Test basic operations
			assertEqualBitmaps(t, our, ref)

			// Test removal pattern
			for i, v := range data {
				if i%2 == 0 {
					our.Remove(v)
					ref.Remove(v)
				}
			}
			assertEqualBitmaps(t, our, ref)

			// Test clear
			our.Clear()
			ref.Clear()
			assertEqualBitmaps(t, our, ref)
		})
	}
}

func TestOperationsComprehensive(t *testing.T) {
	tests := []struct {
		name string
		gen  func(int) dataGen
	}{
		{"seq", func(size int) dataGen { return genSeq(size, 0) }},
		{"rnd", func(size int) dataGen { return genRand(size, uint32(size*10)) }},
		{"sps", func(size int) dataGen { return genSparse(size) }},
		{"dns", func(size int) dataGen { return genDense(size) }},
	}

	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		for _, tt := range tests {
			gen := tt.gen(size)
			data, _ := gen()

			t.Run(fmt.Sprintf("%s_%d", tt.name, size), func(t *testing.T) {
				our, ref := testPairRandom(data)

				// Test with random 50% fill
				assertEqualBitmaps(t, our, ref)

				// Test optimize
				our.Optimize()
				assertEqualBitmaps(t, our, ref)

				// Test more operations
				for i := 0; i < len(data)/4; i++ {
					v := data[rand.IntN(len(data))]
					switch rand.IntN(3) {
					case 0:
						our.Set(v)
						ref.Set(v)
					case 1:
						our.Remove(v)
						ref.Remove(v)
					case 2:
						// Just check contains
						assert.Equal(t, ref.Contains(v), our.Contains(v))
					}
				}
				assertEqualBitmaps(t, our, ref)
			})
		}
	}
}

func TestTransitions(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *Bitmap
		validate func(*testing.T, *Bitmap)
	}{
		{
			name: "array_to_bitmap",
			setup: func() *Bitmap {
				rb := New()
				// Add many sparse values to force bitmap (not consecutive to avoid run)
				for i := 0; i < 5000; i++ {
					rb.Set(uint32(i * 3)) // Sparse values
				}
				return rb
			},
			validate: func(t *testing.T, rb *Bitmap) {
				c, exists := rb.findContainer(0)
				assert.True(t, exists)
				assert.Equal(t, typeBitmap, c.Type)
				assert.Equal(t, 5000, rb.Count())
			},
		},
		{
			name: "bitmap_to_run",
			setup: func() *Bitmap {
				rb := New()
				// Create bitmap with consecutive values
				for i := 0; i < 60000; i++ {
					rb.Set(uint32(i))
				}
				rb.Optimize() // Should convert to run
				return rb
			},
			validate: func(t *testing.T, rb *Bitmap) {
				c, exists := rb.findContainer(0)
				assert.True(t, exists)
				assert.Equal(t, typeRun, c.Type)
				assert.Equal(t, 60000, rb.Count())
			},
		},
		{
			name: "run_split",
			setup: func() *Bitmap {
				rb := New()
				// Create run
				for i := 1000; i <= 2000; i++ {
					rb.Set(uint32(i))
				}
				rb.Optimize()
				// Split the run by removing middle
				rb.Remove(1500)
				return rb
			},
			validate: func(t *testing.T, rb *Bitmap) {
				assert.False(t, rb.Contains(1500))
				assert.True(t, rb.Contains(1499))
				assert.True(t, rb.Contains(1501))
				assert.Equal(t, 1000, rb.Count()) // 1001 - 1
			},
		},
		{
			name: "multiple_containers",
			setup: func() *Bitmap {
				rb := New()
				// Container 0: Array
				rb.Set(1)
				rb.Set(5)
				rb.Set(10)

				// Container 1: Bitmap
				for i := 0; i < 3000; i++ {
					rb.Set(uint32(65536 + i*2))
				}

				// Container 2: Run
				for i := 131072; i <= 131572; i++ {
					rb.Set(uint32(i))
				}
				rb.Optimize()
				return rb
			},
			validate: func(t *testing.T, rb *Bitmap) {
				// Verify we have containers
				c0, exists := rb.findContainer(0)
				assert.True(t, exists)
				assert.Equal(t, typeArray, c0.Type)

				c1, exists := rb.findContainer(1)
				assert.True(t, exists)
				assert.Equal(t, typeBitmap, c1.Type)

				c2, exists := rb.findContainer(2)
				assert.True(t, exists)
				assert.Equal(t, typeRun, c2.Type)

				assert.Equal(t, 3504, rb.Count()) // 3 + 3000 + 501
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := tt.setup()
			tt.validate(t, rb)
		})
	}
}

func TestContainerConversions(t *testing.T) {
	t.Run("bitmap_to_array", func(t *testing.T) {
		rb := New()
		// Create bitmap with many values, then remove most to trigger array conversion
		for i := 0; i < 5000; i++ {
			rb.Set(uint32(i * 3))
		}
		// Force optimization to make sure it becomes bitmap
		rb.Optimize()

		// Verify it's bitmap initially
		c, exists := rb.findContainer(0)
		assert.True(t, exists)
		assert.Equal(t, typeBitmap, c.Type)

		// Remove most values to trigger bitmap->array conversion
		for i := 500; i < 5000; i++ {
			rb.Remove(uint32(i * 3))
		}

		// Force optimization to trigger conversion
		rb.Optimize()

		// Should now be array (or at least much smaller)
		c, exists = rb.findContainer(0)
		assert.True(t, exists)
		assert.Equal(t, 500, rb.Count())
		// Should be array type since we reduced to small number of sparse values
		assert.Equal(t, typeArray, c.Type)

		// Verify remaining values are correct
		for i := 0; i < 500; i++ {
			assert.True(t, rb.Contains(uint32(i*3)))
		}
	})

	t.Run("run_to_array", func(t *testing.T) {
		rb := New()
		// Create run container with many consecutive values to ensure it becomes run
		for i := 1000; i <= 2000; i++ {
			rb.Set(uint32(i))
		}
		rb.Optimize()

		// Verify it's run initially
		c, exists := rb.findContainer(0)
		assert.True(t, exists)
		assert.Equal(t, typeRun, c.Type)

		// Remove values to create many single-value runs (avgRunLength < 2.0)
		for i := 1001; i <= 1999; i += 2 {
			rb.Remove(uint32(i))
		}

		// Force optimization - should become array due to low average run length
		rb.Optimize()

		c, exists = rb.findContainer(0)
		assert.True(t, exists)
		expectedCount := 501 // 1000, 1002, 1004, ..., 1998, 2000
		assert.Equal(t, expectedCount, rb.Count())
		// Should be array type since average run length is now 1.0
		assert.Equal(t, typeArray, c.Type)

		// Verify correct values remain
		for i := 1000; i <= 2000; i += 2 {
			assert.True(t, rb.Contains(uint32(i)))
		}
	})
}

func TestContainerEdgeCases(t *testing.T) {
	t.Run("empty_containers", func(t *testing.T) {
		// Test with empty bitmap container
		c := &container{Type: typeBitmap, Size: 0, Data: make([]uint16, 0)}

		// Test bmp() with empty data
		bmp := c.bmp()
		assert.Nil(t, bmp)

		// Test run() with empty data
		c.Type = typeRun
		runs := c.run()
		assert.Nil(t, runs)
	})

	t.Run("container_removal_edge_cases", func(t *testing.T) {
		rb := New()
		rb.Set(65535) // Last value in container 0
		rb.Set(65536) // First value in container 1

		// Remove and verify container cleanup
		rb.Remove(65535)
		rb.Remove(65536)

		assert.Equal(t, 0, rb.Count())

		// Verify containers were removed
		_, exists := rb.findContainer(0)
		assert.False(t, exists)
		_, exists = rb.findContainer(1)
		assert.False(t, exists)
	})

	t.Run("bitmap_del_out_of_bounds", func(t *testing.T) {
		rb := New()
		// Create small bitmap
		rb.Set(1)
		rb.Set(5)

		c, exists := rb.findContainer(0)
		assert.True(t, exists)

		// Try to delete value that would be out of bounds
		deleted := c.bmpDel(65000) // Way beyond the bitmap size
		assert.False(t, deleted)
	})

	t.Run("run_remove_edge_cases", func(t *testing.T) {
		rb := New()
		// Create run with single value
		rb.Set(1000)
		c, _ := rb.findContainer(0)
		c.Type = typeRun
		c.Data = []uint16{1000, 1000} // Single value run
		c.Size = 1

		// Remove the only value - should remove the run
		removed := c.runDel(1000)
		assert.True(t, removed)
		assert.Equal(t, uint32(0), c.Size)

		// Test removing from invalid index (this tests runRemoveRunAt edge case)
		c.runRemoveRunAt(10) // Invalid index - should not panic
	})
}

func TestStress(t *testing.T) {
	t.Run("large_sequential", func(t *testing.T) {
		rb := New()
		size := 100000

		// Add sequential values
		for i := 0; i < size; i++ {
			rb.Set(uint32(i))
		}
		assert.Equal(t, size, rb.Count())

		// Optimize (should become run containers)
		rb.Optimize()
		assert.Equal(t, size, rb.Count())

		// Verify all values present
		for i := 0; i < size; i++ {
			assert.True(t, rb.Contains(uint32(i)))
		}

		// Remove every 10th value
		removed := 0
		for i := 0; i < size; i += 10 {
			rb.Remove(uint32(i))
			removed++
		}
		assert.Equal(t, size-removed, rb.Count())
	})

	t.Run("large_random", func(t *testing.T) {
		rb := New()
		var ref []uint32

		// Add random values
		for i := 0; i < 10000; i++ {
			v := uint32(rand.IntN(1000000))
			if !rb.Contains(v) {
				rb.Set(v)
				ref = append(ref, v)
			}
		}

		// Verify count
		assert.Equal(t, len(ref), rb.Count())

		// Verify all values
		for _, v := range ref {
			assert.True(t, rb.Contains(v))
		}

		// Test optimize doesn't break anything
		rb.Optimize()
		assert.Equal(t, len(ref), rb.Count())
		for _, v := range ref {
			assert.True(t, rb.Contains(v))
		}
	})

	t.Run("container_splits", func(t *testing.T) {
		rb := New()

		// Create runs that will be split
		for container := 0; container < 5; container++ {
			base := uint32(container * 65536)
			// Add consecutive values
			for i := 1000; i <= 2000; i++ {
				rb.Set(base + uint32(i))
			}
		}
		rb.Optimize()

		initialCount := rb.Count()

		// Split each run by removing middle values
		for container := 0; container < 5; container++ {
			base := uint32(container * 65536)
			for i := 1400; i <= 1600; i++ {
				rb.Remove(base + uint32(i))
			}
		}

		expectedRemoved := 5 * 201 // 5 containers × 201 values each
		assert.Equal(t, initialCount-expectedRemoved, rb.Count())

		// Verify the boundaries are intact
		for container := 0; container < 5; container++ {
			base := uint32(container * 65536)
			assert.True(t, rb.Contains(base+1000))
			assert.True(t, rb.Contains(base+1399))
			assert.False(t, rb.Contains(base+1400))
			assert.False(t, rb.Contains(base+1600))
			assert.True(t, rb.Contains(base+1601))
			assert.True(t, rb.Contains(base+2000))
		}
	})
}

func TestContainerOptimization(t *testing.T) {
	t.Run("array_stays_array", func(t *testing.T) {
		rb := New()
		// Add few sparse values that should stay as array
		values := []uint32{1, 100, 1000, 10000, 50000}
		for _, v := range values {
			rb.Set(v)
		}

		rb.Optimize()
		c, exists := rb.findContainer(0)
		assert.True(t, exists)
		assert.Equal(t, typeArray, c.Type)
	})

	t.Run("bitmap_stays_bitmap", func(t *testing.T) {
		rb := New()
		// Add many sparse values that should stay as bitmap
		for i := 0; i < 10000; i++ {
			rb.Set(uint32(i * 5)) // Every 5th value
		}

		rb.Optimize()
		c, exists := rb.findContainer(0)
		assert.True(t, exists)
		assert.Equal(t, typeBitmap, c.Type)
	})

	t.Run("run_stays_run", func(t *testing.T) {
		rb := New()
		// Add consecutive values that should become/stay run
		for i := 5000; i <= 15000; i++ {
			rb.Set(uint32(i))
		}

		rb.Optimize()
		c, exists := rb.findContainer(0)
		assert.True(t, exists)
		assert.Equal(t, typeRun, c.Type)

		// Optimize again - should still be run
		rb.Optimize()
		c, exists = rb.findContainer(0)
		assert.True(t, exists)
		assert.Equal(t, typeRun, c.Type)
	})
}

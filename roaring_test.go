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

func TestCopyOnWrite(t *testing.T) {
	t.Run("basic_cow_sharing", func(t *testing.T) {
		// Create original bitmap
		original := New()
		for i := 0; i < 1000; i++ {
			original.Set(uint32(i))
		}

		// Clone using COW
		clone := original.Clone(nil)

		// Verify both have the same content
		assert.Equal(t, original.Count(), clone.Count())

		// Verify they share the same underlying data initially
		origContainer, exists := original.findContainer(0)
		assert.True(t, exists)
		cloneContainer, exists := clone.findContainer(0)
		assert.True(t, exists)

		// Both should be marked as shared
		assert.True(t, origContainer.shared, "Original container should be shared")
		assert.True(t, cloneContainer.shared, "Clone container should be shared")

		// Data slices should point to the same underlying array
		assert.Equal(t, &origContainer.Data[0], &cloneContainer.Data[0], "Should share data pointers")
	})

	t.Run("cow_trigger_on_set", func(t *testing.T) {
		original := New()
		for i := 0; i < 1000; i++ {
			original.Set(uint32(i))
		}
		clone := original.Clone(nil)

		// Verify initial state
		assert.Equal(t, original.Count(), clone.Count())
		origContainer, _ := original.findContainer(0)
		cloneContainer, _ := clone.findContainer(0)
		assert.True(t, origContainer.shared, "Original should be shared initially")
		assert.True(t, cloneContainer.shared, "Clone should be shared initially")

		// Store reference to verify sharing worked
		sharedData := cloneContainer.Data

		// Modify the original - this should trigger COW
		original.Set(1500)

		// After modification, sharing state should change
		assert.False(t, origContainer.shared, "Original should not be shared after modification")
		assert.True(t, cloneContainer.shared, "Clone should still be shared")

		// Clone should still reference the original shared data
		assert.Equal(t, sharedData, cloneContainer.Data, "Clone should still reference shared data")

		// Verify counts and content isolation
		assert.Equal(t, 1001, original.Count())
		assert.Equal(t, 1000, clone.Count())
		assert.False(t, clone.Contains(1500), "Clone should not contain new element")
		assert.True(t, original.Contains(1500), "Original should contain new element")

		// Verify all original elements are still in both
		for i := 0; i < 1000; i++ {
			assert.True(t, original.Contains(uint32(i)), "Original should still contain %d", i)
			assert.True(t, clone.Contains(uint32(i)), "Clone should still contain %d", i)
		}
	})

	t.Run("multiple_clones_sharing", func(t *testing.T) {
		original := New()
		for i := 0; i < 500; i++ {
			original.Set(uint32(i))
		}

		clone1 := original.Clone(nil)
		clone2 := original.Clone(nil)
		clone3 := clone1.Clone(nil)

		// All should share data initially
		origContainer, _ := original.findContainer(0)
		clone1Container, _ := clone1.findContainer(0)
		clone2Container, _ := clone2.findContainer(0)
		clone3Container, _ := clone3.findContainer(0)

		assert.True(t, origContainer.shared)
		assert.True(t, clone1Container.shared)
		assert.True(t, clone2Container.shared)
		assert.True(t, clone3Container.shared)

		// All should point to same data
		dataPtr := &origContainer.Data[0]
		assert.Equal(t, dataPtr, &clone1Container.Data[0])
		assert.Equal(t, dataPtr, &clone2Container.Data[0])
		assert.Equal(t, dataPtr, &clone3Container.Data[0])

		// Store shared data reference
		sharedData := origContainer.Data

		// Modify clone2 - only clone2 should break sharing
		clone2.Set(600) // Should be in same container (600 >> 16 = 0)

		// Verify sharing state after modification
		assert.True(t, origContainer.shared, "Original should still be shared")
		assert.True(t, clone1Container.shared, "Clone1 should still be shared")
		assert.False(t, clone2Container.shared, "Clone2 should not be shared")
		assert.True(t, clone3Container.shared, "Clone3 should still be shared")

		// Original, clone1, and clone3 should still reference shared data
		assert.Equal(t, sharedData, origContainer.Data)
		assert.Equal(t, sharedData, clone1Container.Data)
		assert.Equal(t, sharedData, clone3Container.Data)
		assert.NotEqual(t, sharedData, clone2Container.Data)

		// Verify content
		assert.False(t, original.Contains(600))
		assert.False(t, clone1.Contains(600))
		assert.True(t, clone2.Contains(600))
		assert.False(t, clone3.Contains(600))
	})

	t.Run("cow_with_different_operations", func(t *testing.T) {
		original := New()
		for i := 0; i < 1000; i++ {
			original.Set(uint32(i))
		}

		// Test COW with Remove
		clone1 := original.Clone(nil)
		clone1.Remove(500)
		assert.True(t, original.Contains(500))
		assert.False(t, clone1.Contains(500))

		// Test COW with Filter
		clone2 := original.Clone(nil)
		clone2.Filter(func(x uint32) bool { return x%2 == 0 })
		assert.True(t, original.Contains(501)) // odd number
		assert.False(t, clone2.Contains(501))
		assert.True(t, clone2.Contains(500)) // even number

		// Test COW with Optimize
		clone3 := original.Clone(nil)
		clone3.Optimize()
		// Both should have same content after optimize
		assert.Equal(t, original.Count(), clone3.Count())
		for i := 0; i < 1000; i++ {
			assert.Equal(t, original.Contains(uint32(i)), clone3.Contains(uint32(i)))
		}
	})

	t.Run("cow_with_different_container_types", func(t *testing.T) {
		// Test with array container
		arrayBitmap := New()
		for i := 0; i < 10; i++ {
			arrayBitmap.Set(uint32(i * 100)) // sparse
		}
		arrayClone := arrayBitmap.Clone(nil)
		arrayBitmap.Set(1500)
		assert.False(t, arrayClone.Contains(1500))

		// Test with bitmap container
		bitmapBitmap := New()
		for i := 0; i < 5000; i++ {
			bitmapBitmap.Set(uint32(i * 2)) // dense enough for bitmap
		}
		bitmapClone := bitmapBitmap.Clone(nil)
		bitmapBitmap.Remove(1000)
		assert.True(t, bitmapClone.Contains(1000))

		// Test with run container
		runBitmap := New()
		for i := 10000; i < 20000; i++ {
			runBitmap.Set(uint32(i)) // consecutive for run
		}
		runBitmap.Optimize()
		runClone := runBitmap.Clone(nil)
		runBitmap.Remove(15000)
		assert.True(t, runClone.Contains(15000))
	})
}

func TestCopyOnWriteAnd(t *testing.T) {
	t.Run("and_triggers_cow", func(t *testing.T) {
		original := New()
		for i := 0; i < 1000; i++ {
			original.Set(uint32(i))
		}

		other := New()
		for i := 500; i < 1500; i++ {
			other.Set(uint32(i))
		}

		clone := original.Clone(nil)

		// Verify initial sharing
		origContainer, _ := original.findContainer(0)
		cloneContainer, _ := clone.findContainer(0)
		assert.True(t, origContainer.shared)
		assert.True(t, cloneContainer.shared)
		assert.Equal(t, &origContainer.Data[0], &cloneContainer.Data[0])

		// Perform AND operation - should trigger COW
		original.And(other)

		// Verify COW triggered
		assert.False(t, origContainer.shared, "Original should not be shared after AND")
		assert.True(t, cloneContainer.shared, "Clone should still be shared")
		assert.NotEqual(t, &origContainer.Data[0], &cloneContainer.Data[0], "Data pointers should differ")

		// Verify AND operation worked correctly
		expectedIntersection := 500 // [0,999] ∩ [500,1499] = [500,999]
		assert.Equal(t, expectedIntersection, original.Count())
		assert.Equal(t, 1000, clone.Count(), "Clone should have original count")

		// Verify specific values
		assert.True(t, original.Contains(500), "Original should contain intersection start")
		assert.True(t, original.Contains(999), "Original should contain intersection end")
		assert.False(t, original.Contains(100), "Original should not contain pre-intersection")
		assert.False(t, original.Contains(1100), "Original should not contain post-intersection")

		assert.True(t, clone.Contains(100), "Clone should contain all original values")
		assert.True(t, clone.Contains(999), "Clone should contain all original values")
	})

	t.Run("multiple_clones_and_operations", func(t *testing.T) {
		base := New()
		for i := 0; i < 2000; i++ {
			base.Set(uint32(i))
		}

		clone1 := base.Clone(nil)
		clone2 := base.Clone(nil)
		clone3 := base.Clone(nil)

		// Create different AND operands
		mask1 := New()
		for i := 0; i < 1000; i++ {
			mask1.Set(uint32(i))
		}

		mask2 := New()
		for i := 500; i < 1500; i++ {
			mask2.Set(uint32(i))
		}

		// Perform different operations
		clone1.And(mask1) // [0, 999]
		clone2.And(mask2) // [500, 1499]

		// Verify all have different content
		assert.Equal(t, 2000, base.Count(), "Base should be unchanged")
		assert.Equal(t, 1000, clone1.Count(), "Clone1 should have 1000 elements")
		assert.Equal(t, 1000, clone2.Count(), "Clone2 should have 1000 elements")
		assert.Equal(t, 2000, clone3.Count(), "Clone3 should be unchanged")

		// Verify no cross-contamination
		assert.True(t, base.Contains(1900))
		assert.False(t, clone1.Contains(1900))
		assert.False(t, clone2.Contains(1900))
		assert.True(t, clone3.Contains(1900))

		assert.True(t, clone1.Contains(100))
		assert.False(t, clone2.Contains(100))
		assert.True(t, clone2.Contains(1400))
		assert.False(t, clone1.Contains(1400))
	})
}

func TestCopyOnWriteEdgeCases(t *testing.T) {
	t.Run("empty_bitmap_cow", func(t *testing.T) {
		empty := New()
		clone := empty.Clone(nil)

		assert.Equal(t, 0, empty.Count())
		assert.Equal(t, 0, clone.Count())

		// Modifying empty should work
		empty.Set(100)
		assert.True(t, empty.Contains(100))
		assert.False(t, clone.Contains(100))
		assert.Equal(t, 1, empty.Count())
		assert.Equal(t, 0, clone.Count())
	})

	t.Run("single_element_cow", func(t *testing.T) {
		single := New()
		single.Set(42)
		clone := single.Clone(nil)

		origContainer, _ := single.findContainer(0)
		cloneContainer, _ := clone.findContainer(0)
		assert.True(t, origContainer.shared)
		assert.True(t, cloneContainer.shared)

		// Remove from original
		single.Remove(42)
		assert.False(t, single.Contains(42))
		assert.True(t, clone.Contains(42))
	})

	t.Run("cow_after_clear", func(t *testing.T) {
		original := New()
		for i := 0; i < 1000; i++ {
			original.Set(uint32(i))
		}
		clone := original.Clone(nil)

		// Clear original
		original.Clear()
		assert.Equal(t, 0, original.Count())
		assert.Equal(t, 1000, clone.Count())
		assert.True(t, clone.Contains(500))
	})

	t.Run("clone_of_clone_chains", func(t *testing.T) {
		root := New()
		for i := 0; i < 100; i++ {
			root.Set(uint32(i))
		}

		// Create chain of clones
		level1 := root.Clone(nil)
		level2 := level1.Clone(nil)
		level3 := level2.Clone(nil)
		level4 := level3.Clone(nil)

		// All should share data
		rootContainer, _ := root.findContainer(0)
		l1Container, _ := level1.findContainer(0)
		l2Container, _ := level2.findContainer(0)
		l3Container, _ := level3.findContainer(0)
		l4Container, _ := level4.findContainer(0)

		dataPtr := &rootContainer.Data[0]
		assert.Equal(t, dataPtr, &l1Container.Data[0])
		assert.Equal(t, dataPtr, &l2Container.Data[0])
		assert.Equal(t, dataPtr, &l3Container.Data[0])
		assert.Equal(t, dataPtr, &l4Container.Data[0])

		// Modify middle of chain
		level2.Set(200)

		// level2 should break sharing, others should still share
		assert.True(t, rootContainer.shared)
		assert.True(t, l1Container.shared)
		assert.False(t, l2Container.shared)
		assert.True(t, l3Container.shared)
		assert.True(t, l4Container.shared)

		// Verify content isolation
		assert.False(t, root.Contains(200))
		assert.False(t, level1.Contains(200))
		assert.True(t, level2.Contains(200))
		assert.False(t, level3.Contains(200))
		assert.False(t, level4.Contains(200))
	})

	t.Run("cow_with_optimization_changes", func(t *testing.T) {
		// Create bitmap that will change container types during optimization
		rb := New()

		// Start with array (sparse)
		for i := 0; i < 10; i++ {
			rb.Set(uint32(i * 1000))
		}
		container, _ := rb.findContainer(0)
		assert.Equal(t, typeArray, container.Type)

		clone := rb.Clone(nil)

		// Add enough to trigger bitmap conversion
		for i := 0; i < 3000; i++ {
			rb.Set(uint32(i * 2))
		}
		rb.Optimize()

		container, _ = rb.findContainer(0)
		assert.Equal(t, typeBitmap, container.Type)

		// Clone should still be array with original content
		cloneContainer, _ := clone.findContainer(0)
		assert.Equal(t, typeArray, cloneContainer.Type)
		assert.Equal(t, 10, clone.Count())
		assert.True(t, clone.Contains(5000))
		assert.False(t, clone.Contains(10)) // rb added this
	})

	t.Run("stress_many_clones", func(t *testing.T) {
		base := New()
		for i := 0; i < 1000; i++ {
			base.Set(uint32(i))
		}

		// Create many clones
		clones := make([]*Bitmap, 50)
		for i := range clones {
			clones[i] = base.Clone(nil)
		}

		// All should share data initially
		baseContainer, _ := base.findContainer(0)
		baseDataPtr := &baseContainer.Data[0]

		for i, clone := range clones {
			container, exists := clone.findContainer(0)
			assert.True(t, exists, "Clone %d should have container", i)
			assert.True(t, container.shared, "Clone %d should be shared", i)
			assert.Equal(t, baseDataPtr, &container.Data[0], "Clone %d should share data", i)
		}

		// Modify every 5th clone
		for i := 0; i < len(clones); i += 5 {
			clones[i].Set(uint32(2000 + i))
		}

		// Only modified clones should break sharing
		for i, clone := range clones {
			container, _ := clone.findContainer(0)
			if i%5 == 0 {
				assert.False(t, container.shared, "Modified clone %d should not be shared", i)
				assert.True(t, clone.Contains(uint32(2000+i)), "Clone %d should contain new element", i)
			} else {
				assert.True(t, container.shared, "Unmodified clone %d should still be shared", i)
				assert.False(t, clone.Contains(uint32(2000+i)), "Clone %d should not contain other's element", i)
			}
		}
	})

	t.Run("cow_preserves_container_metadata", func(t *testing.T) {
		original := New()
		for i := 0; i < 100; i++ {
			original.Set(uint32(i))
		}

		// Access container to increment Call count
		container, _ := original.findContainer(0)
		originalCall := container.Call
		originalSize := container.Size
		originalType := container.Type

		clone := original.Clone(nil)
		cloneContainer, _ := clone.findContainer(0)

		// Metadata should be preserved in clone
		assert.Equal(t, originalCall, cloneContainer.Call)
		assert.Equal(t, originalSize, cloneContainer.Size)
		assert.Equal(t, originalType, cloneContainer.Type)

		// After COW, metadata should still be preserved but independent
		original.Set(200)

		assert.Equal(t, originalCall, cloneContainer.Call, "Clone metadata should be preserved")
		assert.Equal(t, originalSize, cloneContainer.Size, "Clone size should be preserved")
		assert.Equal(t, originalType, cloneContainer.Type, "Clone type should be preserved")
	})
}

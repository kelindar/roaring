// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

package roaring

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasicOperations(t *testing.T) {
	rb := New()

	// Test empty bitmap
	assert.Equal(t, 0, rb.Count())
	assert.False(t, rb.Contains(123))

	// Test setting bits
	rb.Set(42)
	assert.True(t, rb.Contains(42))
	assert.False(t, rb.Contains(41))
	assert.Equal(t, 1, rb.Count())

	// Test setting same bit again
	rb.Set(42)
	assert.True(t, rb.Contains(42))
	assert.Equal(t, 1, rb.Count())

	// Test setting more bits
	rb.Set(100)
	rb.Set(1000)
	rb.Set(10000)
	assert.Equal(t, 4, rb.Count())
	assert.True(t, rb.Contains(100))
	assert.True(t, rb.Contains(1000))
	assert.True(t, rb.Contains(10000))

	// Test removing bits
	rb.Remove(42)
	assert.False(t, rb.Contains(42))
	assert.Equal(t, 3, rb.Count())

	// Test removing non-existent bit
	rb.Remove(999)
	assert.Equal(t, 3, rb.Count())

	// Test clear
	rb.Clear()
	assert.Equal(t, 0, rb.Count())
	assert.False(t, rb.Contains(100))
}

func TestOperationsComprehensive(t *testing.T) {
	rb := New()

	// Add many values across multiple containers
	values := []uint32{0, 1, 65535, 65536, 131072, 131073, 4294967295}
	for _, v := range values {
		rb.Set(v)
	}

	assert.Equal(t, len(values), rb.Count())

	// Test all values are present
	for _, v := range values {
		assert.True(t, rb.Contains(v), "Value %d should be present", v)
	}

	// Test some values that shouldn't be present
	nonValues := []uint32{2, 65534, 65537, 131071, 131074}
	for _, v := range nonValues {
		assert.False(t, rb.Contains(v), "Value %d should not be present", v)
	}

	// Remove some values
	toRemove := []uint32{1, 65536, 4294967295}
	for _, v := range toRemove {
		rb.Remove(v)
		assert.False(t, rb.Contains(v), "Value %d should be removed", v)
	}

	expectedCount := len(values) - len(toRemove)
	assert.Equal(t, expectedCount, rb.Count())

	// Verify remaining values
	remaining := []uint32{0, 65535, 131072, 131073}
	for _, v := range remaining {
		assert.True(t, rb.Contains(v), "Value %d should still be present", v)
	}
}

func TestTransitions(t *testing.T) {
	t.Run("array_to_bitmap", func(t *testing.T) {
		rb := New()

		// Add enough values to trigger array->bitmap transition
		for i := 0; i < 5000; i++ {
			rb.Set(uint32(i))
		}

		// Should have bitmap container now
		assert.Equal(t, 5000, rb.Count())
		assert.True(t, rb.Contains(0))
		assert.True(t, rb.Contains(4999))
		assert.False(t, rb.Contains(5000))
	})

	t.Run("bitmap_to_array", func(t *testing.T) {
		rb := New()

		// Create bitmap container
		for i := 0; i < 5000; i++ {
			rb.Set(uint32(i))
		}

		// Remove most values to trigger bitmap->array transition
		for i := 100; i < 5000; i++ {
			rb.Remove(uint32(i))
		}

		// Should have array container now with 100 values
		assert.Equal(t, 100, rb.Count())
		for i := 0; i < 100; i++ {
			assert.True(t, rb.Contains(uint32(i)))
		}
		assert.False(t, rb.Contains(100))
	})

	t.Run("run_transitions", func(t *testing.T) {
		rb := New()

		// Create run container
		for i := 1000; i <= 2000; i++ {
			rb.Set(uint32(i))
		}

		assert.Equal(t, 1001, rb.Count())
		assert.True(t, rb.Contains(1000))
		assert.True(t, rb.Contains(2000))
		assert.False(t, rb.Contains(999))
		assert.False(t, rb.Contains(2001))

		// Sparse out the data to trigger run->array transition
		for i := 1000; i <= 2000; i += 10 {
			rb.Remove(uint32(i))
		}

		// Should be mostly sparse now
		assert.True(t, rb.Contains(1001))
		assert.False(t, rb.Contains(1000))
		assert.True(t, rb.Contains(1999))
		assert.False(t, rb.Contains(2000))
	})
}

func TestContainerConversions(t *testing.T) {
	t.Run("empty_to_array", func(t *testing.T) {
		rb := New()
		rb.Set(100)
		assert.Equal(t, 1, rb.Count())
		assert.True(t, rb.Contains(100))
	})

	t.Run("array_growth", func(t *testing.T) {
		rb := New()
		values := []uint32{1, 10, 100, 1000, 10000}
		for _, v := range values {
			rb.Set(v)
		}
		assert.Equal(t, len(values), rb.Count())
		for _, v := range values {
			assert.True(t, rb.Contains(v))
		}
	})

	t.Run("dense_bitmap", func(t *testing.T) {
		rb := New()
		// Create dense bitmap
		for i := 0; i < 50000; i++ {
			rb.Set(uint32(i))
		}
		assert.Equal(t, 50000, rb.Count())
		assert.True(t, rb.Contains(0))
		assert.True(t, rb.Contains(49999))
		assert.False(t, rb.Contains(50000))
	})

	t.Run("runs_creation", func(t *testing.T) {
		rb := New()
		// Create long runs
		for i := 0; i < 10000; i++ {
			rb.Set(uint32(i))
		}
		for i := 20000; i < 30000; i++ {
			rb.Set(uint32(i))
		}
		assert.Equal(t, 20000, rb.Count())
		assert.True(t, rb.Contains(5000))
		assert.True(t, rb.Contains(25000))
		assert.False(t, rb.Contains(15000))
	})
}

func TestContainerEdgeCases(t *testing.T) {
	t.Run("container_boundaries", func(t *testing.T) {
		rb := New()

		// Test values at container boundaries (every 65536)
		boundaries := []uint32{0, 65535, 65536, 131071, 131072, 196607, 196608}
		for _, v := range boundaries {
			rb.Set(v)
		}

		for _, v := range boundaries {
			assert.True(t, rb.Contains(v), "Boundary value %d should be present", v)
		}

		// Test adjacent values
		adjacent := []uint32{1, 65534, 65537, 131070, 131073, 196606, 196609}
		for _, v := range adjacent {
			assert.False(t, rb.Contains(v), "Adjacent value %d should not be present", v)
		}
	})

	t.Run("max_uint32", func(t *testing.T) {
		rb := New()
		maxVal := uint32(4294967295)
		rb.Set(maxVal)
		assert.True(t, rb.Contains(maxVal))
		assert.Equal(t, 1, rb.Count())

		rb.Remove(maxVal)
		assert.False(t, rb.Contains(maxVal))
		assert.Equal(t, 0, rb.Count())
	})

	t.Run("sparse_across_containers", func(t *testing.T) {
		rb := New()
		// Add one value per container across many containers
		for i := 0; i < 100; i++ {
			rb.Set(uint32(i * 65536))
		}

		assert.Equal(t, 100, rb.Count())
		for i := 0; i < 100; i++ {
			assert.True(t, rb.Contains(uint32(i*65536)))
			if i < 99 {
				assert.False(t, rb.Contains(uint32(i*65536+1)))
			}
		}
	})
}

func TestStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	t.Run("random_operations", func(t *testing.T) {
		rb := New()
		reference := make(map[uint32]bool)

		rand.Seed(42)
		const numOps = 10000

		for i := 0; i < numOps; i++ {
			value := rand.Uint32() % 1000000 // Limit range for reasonable test time
			op := rand.Intn(3)

			switch op {
			case 0: // Set
				rb.Set(value)
				reference[value] = true
			case 1: // Remove
				rb.Remove(value)
				delete(reference, value)
			case 2: // Contains check
				expected := reference[value]
				actual := rb.Contains(value)
				assert.Equal(t, expected, actual, "Contains mismatch for value %d", value)
			}
		}

		// Final verification
		assert.Equal(t, len(reference), rb.Count(), "Count mismatch")

		for value := range reference {
			assert.True(t, rb.Contains(value), "Missing value %d", value)
		}
	})

	t.Run("sequential_large", func(t *testing.T) {
		rb := New()
		const numValues = 100000

		// Sequential add
		for i := 0; i < numValues; i++ {
			rb.Set(uint32(i))
		}
		assert.Equal(t, numValues, rb.Count())

		// Random contains checks
		rand.Seed(123)
		for i := 0; i < 1000; i++ {
			value := rand.Uint32() % (numValues * 2)
			expected := value < numValues
			actual := rb.Contains(value)
			assert.Equal(t, expected, actual, "Contains check failed for %d", value)
		}

		// Sequential remove
		for i := 0; i < numValues; i += 2 {
			rb.Remove(uint32(i))
		}
		assert.Equal(t, numValues/2, rb.Count())

		// Verify remaining values
		for i := 1; i < numValues; i += 2 {
			assert.True(t, rb.Contains(uint32(i)), "Odd value %d should remain", i)
		}
		for i := 0; i < numValues; i += 2 {
			assert.False(t, rb.Contains(uint32(i)), "Even value %d should be removed", i)
		}
	})
}

func TestContainerOptimization(t *testing.T) {
	t.Run("optimize_array", func(t *testing.T) {
		rb := New()
		for i := 0; i < 100; i++ {
			rb.Set(uint32(i * 100)) // Sparse array
		}
		rb.Optimize()
		assert.Equal(t, 100, rb.Count())
	})

	t.Run("optimize_bitmap", func(t *testing.T) {
		rb := New()
		for i := 0; i < 50000; i++ {
			rb.Set(uint32(i)) // Dense bitmap
		}
		rb.Optimize()
		assert.Equal(t, 50000, rb.Count())
	})

	t.Run("optimize_runs", func(t *testing.T) {
		rb := New()
		// Create potential run containers
		for i := 0; i < 10000; i++ {
			rb.Set(uint32(i))
		}
		for i := 20000; i < 30000; i++ {
			rb.Set(uint32(i))
		}
		rb.Optimize()
		assert.Equal(t, 20000, rb.Count())
	})
}

func TestClone(t *testing.T) {
	t.Run("clone_empty", func(t *testing.T) {
		original := New()
		clone := original.Clone(nil)

		assert.Equal(t, 0, original.Count())
		assert.Equal(t, 0, clone.Count())

		// Modify original
		original.Set(42)
		assert.True(t, original.Contains(42))
		assert.False(t, clone.Contains(42))
	})

	t.Run("clone_simple", func(t *testing.T) {
		original := New()
		for i := 0; i < 1000; i++ {
			original.Set(uint32(i))
		}

		clone := original.Clone(nil)
		assert.Equal(t, original.Count(), clone.Count())

		// Both should have same values
		for i := 0; i < 1000; i++ {
			assert.True(t, original.Contains(uint32(i)))
			assert.True(t, clone.Contains(uint32(i)))
		}

		// Modify original
		original.Set(2000)
		assert.True(t, original.Contains(2000))
		assert.False(t, clone.Contains(2000))
		assert.Equal(t, 1001, original.Count())
		assert.Equal(t, 1000, clone.Count())
	})

	t.Run("clone_into_existing", func(t *testing.T) {
		original := New()
		for i := 0; i < 100; i++ {
			original.Set(uint32(i))
		}

		existing := New()
		existing.Set(999)

		clone := original.Clone(existing)
		assert.Equal(t, original.Count(), clone.Count())
		assert.False(t, clone.Contains(999)) // Should be replaced, not merged

		for i := 0; i < 100; i++ {
			assert.True(t, clone.Contains(uint32(i)))
		}
	})
}

func TestMinMax(t *testing.T) {
	type testCase struct {
		name string
		cnr  *container
		val  uint32
		has  bool
	}

	t.Run("min", func(t *testing.T) {
		for _, tc := range []testCase{
			{"arr empty", newArr(), 0, false},
			{"arr single", newArr(42), 42, true},
			{"arr multiple", newArr(10, 20, 30), 10, true},
			{"arr boundary", newArr(0, 65535), 0, true},
			{"bmp empty", newBmp(), 0, false},
			{"bmp single", newBmp(42), 42, true},
			{"bmp multiple", newBmp(10, 20, 30), 10, true},
			{"bmp boundary", newBmp(0, 65535), 0, true},
			{"run empty", newRun(), 0, false},
			{"run single", newRun(42), 42, true},
			{"run multiple", newRun(10, 11, 12, 20, 21, 22), 10, true},
			{"run boundary", newRun(0, 65535), 0, true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				rb, _ := bitmapWith(tc.cnr)
				min, minOk := rb.Min()
				assert.Equal(t, tc.has, minOk, "min() ok result")
				assert.Equal(t, tc.val, min, "min() value")
			})
		}
	})

	t.Run("max", func(t *testing.T) {
		for _, tc := range []testCase{
			{"arr empty", newArr(), 0, false},
			{"arr single", newArr(42), 42, true},
			{"arr multiple", newArr(10, 20, 30), 30, true},
			{"arr boundary", newArr(0, 65535), 65535, true},
			{"bmp empty", newBmp(), 0, false},
			{"bmp single", newBmp(42), 42, true},
			{"bmp multiple", newBmp(10, 20, 30), 30, true},
			{"bmp boundary", newBmp(0, 65535), 65535, true},
			{"run empty", newRun(), 0, false},
			{"run single", newRun(42), 42, true},
			{"run multiple", newRun(10, 11, 12, 20, 21, 22), 22, true},
			{"run boundary", newRun(0, 65535), 65535, true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				rb, _ := bitmapWith(tc.cnr)
				max, maxOk := rb.Max()
				assert.Equal(t, tc.has, maxOk, "max() ok result")
				assert.Equal(t, tc.val, max, "max() value")
			})
		}
	})

	t.Run("minZero", func(t *testing.T) {
		for _, tc := range []testCase{
			{"arr empty", newArr(), 0, true},
			{"arr single", newArr(42), 0, true},
			{"arr multiple", newArr(10, 20, 30), 0, true},
			{"arr boundary", newArr(0, 65535), 1, true},
			{"bmp empty", newBmp(), 0, true},
			{"bmp single", newBmp(42), 0, true},
			{"bmp multiple", newBmp(10, 20, 30), 0, true},
			{"bmp boundary", newBmp(0, 65535), 1, true},
			{"run empty", newRun(), 0, true},
			{"run single", newRun(42), 0, true},
			{"run multiple", newRun(10, 11, 12, 20, 21, 22), 0, true},
			{"run boundary", newRun(0, 65535), 1, true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				rb, _ := bitmapWith(tc.cnr)
				minZero, minZeroOk := rb.MinZero()
				assert.Equal(t, tc.has, minZeroOk, "minZero() ok result")
				assert.Equal(t, tc.val, minZero, "minZero() value")
			})
		}
	})

	t.Run("maxZero", func(t *testing.T) {
		for _, tc := range []testCase{
			{"arr empty", newArr(), 4294967295, true},
			{"arr single", newArr(42), 4294967295, true},
			{"arr multiple", newArr(10, 20, 30), 4294967295, true},
			{"arr boundary", newArr(0, 65535), 4294967295, true},
			{"bmp empty", newBmp(), 4294967295, true},
			{"bmp single", newBmp(42), 4294967295, true},
			{"bmp multiple", newBmp(10, 20, 30), 4294967295, true},
			{"bmp boundary", newBmp(0, 65535), 4294967295, true},
			{"run empty", newRun(), 4294967295, true},
			{"run single", newRun(42), 4294967295, true},
			{"run multiple", newRun(10, 11, 12, 20, 21, 22), 4294967295, true},
			{"run boundary", newRun(0, 65535), 4294967295, true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				rb, _ := bitmapWith(tc.cnr)
				maxZero, maxZeroOk := rb.MaxZero()
				assert.Equal(t, tc.has, maxZeroOk, "maxZero() ok result")
				assert.Equal(t, tc.val, maxZero, "maxZero() value")
			})
		}
	})

}

func TestRunContainerGaps(t *testing.T) {
	tc := []struct {
		name       string
		values     []uint32
		minZeroVal uint32
		minZeroOk  bool
		maxZeroVal uint32
		maxZeroOk  bool
	}{
		{"gap before first container", []uint32{131072, 131073, 131074}, 0, true, 4294967295, true},
		{"gap between containers", []uint32{65536, 65537, 196608, 196609}, 0, true, 4294967295, true},
		{"gap after last container", []uint32{100, 101, 102}, 0, true, 4294967295, true},
		{"consecutive containers", []uint32{65534, 65535, 65536, 65537}, 0, true, 4294967295, true},
		{"container boundary gaps", []uint32{65535, 131072}, 0, true, 4294967295, true},
		{"multiple gaps", []uint32{65536, 131072, 196608}, 0, true, 4294967295, true},
		{"single value per container", []uint32{0, 65536, 131072}, 1, true, 4294967295, true},
		{"runs spanning containers", []uint32{65534, 65535, 65536, 65537, 131070, 131071, 131072}, 0, true, 4294967295, true},
		{"gap at container 0 boundary", []uint32{1, 2, 3, 65537, 65538}, 0, true, 4294967295, true},
		{"gap at container end boundary", []uint32{65533, 65534, 131073, 131074}, 0, true, 4294967295, true},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			rb := New()
			for _, v := range tt.values {
				rb.Set(v)
			}

			// Force optimization to create run containers
			rb.Optimize()

			minZero, minZeroOk := rb.MinZero()
			maxZero, maxZeroOk := rb.MaxZero()

			assert.Equal(t, tt.minZeroOk, minZeroOk, "MinZero() ok result")
			assert.Equal(t, tt.maxZeroOk, maxZeroOk, "MaxZero() ok result")

			if tt.minZeroOk {
				assert.Equal(t, tt.minZeroVal, minZero, "MinZero() value")
				assert.False(t, rb.Contains(minZero), "MinZero position should be unset")
			}
			if tt.maxZeroOk {
				assert.Equal(t, tt.maxZeroVal, maxZero, "MaxZero() value")
				assert.False(t, rb.Contains(maxZero), "MaxZero position should be unset")
			}
		})
	}
}

// TestRunMaxZeroBackwardSearch tests the backward search logic in runMaxZero
// that finds the last gap between runs within a single container
func TestRunMaxZeroBackwardSearch(t *testing.T) {
	tc := []struct {
		name       string
		runs       [][2]uint16 // [start, end] pairs for runs
		maxZeroVal uint16
		maxZeroOk  bool
	}{
		{
			name:       "single gap ending at max - should find gap",
			runs:       [][2]uint16{{10, 12}, {20, 65535}},
			maxZeroVal: 19, // Last position in gap between runs
			maxZeroOk:  true,
		},
		{
			name:       "multiple gaps ending at max - should find last gap",
			runs:       [][2]uint16{{5, 7}, {10, 12}, {20, 65535}},
			maxZeroVal: 19, // Last position in the rightmost gap (between runs 2 and 3)
			maxZeroOk:  true,
		},
		{
			name:       "three gaps ending at max - should find rightmost",
			runs:       [][2]uint16{{5, 6}, {10, 11}, {15, 16}, {25, 65535}},
			maxZeroVal: 24, // Last position in gap between runs 3 and 4
			maxZeroOk:  true,
		},
		{
			name:       "gaps of different sizes ending at max - should find last gap",
			runs:       [][2]uint16{{10, 15}, {20, 25}, {35, 65535}},
			maxZeroVal: 34, // Last position in gap between runs 2 and 3
			maxZeroOk:  true,
		},
		{
			name:       "no gaps between runs ending at max - should find before first",
			runs:       [][2]uint16{{10, 15}, {16, 20}, {21, 65535}},
			maxZeroVal: 9, // Before first run
			maxZeroOk:  true,
		},
		{
			name:       "multiple gaps, search backwards for rightmost",
			runs:       [][2]uint16{{0, 5}, {10, 15}, {20, 25}, {30, 65535}},
			maxZeroVal: 29, // Last position in gap between runs 3 and 4 (rightmost gap)
			maxZeroOk:  true,
		},
		{
			name:       "gap pattern ending at max - should find second-to-last gap",
			runs:       [][2]uint16{{0, 10}, {15, 20}, {25, 30}, {35, 65535}},
			maxZeroVal: 34, // Last position in gap between runs 3 and 4
			maxZeroOk:  true,
		},
		{
			name:       "no gaps at all - starts at 0 ends at max",
			runs:       [][2]uint16{{0, 65535}},
			maxZeroVal: 0, // No gaps found
			maxZeroOk:  false,
		},
		{
			name:       "consecutive runs from 0 to max - no gaps",
			runs:       [][2]uint16{{0, 20}, {21, 40}, {41, 65535}},
			maxZeroVal: 0, // No gaps found
			maxZeroOk:  false,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			// Create a run container directly
			c := &container{
				Type: typeRun,
				Size: 0,
				Data: make([]uint16, 0),
			}

			// Add runs and calculate size
			for _, run := range tt.runs {
				c.Data = append(c.Data, run[0], run[1])
				c.Size += uint32(run[1] - run[0] + 1)
			}

			maxZero, maxZeroOk := c.maxZero()

			assert.Equal(t, tt.maxZeroOk, maxZeroOk, "maxZero() ok result")
			if tt.maxZeroOk {
				assert.Equal(t, tt.maxZeroVal, maxZero, "maxZero() value")
				assert.False(t, c.contains(maxZero), "maxZero position should be unset")
			}
		})
	}
}

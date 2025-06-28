// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

package roaring

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRange(t *testing.T) {
	tests := []struct {
		name string
		gen  dataGen
	}{
		{"empty", func() ([]uint32, string) { return []uint32{}, "emp" }},
		{"single", func() ([]uint32, string) { return []uint32{42}, "sgl" }},
		{"sequential", genSeq(1000, 0)},
		{"random", genRand(1000, 100000)},
		{"sparse", genSparse(100)},
		{"dense", genDense(1000)},
		{"boundary", genBoundary()},
		{"mixed", genMixed()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, _ := tt.gen()
			our, ref := testPair(data)

			// Test Range output matches reference
			var ourValues, refValues []uint32
			our.Range(func(x uint32) bool { ourValues = append(ourValues, x); return true })
			ref.Range(func(x uint32) { refValues = append(refValues, x) })

			assert.Equal(t, refValues, ourValues)
		})
	}
}

func TestFilter(t *testing.T) {
	t.Run("filter_even_numbers", func(t *testing.T) {
		rb := New()

		// Add values 0-99
		for i := 0; i < 100; i++ {
			rb.Set(uint32(i))
		}

		// Filter to keep only even numbers
		rb.Filter(func(x uint32) bool {
			return x%2 == 0
		})

		// Verify only even numbers remain
		assert.Equal(t, 50, rb.Count())
		for i := 0; i < 100; i += 2 {
			assert.True(t, rb.Contains(uint32(i)), "Even number %d should be present", i)
		}
		for i := 1; i < 100; i += 2 {
			assert.False(t, rb.Contains(uint32(i)), "Odd number %d should be removed", i)
		}
	})

	t.Run("filter_empty_bitmap", func(t *testing.T) {
		rb := New()

		// Filter empty bitmap - should not panic
		rb.Filter(func(x uint32) bool {
			return x%2 == 0
		})

		assert.Equal(t, 0, rb.Count())
	})

	t.Run("filter_all_pass", func(t *testing.T) {
		rb := New()
		values := []uint32{1, 5, 10, 100, 1000, 10000}

		for _, v := range values {
			rb.Set(v)
		}

		originalCount := rb.Count()

		// Filter that passes everything
		rb.Filter(func(x uint32) bool {
			return true
		})

		// Nothing should be removed
		assert.Equal(t, originalCount, rb.Count())
		for _, v := range values {
			assert.True(t, rb.Contains(v))
		}
	})

	t.Run("filter_all_fail", func(t *testing.T) {
		rb := New()
		values := []uint32{1, 5, 10, 100, 1000, 10000}

		for _, v := range values {
			rb.Set(v)
		}

		// Filter that rejects everything
		rb.Filter(func(x uint32) bool {
			return false
		})

		// Everything should be removed
		assert.Equal(t, 0, rb.Count())
		for _, v := range values {
			assert.False(t, rb.Contains(v))
		}
	})

	t.Run("filter_range_predicate", func(t *testing.T) {
		rb := New()

		// Add values 0-199
		for i := 0; i < 200; i++ {
			rb.Set(uint32(i))
		}

		// Filter to keep only values in range [50, 150)
		rb.Filter(func(x uint32) bool {
			return x >= 50 && x < 150
		})

		// Verify correct range remains
		assert.Equal(t, 100, rb.Count())
		for i := 0; i < 50; i++ {
			assert.False(t, rb.Contains(uint32(i)))
		}
		for i := 50; i < 150; i++ {
			assert.True(t, rb.Contains(uint32(i)))
		}
		for i := 150; i < 200; i++ {
			assert.False(t, rb.Contains(uint32(i)))
		}
	})

	t.Run("filter_multiple_containers", func(t *testing.T) {
		rb := New()

		// Add values across multiple containers
		values := []uint32{
			100,    // Container 0
			65636,  // Container 1
			131172, // Container 2
			196708, // Container 3
		}

		for _, v := range values {
			rb.Set(v)
		}

		// Filter to keep only values > 100000
		rb.Filter(func(x uint32) bool {
			return x > 100000
		})

		// Verify correct values remain
		assert.Equal(t, 2, rb.Count())
		assert.False(t, rb.Contains(100))
		assert.False(t, rb.Contains(65636))
		assert.True(t, rb.Contains(131172))
		assert.True(t, rb.Contains(196708))
	})

	t.Run("filter_with_optimization", func(t *testing.T) {
		rb := New()

		// Create array container
		for i := 0; i < 10; i++ {
			rb.Set(uint32(i))
		}

		// Create bitmap container
		for i := 0; i < 5000; i++ {
			rb.Set(uint32(100000 + i*2)) // Sparse pattern
		}

		// Create run container (consecutive values)
		for i := 200000; i < 201000; i++ {
			rb.Set(uint32(i))
		}
		rb.Optimize()

		originalCount := rb.Count()

		// Filter to keep values divisible by 5
		rb.Filter(func(x uint32) bool {
			return x%5 == 0
		})

		// Verify filtering worked across all container types
		assert.True(t, rb.Count() < originalCount)

		// Check some specific values
		assert.True(t, rb.Contains(0))  // 0 % 5 == 0
		assert.True(t, rb.Contains(5))  // 5 % 5 == 0
		assert.False(t, rb.Contains(1)) // 1 % 5 != 0
		assert.False(t, rb.Contains(3)) // 3 % 5 != 0

		// Verify all remaining values pass the predicate
		rb.Range(func(x uint32) bool {
			assert.Equal(t, uint32(0), x%5, "Value %d should be divisible by 5", x)
			return true
		})
	})

	t.Run("filter_boundary_values", func(t *testing.T) {
		rb := New()

		// Add boundary values
		boundaries := []uint32{0, 65535, 65536, 131071, 131072, 4294967295}
		for _, v := range boundaries {
			rb.Set(v)
		}

		// Filter to keep only values >= 65536
		rb.Filter(func(x uint32) bool {
			return x >= 65536
		})

		// Verify correct boundaries remain
		assert.False(t, rb.Contains(0))
		assert.False(t, rb.Contains(65535))
		assert.True(t, rb.Contains(65536))
		assert.True(t, rb.Contains(131071))
		assert.True(t, rb.Contains(131072))
		assert.True(t, rb.Contains(4294967295))
	})
}

func TestRangeAndFilterConsistency(t *testing.T) {
	t.Run("range_after_filter", func(t *testing.T) {
		rb := New()

		// Add random values
		original := []uint32{1, 3, 5, 7, 9, 11, 13, 15, 17, 19, 100, 200, 300}
		for _, v := range original {
			rb.Set(v)
		}

		// Filter to keep only values > 10
		rb.Filter(func(x uint32) bool {
			return x > 10
		})

		// Use Range to collect remaining values
		var remaining []uint32
		rb.Range(func(x uint32) bool {
			remaining = append(remaining, x)
			return true
		})

		// Sort both slices for comparison
		sort.Slice(remaining, func(i, j int) bool { return remaining[i] < remaining[j] })

		expected := []uint32{11, 13, 15, 17, 19, 100, 200, 300}
		assert.Equal(t, expected, remaining)

		// Verify Count matches Range results
		assert.Equal(t, len(remaining), rb.Count())
	})

	t.Run("multiple_filters", func(t *testing.T) {
		rb := New()

		// Add values 1-100
		for i := 1; i <= 100; i++ {
			rb.Set(uint32(i))
		}

		// First filter: keep even numbers
		rb.Filter(func(x uint32) bool {
			return x%2 == 0
		})

		// Second filter: keep numbers divisible by 4
		rb.Filter(func(x uint32) bool {
			return x%4 == 0
		})

		// Should have numbers divisible by 4: 4, 8, 12, 16, ..., 100
		assert.Equal(t, 25, rb.Count()) // 100/4 = 25

		rb.Range(func(x uint32) bool {
			assert.Equal(t, uint32(0), x%4, "Value %d should be divisible by 4", x)
			return true
		})
	})
}

func TestContainerTypes(t *testing.T) {
	tests := []struct {
		name          string
		containerType ctype
	}{
		{"array", typeArray},
		{"bitmap", typeBitmap},
		{"run", typeRun},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			our, values := changeType(tt.containerType)

			// Verify container type
			idx, exists := find16(our.index, 0)
			assert.True(t, exists)
			assert.Equal(t, tt.containerType, our.containers[idx].Type)

			// Test all operations work correctly
			assert.Equal(t, len(values), our.Count())
			for _, v := range values {
				assert.True(t, our.Contains(v))
			}

			// Test Range
			var result []uint32
			our.Range(func(x uint32) bool { result = append(result, x); return true })
			assert.Equal(t, values, result)
		})
	}
}

func TestEdgeCases(t *testing.T) {
	t.Run("empty_operations", func(t *testing.T) {
		rb := New()
		assert.Equal(t, 0, rb.Count())
		assert.False(t, rb.Contains(0))
		rb.Remove(123) // Should not panic
		assert.Equal(t, 0, rb.Count())

		var values []uint32
		rb.Range(func(x uint32) bool { values = append(values, x); return true })
		assert.Empty(t, values)
	})

	t.Run("single_value_operations", func(t *testing.T) {
		rb := New()
		rb.Set(42)
		assert.Equal(t, 1, rb.Count())
		assert.True(t, rb.Contains(42))
		assert.False(t, rb.Contains(41))

		rb.Set(42) // Duplicate set
		assert.Equal(t, 1, rb.Count())

		rb.Remove(42)
		assert.Equal(t, 0, rb.Count())
		assert.False(t, rb.Contains(42))
	})

	t.Run("boundary_values", func(t *testing.T) {
		rb := New()
		boundaries := []uint32{0, 65535, 65536, 131071, 131072, 4294967295}

		for _, v := range boundaries {
			rb.Set(v)
			assert.True(t, rb.Contains(v))
		}
		assert.Equal(t, len(boundaries), rb.Count())

		// Test range maintains order
		var result []uint32
		rb.Range(func(x uint32) bool { result = append(result, x); return true })
		assert.Equal(t, boundaries, result)
	})

	t.Run("container_boundaries", func(t *testing.T) {
		rb := New()
		// Test values right at container boundaries
		testValues := []uint32{
			65535, 65536, 65537, // Container 0-1 boundary
			131071, 131072, 131073, // Container 1-2 boundary
			196607, 196608, 196609, // Container 2-3 boundary
		}

		for _, v := range testValues {
			rb.Set(v)
		}

		for _, v := range testValues {
			assert.True(t, rb.Contains(v), "Value %d should be present", v)
		}

		// Remove every other value
		for i, v := range testValues {
			if i%2 == 0 {
				rb.Remove(v)
			}
		}

		for i, v := range testValues {
			if i%2 == 0 {
				assert.False(t, rb.Contains(v), "Value %d should be removed", v)
			} else {
				assert.True(t, rb.Contains(v), "Value %d should still be present", v)
			}
		}
	})
}

func TestRangeStop(t *testing.T) {
	rb := New()
	rb.ctrAdd(0, 0, newBmpPermutations())

	var count int
	for i := 1; i < 64; i++ {
		rb.Range(func(x uint32) bool {
			if x >= uint32(i) {
				count++
				return false
			}

			return true
		})
	}

	assert.Equal(t, 63, count)
}

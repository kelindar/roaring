package roaring

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAndBasic(t *testing.T) {
	t.Run("simple_intersection", func(t *testing.T) {
		rb1 := New()
		rb2 := New()

		// Add overlapping values
		values1 := []uint32{1, 2, 3, 4, 5}
		values2 := []uint32{3, 4, 5, 6, 7}

		for _, v := range values1 {
			rb1.Set(v)
		}
		for _, v := range values2 {
			rb2.Set(v)
		}

		rb1.And(rb2)

		// Should contain intersection: {3, 4, 5}
		assert.Equal(t, 3, rb1.Count())
		assert.True(t, rb1.Contains(3))
		assert.True(t, rb1.Contains(4))
		assert.True(t, rb1.Contains(5))
		assert.False(t, rb1.Contains(1))
		assert.False(t, rb1.Contains(2))
		assert.False(t, rb1.Contains(6))
		assert.False(t, rb1.Contains(7))
	})

	t.Run("no_intersection", func(t *testing.T) {
		rb1 := New()
		rb2 := New()

		// Add non-overlapping values
		rb1.Set(1)
		rb1.Set(2)
		rb2.Set(3)
		rb2.Set(4)

		rb1.And(rb2)

		// Should be empty
		assert.Equal(t, 0, rb1.Count())
		assert.False(t, rb1.Contains(1))
		assert.False(t, rb1.Contains(2))
		assert.False(t, rb1.Contains(3))
		assert.False(t, rb1.Contains(4))
	})

	t.Run("empty_bitmaps", func(t *testing.T) {
		rb1 := New()
		rb2 := New()

		rb1.And(rb2)

		assert.Equal(t, 0, rb1.Count())
	})

	t.Run("and_with_empty", func(t *testing.T) {
		rb1 := New()
		rb2 := New()

		rb1.Set(1)
		rb1.Set(2)
		rb1.Set(3)

		rb1.And(rb2)

		assert.Equal(t, 0, rb1.Count())
	})

	t.Run("and_empty_with_full", func(t *testing.T) {
		rb1 := New()
		rb2 := New()

		rb2.Set(1)
		rb2.Set(2)
		rb2.Set(3)

		rb1.And(rb2)

		assert.Equal(t, 0, rb1.Count())
	})
}

func TestAndNil(t *testing.T) {
	t.Run("and_with_nil", func(t *testing.T) {
		rb := New()
		rb.Set(1)
		rb.Set(2)
		rb.Set(3)

		rb.And(nil)

		assert.Equal(t, 0, rb.Count())
	})

	t.Run("and_with_nil_in_extra", func(t *testing.T) {
		rb1 := New()
		rb2 := New()

		rb1.Set(1)
		rb1.Set(2)
		rb2.Set(1)
		rb2.Set(3)

		rb1.And(rb2, nil)

		// Should work as if nil wasn't there
		assert.Equal(t, 1, rb1.Count())
		assert.True(t, rb1.Contains(1))
	})

	t.Run("and_with_all_nil_extra", func(t *testing.T) {
		rb := New()
		rb.Set(1)
		rb.Set(2)

		rb.And(nil, nil, nil)

		assert.Equal(t, 0, rb.Count())
	})
}

func TestAndMultiple(t *testing.T) {
	t.Run("three_bitmaps", func(t *testing.T) {
		rb1 := New()
		rb2 := New()
		rb3 := New()

		// Add overlapping values
		values1 := []uint32{1, 2, 3, 4, 5, 6}
		values2 := []uint32{2, 3, 4, 5, 6, 7}
		values3 := []uint32{3, 4, 5, 6, 7, 8}

		for _, v := range values1 {
			rb1.Set(v)
		}
		for _, v := range values2 {
			rb2.Set(v)
		}
		for _, v := range values3 {
			rb3.Set(v)
		}

		rb1.And(rb2, rb3)

		// Should contain intersection: {3, 4, 5, 6}
		assert.Equal(t, 4, rb1.Count())
		assert.True(t, rb1.Contains(3))
		assert.True(t, rb1.Contains(4))
		assert.True(t, rb1.Contains(5))
		assert.True(t, rb1.Contains(6))
		assert.False(t, rb1.Contains(1))
		assert.False(t, rb1.Contains(2))
		assert.False(t, rb1.Contains(7))
		assert.False(t, rb1.Contains(8))
	})

	t.Run("multiple_no_intersection", func(t *testing.T) {
		rb1 := New()
		rb2 := New()
		rb3 := New()

		rb1.Set(1)
		rb2.Set(2)
		rb3.Set(3)

		rb1.And(rb2, rb3)

		assert.Equal(t, 0, rb1.Count())
	})
}

func TestAndContainerTypes(t *testing.T) {
	t.Run("array_and_array", func(t *testing.T) {
		rb1 := New()
		rb2 := New()

		// Small arrays
		for i := 0; i < 10; i += 2 {
			rb1.Set(uint32(i))
		}
		for i := 1; i < 11; i += 2 {
			rb2.Set(uint32(i))
		}

		rb1.And(rb2)
		assert.Equal(t, 0, rb1.Count()) // No intersection
	})

	t.Run("array_and_bitmap", func(t *testing.T) {
		rb1 := New()
		rb2 := New()

		// Small array in rb1
		values := []uint32{1, 5, 10, 15, 20}
		for _, v := range values {
			rb1.Set(v)
		}

		// Large bitmap in rb2
		for i := 0; i < 5000; i++ {
			rb2.Set(uint32(i * 2)) // Even numbers
		}

		rb1.And(rb2)

		// Only even values from rb1 should remain
		assert.Equal(t, 2, rb1.Count()) // 10 and 20
		assert.True(t, rb1.Contains(10))
		assert.True(t, rb1.Contains(20))
		assert.False(t, rb1.Contains(1))
		assert.False(t, rb1.Contains(5))
		assert.False(t, rb1.Contains(15))
	})

	t.Run("bitmap_and_run", func(t *testing.T) {
		rb1 := New()
		rb2 := New()

		// Large bitmap
		for i := 0; i < 10000; i += 3 {
			rb1.Set(uint32(i))
		}

		// Consecutive run
		for i := 5000; i <= 6000; i++ {
			rb2.Set(uint32(i))
		}
		rb2.Optimize() // Should become run

		rb1.And(rb2)

		// Count intersections
		count := 0
		for i := 5000; i <= 6000; i++ {
			if i%3 == 0 && rb1.Contains(uint32(i)) {
				count++
			}
		}
		assert.Equal(t, count, rb1.Count())
	})

	t.Run("run_and_run", func(t *testing.T) {
		rb1 := New()
		rb2 := New()

		// First run: 1000-2000
		for i := 1000; i <= 2000; i++ {
			rb1.Set(uint32(i))
		}
		rb1.Optimize()

		// Second run: 1500-2500
		for i := 1500; i <= 2500; i++ {
			rb2.Set(uint32(i))
		}
		rb2.Optimize()

		rb1.And(rb2)

		// Should contain intersection: 1500-2000
		assert.Equal(t, 501, rb1.Count()) // 2000-1500+1
		assert.True(t, rb1.Contains(1500))
		assert.True(t, rb1.Contains(2000))
		assert.False(t, rb1.Contains(1499))
		assert.False(t, rb1.Contains(2001))
	})
}

func TestAndMultipleContainers(t *testing.T) {
	t.Run("multiple_containers", func(t *testing.T) {
		rb1 := New()
		rb2 := New()

		// Container 0
		rb1.Set(10)
		rb2.Set(10)

		// Container 1
		rb1.Set(65536 + 20)
		rb2.Set(65536 + 20)

		// Container 2
		rb1.Set(131072 + 30)
		rb2.Set(131072 + 40) // Different value

		rb1.And(rb2)

		assert.Equal(t, 2, rb1.Count())
		assert.True(t, rb1.Contains(10))
		assert.True(t, rb1.Contains(65536+20))
		assert.False(t, rb1.Contains(131072+30))
		assert.False(t, rb1.Contains(131072+40))
	})

	t.Run("missing_containers", func(t *testing.T) {
		rb1 := New()
		rb2 := New()

		// rb1 has containers 0 and 2
		rb1.Set(10)
		rb1.Set(131072 + 30)

		// rb2 has containers 0 and 1
		rb2.Set(10)
		rb2.Set(65536 + 20)

		rb1.And(rb2)

		// Only container 0 intersection should remain
		assert.Equal(t, 1, rb1.Count())
		assert.True(t, rb1.Contains(10))
		assert.False(t, rb1.Contains(131072+30))
		assert.False(t, rb1.Contains(65536+20))
	})
}

func TestAndLarge(t *testing.T) {
	t.Run("large_intersection", func(t *testing.T) {
		rb1 := New()
		rb2 := New()

		// Large bitmaps with partial overlap
		for i := 0; i < 100000; i++ {
			if i%2 == 0 {
				rb1.Set(uint32(i))
			}
			if i%3 == 0 {
				rb2.Set(uint32(i))
			}
		}

		rb1.And(rb2)

		// Should contain numbers divisible by both 2 and 3 (i.e., by 6)
		expected := 0
		for i := 0; i < 100000; i += 6 {
			expected++
			assert.True(t, rb1.Contains(uint32(i)), "Should contain %d", i)
		}
		assert.Equal(t, expected, rb1.Count())
	})

	t.Run("large_no_intersection", func(t *testing.T) {
		rb1 := New()
		rb2 := New()

		// Non-overlapping ranges
		for i := 0; i < 50000; i++ {
			rb1.Set(uint32(i))
		}
		for i := 50000; i < 100000; i++ {
			rb2.Set(uint32(i))
		}

		rb1.And(rb2)
		assert.Equal(t, 0, rb1.Count())
	})
}

func TestAndOptimization(t *testing.T) {
	t.Run("and_then_optimize", func(t *testing.T) {
		rb1 := New()
		rb2 := New()

		// Create patterns that might change container types
		for i := 0; i < 1000; i++ {
			rb1.Set(uint32(i))
			if i%2 == 0 {
				rb2.Set(uint32(i))
			}
		}

		rb1.And(rb2)
		rb1.Optimize()

		// Should contain even numbers 0-998
		assert.Equal(t, 500, rb1.Count())
		for i := 0; i < 1000; i += 2 {
			assert.True(t, rb1.Contains(uint32(i)))
		}
	})
}

func TestAndEdgeCases(t *testing.T) {
	t.Run("boundary_values", func(t *testing.T) {
		rb1 := New()
		rb2 := New()

		boundaries := []uint32{0, 65535, 65536, 131071, 131072, 4294967295}
		for _, v := range boundaries {
			rb1.Set(v)
			rb2.Set(v)
		}

		rb1.And(rb2)

		assert.Equal(t, len(boundaries), rb1.Count())
		for _, v := range boundaries {
			assert.True(t, rb1.Contains(v))
		}
	})

	t.Run("single_value", func(t *testing.T) {
		rb1 := New()
		rb2 := New()

		rb1.Set(42)
		rb2.Set(42)

		rb1.And(rb2)

		assert.Equal(t, 1, rb1.Count())
		assert.True(t, rb1.Contains(42))
	})

	t.Run("self_and", func(t *testing.T) {
		rb := New()
		values := []uint32{1, 2, 3, 100, 65536, 131072}
		for _, v := range values {
			rb.Set(v)
		}

		originalCount := rb.Count()
		rb.And(rb)

		// Should remain unchanged
		assert.Equal(t, originalCount, rb.Count())
		for _, v := range values {
			assert.True(t, rb.Contains(v))
		}
	})
}

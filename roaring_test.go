package roaring

import (
	"math/rand/v2"
	"testing"

	"github.com/kelindar/bitmap"
	"github.com/stretchr/testify/assert"
)

func TestBasicOperations(t *testing.T) {
	rb := New()

	// Test empty bitmap
	assert.Equal(t, 0, rb.Count())
	assert.False(t, rb.Contains(42))

	// Test basic Set and Contains
	rb.Set(1)
	rb.Set(100)
	rb.Set(65536) // Different container

	assert.True(t, rb.Contains(1))
	assert.True(t, rb.Contains(100))
	assert.True(t, rb.Contains(65536))
	assert.False(t, rb.Contains(2))
	assert.Equal(t, 3, rb.Count())

	// Test Remove
	rb.Remove(100)
	assert.False(t, rb.Contains(100))
	assert.Equal(t, 2, rb.Count())

	// Test Clear
	rb.Clear()
	assert.Equal(t, 0, rb.Count())
	assert.False(t, rb.Contains(1))
}

func TestTransitions(t *testing.T) {
	const i0, i1 = 60000, 100000

	// array -> bitmap
	rb := New()
	for i := 0; i < i0; i++ {
		rb.Set(uint32(i))
		assert.True(t, rb.Contains(uint32(i)))
	}
	assert.Equal(t, i0, rb.Count())

	// bitmap -> run
	rb.Optimize()
	assert.Equal(t, i0, rb.Count())

	// Expand the run
	for i := i0; i < i1; i++ {
		rb.Set(uint32(i))
		assert.True(t, rb.Contains(uint32(i)))
	}
	assert.Equal(t, i1, rb.Count())

	// Remove from a run
	for i := i0; i < i1; i++ {
		rb.Remove(uint32(i))
		assert.False(t, rb.Contains(uint32(i)))
	}
	assert.Equal(t, i0, rb.Count())

	for i := 0; i < i0; i += 2 {
		rb.Remove(uint32(i))
		assert.False(t, rb.Contains(uint32(i)))
	}
	assert.Equal(t, i0/2, rb.Count())
}

// TestMixedOperations covers various operation patterns in single test
func TestMixedOperations(t *testing.T) {
	testCases := [][]uint32{
		{1, 2, 3},                     // Simple case
		{0, 65535, 65536, 131071},     // Container boundaries
		{100, 101, 102, 103, 104},     // Consecutive (run-friendly)
		{1, 100, 1000, 10000, 100000}, // Sparse
	}

	for _, values := range testCases {
		rb := New()

		// Set all values
		for _, v := range values {
			rb.Set(v)
		}

		// Verify count and contains
		assert.Equal(t, len(values), rb.Count())
		for _, v := range values {
			assert.True(t, rb.Contains(v))
		}

		// Test removal pattern
		removed := 0
		for i, v := range values {
			if i%2 == 0 { // Remove every other value
				rb.Remove(v)
				removed++
				assert.False(t, rb.Contains(v))
			}
		}

		assert.Equal(t, len(values)-removed, rb.Count())
	}
}

func TestRandomOperations(t *testing.T) {
	rb := New()
	var ref bitmap.Bitmap

	for i := 0; i < 1e4; i++ {
		value := uint32(rand.IntN(10000))
		switch rand.IntN(10) {
		case 0:
			rb.Remove(value)
			ref.Remove(value)
		default:
			rb.Set(value)
			ref.Set(value)

		}
	}

	assert.Equal(t, ref.Count(), rb.Count())
	ref.Range(func(x uint32) {
		assert.True(t, rb.Contains(x))
	})
}

// TestEdgeCases covers boundary conditions and special values
func TestEdgeCases(t *testing.T) {
	rb := New()

	// Test boundary values
	rb.Set(0)          // Minimum value
	rb.Set(65535)      // Container boundary
	rb.Set(65536)      // Next container
	rb.Set(4294967295) // Maximum uint32

	assert.True(t, rb.Contains(0))
	assert.True(t, rb.Contains(65535))
	assert.True(t, rb.Contains(65536))
	assert.True(t, rb.Contains(4294967295))
	assert.Equal(t, 4, rb.Count())

	// Test duplicate sets (should not increase count)
	rb.Set(0)
	assert.Equal(t, 4, rb.Count())

	// Test removing non-existent value
	rb.Remove(12345)
	assert.Equal(t, 4, rb.Count())
}

// TestRunOperations specifically tests run container behavior
func TestRunOperations(t *testing.T) {
	rb := New()

	// Create consecutive sequence (should form runs efficiently)
	for i := 1000; i <= 1010; i++ {
		rb.Set(uint32(i))
	}

	assert.Equal(t, 11, rb.Count())

	// Verify all values in run
	for i := 1000; i <= 1010; i++ {
		assert.True(t, rb.Contains(uint32(i)))
	}

	// Test run extension
	rb.Set(999)  // Extend backward
	rb.Set(1011) // Extend forward
	assert.Equal(t, 13, rb.Count())

	// Test run splitting by removing middle value
	rb.Remove(1005)
	assert.Equal(t, 12, rb.Count())
	assert.False(t, rb.Contains(1005))
	assert.True(t, rb.Contains(1004))
	assert.True(t, rb.Contains(1006))
}

// TestRange tests the Range function
func TestRange(t *testing.T) {
	rb := New()
	
	// Test empty bitmap
	var values []uint32
	rb.Range(func(x uint32) {
		values = append(values, x)
	})
	assert.Empty(t, values)
	
	// Test single value
	rb.Set(42)
	values = nil
	rb.Range(func(x uint32) {
		values = append(values, x)
	})
	assert.Equal(t, []uint32{42}, values)
	
	// Test multiple values in same container
	rb.Set(10)
	rb.Set(100)
	values = nil
	rb.Range(func(x uint32) {
		values = append(values, x)
	})
	// Values should be in sorted order
	expected := []uint32{10, 42, 100}
	assert.Equal(t, expected, values)
	
	// Test values across multiple containers
	rb.Set(65536)  // Different container
	rb.Set(131072) // Another container
	values = nil
	rb.Range(func(x uint32) {
		values = append(values, x)
	})
	expected = []uint32{10, 42, 100, 65536, 131072}
	assert.Equal(t, expected, values)
}

// TestRangeWithDifferentContainerTypes tests Range with array, bitmap, and run containers
func TestRangeWithDifferentContainerTypes(t *testing.T) {
	rb := New()
	
	// Create array container (few values)
	rb.Set(1)
	rb.Set(3)
	rb.Set(5)
	
	// Create run container (consecutive values)
	for i := 1000; i <= 1010; i++ {
		rb.Set(uint32(i))
	}
	
	// Create bitmap container (many values)
	for i := 0; i < 5000; i++ {
		rb.Set(uint32(i * 10))
	}
	
	// Collect all values
	var values []uint32
	rb.Range(func(x uint32) {
		values = append(values, x)
	})
	
	// Verify all values are present and in order
	assert.True(t, len(values) > 5000)
	
	// Check that values are in ascending order
	for i := 1; i < len(values); i++ {
		assert.True(t, values[i] > values[i-1], "Values should be in ascending order")
	}
	
	// Verify specific values are present
	assert.Contains(t, values, uint32(1))
	assert.Contains(t, values, uint32(3))
	assert.Contains(t, values, uint32(5))
	assert.Contains(t, values, uint32(1005))
}

// TestRangeCompareWithReference compares Range output with reference bitmap
func TestRangeCompareWithReference(t *testing.T) {
	rb := New()
	var ref bitmap.Bitmap
	
	// Add same values to both bitmaps
	testValues := []uint32{1, 10, 100, 1000, 10000, 65536, 100000}
	for _, v := range testValues {
		rb.Set(v)
		ref.Set(v)
	}
	
	// Collect values from both
	var ourValues []uint32
	rb.Range(func(x uint32) {
		ourValues = append(ourValues, x)
	})
	
	var refValues []uint32
	ref.Range(func(x uint32) {
		refValues = append(refValues, x)
	})
	
	// Should be identical
	assert.Equal(t, refValues, ourValues)
}

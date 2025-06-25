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

func TestRange(t *testing.T) {
	rb := New()
	var ref bitmap.Bitmap

	// Test empty bitmap
	var values []uint32
	rb.Range(func(x uint32) {
		values = append(values, x)
	})
	assert.Empty(t, values)

	// Test various patterns against reference implementation
	testCases := [][]uint32{
		{42},                                   // Single value
		{1, 10, 100},                           // Multiple values same container
		{1, 65536, 131072},                     // Multiple containers
		{0, 65535, 65536, 4294967295},          // Container boundaries
		{100, 101, 102, 103, 104},              // Consecutive values
		{1, 100, 1000, 10000, 100000, 1000000}, // Sparse values
	}

	for _, testValues := range testCases {
		rb.Clear()
		ref.Clear()

		// Set same values in both bitmaps
		for _, v := range testValues {
			rb.Set(v)
			ref.Set(v)
		}

		// Compare Range output
		var ourValues, refValues []uint32
		rb.Range(func(x uint32) { ourValues = append(ourValues, x) })
		ref.Range(func(x uint32) { refValues = append(refValues, x) })

		assert.Equal(t, refValues, ourValues, "Range mismatch for values: %v", testValues)
	}
}

func TestRange_ArrayContainer(t *testing.T) {
	our := New()

	// Force array container by adding few sparse values
	values := []uint32{1, 5, 10, 100, 500, 1000}
	for _, v := range values {
		our.Set(v)
	}

	// Verify it's an array container
	c, exists := our.findContainer(0)
	assert.True(t, exists)
	assert.Equal(t, typeArray, c.Type)

	// Test Range on array container
	var result []uint32
	our.Range(func(x uint32) {
		result = append(result, x)
	})

	assert.Equal(t, values, result)
}

func TestRange_BitmapContainer(t *testing.T) {
	rb := New()

	// Force bitmap container by adding many sparse values to prevent run optimization
	var expected []uint32
	for i := 0; i < 5000; i++ {
		v := uint32(i * 3) // Sparse values to prevent run container optimization
		rb.Set(v)
		expected = append(expected, v)
	}

	// Verify it's a bitmap container
	c, exists := rb.findContainer(0)
	assert.True(t, exists)
	assert.Equal(t, typeBitmap, c.Type)

	// Test Range on bitmap container
	var result []uint32
	rb.Range(func(x uint32) {
		result = append(result, x)
	})

	assert.Equal(t, expected, result)
}

func TestRange_RunContainer(t *testing.T) {
	our := New()

	// Create consecutive values and optimize to force run container
	for i := 1000; i <= 2000; i++ {
		our.Set(uint32(i))
	}
	our.Optimize() // Force optimization to run container

	// Verify it's a run container
	c, exists := our.findContainer(0)
	assert.True(t, exists)
	assert.Equal(t, typeRun, c.Type)

	// Test Range on run container
	var result []uint32
	our.Range(func(x uint32) {
		result = append(result, x)
	})

	// Verify all consecutive values are present
	assert.Equal(t, 1001, len(result))
	for i, v := range result {
		assert.Equal(t, uint32(1000+i), v)
	}
}

func TestRange_MultipleContainers(t *testing.T) {
	our := New()
	var ref bitmap.Bitmap

	// Create different container types in different containers
	// Container 0: array (few values)
	arrayValues := []uint32{1, 5, 10}

	// Container 1: bitmap (many values)
	var bitmapValues []uint32
	for i := 0; i < 3000; i++ {
		v := uint32(65536 + i)
		bitmapValues = append(bitmapValues, v)
	}

	// Container 2: run (consecutive values)
	var runValues []uint32
	for i := 131072; i <= 131572; i++ {
		runValues = append(runValues, uint32(i))
	}

	// Add all values to both bitmaps
	allValues := append(append(arrayValues, bitmapValues...), runValues...)
	for _, v := range allValues {
		our.Set(v)
		ref.Set(v)
	}

	// Optimize to ensure proper container types
	our.Optimize()

	// Compare Range output
	var ourValues, refValues []uint32
	our.Range(func(x uint32) { ourValues = append(ourValues, x) })
	ref.Range(func(x uint32) { refValues = append(refValues, x) })

	assert.Equal(t, refValues, ourValues)
	assert.Equal(t, len(allValues), len(ourValues))
}

func TestRange_RandomData(t *testing.T) {
	our := New()
	var ref bitmap.Bitmap

	// Generate random data and compare with reference
	for i := 0; i < 1e4; i++ {
		value := uint32(rand.IntN(100000))
		our.Set(value)
		ref.Set(value)
	}

	var ourValues, refValues []uint32
	our.Range(func(x uint32) { ourValues = append(ourValues, x) })
	ref.Range(func(x uint32) { refValues = append(refValues, x) })

	assert.Equal(t, refValues, ourValues)
}

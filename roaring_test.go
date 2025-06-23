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
	const count = 60000

	t.Run("array -> bitmap -> array", func(t *testing.T) {
		rb := New()
		for i := 0; i < count; i++ {
			rb.Set(uint32(i))
			assert.True(t, rb.Contains(uint32(i)))
		}
		assert.Equal(t, count, rb.Count())
		for i := 0; i < count; i++ {
			rb.Remove(uint32(i))
			assert.False(t, rb.Contains(uint32(i)))
		}
		assert.Equal(t, 0, rb.Count())
	})

	t.Run("bitmap -> run -> bitmap", func(t *testing.T) {
		rb := New()
		for i := 0; i < count; i++ {
			rb.Set(uint32(i))
			assert.True(t, rb.Contains(uint32(i)))
		}

		rb.Optimize()
		assert.Equal(t, count, rb.Count())

		for i := 0; i < count; i++ {
			rb.Remove(uint32(i))
			assert.False(t, rb.Contains(uint32(i)))
		}
		assert.Equal(t, 0, rb.Count())
	})

	t.Run("array -> run", func(t *testing.T) {
		rb := New()
		for i := 0; i < 500; i++ {
			rb.Set(uint32(i))
			assert.True(t, rb.Contains(uint32(i)))
		}
		rb.Optimize()
		assert.Equal(t, 500, rb.Count())
	})

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
		switch rand.IntN(3) {
		case 0:
			rb.Set(value)
			ref.Set(value)
		case 1:
			rb.Remove(value)
			ref.Remove(value)
		case 3:
			rb.Optimize()
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

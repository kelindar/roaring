package roaring

import (
	"math/rand/v2"
	"testing"
)

// Test helper functions for clean assertions
func assertEqual(t *testing.T, expected, actual interface{}) {
	t.Helper()
	if expected != actual {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

func assertTrue(t *testing.T, value bool) {
	t.Helper()
	if !value {
		t.Error("Expected true, got false")
	}
}

func assertFalse(t *testing.T, value bool) {
	t.Helper()
	if value {
		t.Error("Expected false, got true")
	}
}

// TestBasicOperations covers 80% of use cases: Set, Contains, Remove, Count, Clear
func TestBasicOperations(t *testing.T) {
	rb := New()

	// Test empty bitmap
	assertEqual(t, 0, rb.Count())
	assertFalse(t, rb.Contains(42))

	// Test basic Set and Contains
	rb.Set(1)
	rb.Set(100)
	rb.Set(65536) // Different container

	assertTrue(t, rb.Contains(1))
	assertTrue(t, rb.Contains(100))
	assertTrue(t, rb.Contains(65536))
	assertFalse(t, rb.Contains(2))
	assertEqual(t, 3, rb.Count())

	// Test Remove
	rb.Remove(100)
	assertFalse(t, rb.Contains(100))
	assertEqual(t, 2, rb.Count())

	// Test Clear
	rb.Clear()
	assertEqual(t, 0, rb.Count())
	assertFalse(t, rb.Contains(1))
}

// TestContainerTransitions verifies container type changes work correctly
func TestContainerTransitions(t *testing.T) {
	const count = 60000
	rb := New()

	// Force transitions array -> bitmap -> run
	for i := 0; i < count; i++ {
		rb.Set(uint32(i))
	}
	assertEqual(t, count, rb.Count())

	// Force transitions run -> bitmap -> array
	for i := 0; i < count; i++ {
		rb.Remove(uint32(i))
	}
	assertEqual(t, 0, rb.Count())

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
		assertEqual(t, len(values), rb.Count())
		for _, v := range values {
			assertTrue(t, rb.Contains(v))
		}

		// Test removal pattern
		removed := 0
		for i, v := range values {
			if i%2 == 0 { // Remove every other value
				rb.Remove(v)
				removed++
				assertFalse(t, rb.Contains(v))
			}
		}

		assertEqual(t, len(values)-removed, rb.Count())
	}
}

// TestRandomOperations uses random data to catch edge cases
func TestRandomOperations(t *testing.T) {
	const numOps = 1000
	rb := New()
	setValues := make(map[uint32]bool)

	for i := 0; i < numOps; i++ {
		value := uint32(rand.IntN(10000)) // Smaller range for more collisions

		if rand.IntN(2) == 0 {
			// Set operation
			rb.Set(value)
			setValues[value] = true
		} else {
			// Remove operation
			rb.Remove(value)
			delete(setValues, value)
		}

		// Verify consistency every 100 operations
		if i%100 == 0 {
			assertEqual(t, len(setValues), rb.Count())
		}
	}

	// Final verification - count should match our tracking
	assertEqual(t, len(setValues), rb.Count())

	// Spot check some values
	for value := range setValues {
		assertTrue(t, rb.Contains(value))
	}
}

// TestEdgeCases covers boundary conditions and special values
func TestEdgeCases(t *testing.T) {
	rb := New()

	// Test boundary values
	rb.Set(0)          // Minimum value
	rb.Set(65535)      // Container boundary
	rb.Set(65536)      // Next container
	rb.Set(4294967295) // Maximum uint32

	assertTrue(t, rb.Contains(0))
	assertTrue(t, rb.Contains(65535))
	assertTrue(t, rb.Contains(65536))
	assertTrue(t, rb.Contains(4294967295))
	assertEqual(t, 4, rb.Count())

	// Test duplicate sets (should not increase count)
	rb.Set(0)
	assertEqual(t, 4, rb.Count())

	// Test removing non-existent value
	rb.Remove(12345)
	assertEqual(t, 4, rb.Count())
}

// TestRunOperations specifically tests run container behavior
func TestRunOperations(t *testing.T) {
	rb := New()

	// Create consecutive sequence (should form runs efficiently)
	for i := 1000; i <= 1010; i++ {
		rb.Set(uint32(i))
	}

	assertEqual(t, 11, rb.Count())

	// Verify all values in run
	for i := 1000; i <= 1010; i++ {
		assertTrue(t, rb.Contains(uint32(i)))
	}

	// Test run extension
	rb.Set(999)  // Extend backward
	rb.Set(1011) // Extend forward
	assertEqual(t, 13, rb.Count())

	// Test run splitting by removing middle value
	rb.Remove(1005)
	assertEqual(t, 12, rb.Count())
	assertFalse(t, rb.Contains(1005))
	assertTrue(t, rb.Contains(1004))
	assertTrue(t, rb.Contains(1006))
}

package roaring

import (
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
			our.Range(func(x uint32) { ourValues = append(ourValues, x) })
			ref.Range(func(x uint32) { refValues = append(refValues, x) })

			assert.Equal(t, refValues, ourValues)
		})
	}
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
			c, exists := our.findContainer(0)
			assert.True(t, exists)
			assert.Equal(t, tt.containerType, c.Type)

			// Test all operations work correctly
			assert.Equal(t, len(values), our.Count())
			for _, v := range values {
				assert.True(t, our.Contains(v))
			}

			// Test Range
			var result []uint32
			our.Range(func(x uint32) { result = append(result, x) })
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
		rb.Range(func(x uint32) { values = append(values, x) })
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
		rb.Range(func(x uint32) { result = append(result, x) })
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

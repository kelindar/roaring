package roaring

import "sort"

// arraySet sets a value in an array container
func (c *container) arraySet(value uint16) bool {
	array := c.array()

	// Check if value already exists and find insertion point
	for i, v := range array {
		if v == value {
			return false // Already exists
		}
		if v > value {
			// Insert at position i
			c.Data = append(c.Data, 0, 0) // Add space for new uint16
			newArray := c.array()
			copy(newArray[i+1:], newArray[i:])
			newArray[i] = value
			c.Size++ // Increment cardinality
			return true
		}
	}

	// Append at end
	c.Data = append(c.Data, 0, 0) // Add space for new uint16
	newArray := c.array()
	newArray[len(newArray)-1] = value
	c.Size++ // Increment cardinality
	return true
}

// arrayRemove removes a value from an array container
func (c *container) arrayRemove(value uint16) bool {
	array := c.array()
	for i, v := range array {
		if v == value {
			// Remove element at index i
			copy(array[i:], array[i+1:])
			c.Data = c.Data[:len(c.Data)-2] // Shrink by one uint16
			c.Size--                        // Decrement cardinality
			return true
		}
	}
	return false
}

// arrayContains checks if a value exists in an array container
func (c *container) arrayContains(value uint16) bool {
	array := c.array()
	// Binary search for efficiency
	i := sort.Search(len(array), func(i int) bool {
		return array[i] >= value
	})
	return i < len(array) && array[i] == value
}

// arrayShouldConvertToBitmap returns true if array should be converted to bitmap
func (c *container) arrayShouldConvertToBitmap() bool {
	return c.Size > 4096
}

// arrayShouldConvertToRun returns true if array should be converted to run
func (c *container) arrayShouldConvertToRun() bool {
	// For now, we don't auto-convert array to run
	// This would require analyzing the array for consecutive sequences
	// which is not critical for basic functionality
	return false
}

// arrayConvertFromBitmap converts this container from bitmap to array
func (c *container) arrayConvertFromBitmap() {
	bm := c.bitmap()
	var values []uint16

	// Collect all set bits
	for i := uint32(0); i < 65536; i++ {
		if bm.Contains(i) {
			values = append(values, uint16(i))
		}
	}

	// Create new array data
	c.Data = make([]byte, len(values)*2)
	c.Type = typeArray
	c.Size = uint16(len(values)) // Set cardinality
	array := c.array()
	copy(array, values)
}

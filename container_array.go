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
	array := c.array()
	if len(array) < 128 {
		return false // Need at least 128 elements to form a meaningful run
	}

	// Count potential runs by analyzing consecutive sequences
	numRuns := 1
	for i := 1; i < len(array); i++ {
		if array[i] != array[i-1]+1 {
			numRuns++
		}
	}

	// Convert to run if it would save significant space and we have few runs
	// Array: 2 bytes per element
	// Run: 4 bytes per run + 2 bytes header
	sizeAsArray := len(array) * 2
	sizeAsRun := numRuns*4 + 2

	// Only convert if we save at least 25% space and have reasonable compression
	return sizeAsRun < sizeAsArray*3/4 && numRuns <= len(array)/3
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

// arrayConvertFromRun converts this container from run to array
func (c *container) arrayConvertFromRun() {
	runs := c.run()
	var values []uint16

	// Extract all values from runs
	for _, r := range runs {
		for value := r[0]; value <= r[1]; value++ {
			values = append(values, value)
			if value == r[1] {
				break // Prevent uint16 overflow when r[1] is 65535
			}
		}
	}

	// Create new array data
	c.Data = make([]byte, len(values)*2)
	c.Type = typeArray
	c.Size = uint16(len(values)) // Set cardinality
	array := c.array()
	copy(array, values)
}

package roaring

import (
	"sort"
	"unsafe"
)

// arr converts the container to an []uint16
func (c *container) arr() []uint16 {
	if len(c.Data) == 0 {
		return nil
	}

	return unsafe.Slice((*uint16)(unsafe.Pointer(&c.Data[0])), len(c.Data)/2)
}

// arrSet sets a value in an array container
func (c *container) arrSet(value uint16) bool {
	array := c.arr()

	// Check if value already exists and find insertion point
	for i, v := range array {
		if v == value {
			return false // Already exists
		}
		if v > value {
			// Insert at position i
			c.Data = append(c.Data, 0, 0) // Add space for new uint16
			newArray := c.arr()
			copy(newArray[i+1:], newArray[i:])
			newArray[i] = value
			c.Size++ // Increment cardinality
			return true
		}
	}

	// Append at end
	c.Data = append(c.Data, 0, 0) // Add space for new uint16
	newArray := c.arr()
	newArray[len(newArray)-1] = value
	c.Size++ // Increment cardinality
	return true
}

// arrDel removes a value from an array container
func (c *container) arrDel(value uint16) bool {
	array := c.arr()
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

// arrHas checks if a value exists in an array container
func (c *container) arrHas(value uint16) bool {
	array := c.arr()
	i := sort.Search(len(array), func(i int) bool {
		return array[i] >= value
	})
	return i < len(array) && array[i] == value
}

// arrShouldConvertToBitmap returns true if array should be converted to bitmap
func (c *container) arrShouldConvertToBitmap() bool {
	return c.Size > arrMinSize
}

// arrTryConvertToRun attempts to convert array to run in a single pass
// Returns true if conversion was performed, false otherwise
func (c *container) arrTryConvertToRun() bool {
	array := c.arr()
	if len(array) < 128 {
		return false // Need at least 128 elements to form a meaningful run
	}

	var runs []run

	// Single iteration: build runs AND count them
	currentStart := array[0]
	currentEnd := array[0]

	for i := 1; i < len(array); i++ {
		if array[i] == currentEnd+1 {
			// Continue current run
			currentEnd = array[i]
		} else {
			// End current run and start new one
			runs = append(runs, run{currentStart, currentEnd})
			currentStart = array[i]
			currentEnd = array[i]
		}
	}

	// Add the final run
	runs = append(runs, run{currentStart, currentEnd})

	// Now check conversion criteria with the actual run count
	numRuns := len(runs)

	// Convert to run if it would save significant space and we have few runs
	// Array: 2 bytes per element
	// Run: 4 bytes per run + 2 bytes header
	sizeAsArray := len(array) * 2
	sizeAsRun := numRuns*4 + 2

	// Only convert if we save at least 25% space and have reasonable compression
	shouldConvert := sizeAsRun < sizeAsArray*3/4 && numRuns <= len(array)/3
	if shouldConvert {
		c.Data = make([]byte, len(runs)*4) // 4 bytes per run (2 uint16s)
		c.Type = typeRun
		newRuns := c.run()
		copy(newRuns, runs)
		return true
	}

	return false
}

// arrToBmp converts this container from array to bitmap
func (c *container) arrToBmp() {
	array := c.arr()

	// Create bitmap data (65536 bits = 8192 bytes)
	c.Data = make([]byte, 8192)
	c.Type = typeBitmap
	bm := c.bmp()

	// Set all bits from the array
	for _, value := range array {
		bm.Set(uint32(value))
	}

	// Update cardinality from bitmap
	c.Size = uint16(bm.Count())
}

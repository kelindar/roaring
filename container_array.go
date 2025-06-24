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

	// Use binary search to find insertion point
	idx := sort.Search(len(array), func(i int) bool {
		return array[i] >= value
	})

	// Check if value already exists
	if idx < len(array) && array[idx] == value {
		return false // Already exists
	}

	// Insert at position idx
	c.Data = append(c.Data, 0, 0) // Add space for new uint16
	newArray := c.arr()
	copy(newArray[idx+1:], newArray[idx:len(newArray)-1])
	newArray[idx] = value
	c.Size++
	return true
}

// arrDel removes a value from an array container
func (c *container) arrDel(value uint16) bool {
	array := c.arr()

	// Use binary search to find the value
	idx := sort.Search(len(array), func(i int) bool {
		return array[i] >= value
	})

	// Check if value exists
	if idx >= len(array) || array[idx] != value {
		return false
	}

	// Remove element at index idx
	copy(array[idx:], array[idx+1:])
	c.Data = c.Data[:len(c.Data)-2] // Shrink by one uint16
	c.Size--                        // Decrement cardinality
	return true
}

// arrHas checks if a value exists in an array container
func (c *container) arrHas(value uint16) bool {
	array := c.arr()
	i := sort.Search(len(array), func(i int) bool {
		return array[i] >= value
	})
	return i < len(array) && array[i] == value
}

// arrOptimize tries to optimize the container
func (c *container) arrOptimize() {
	switch {
	case c.arrTryConvertToRun():
	case c.Size > arrMinSize:
		c.arrToBmp()
	}
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
	src := c.arr()

	// Create bitmap data (65536 bits = 8192 bytes)
	c.Data = make([]byte, 8192)
	c.Type = typeBitmap
	dst := c.bmp()

	// Copy all values to the bitmap
	for _, value := range src {
		dst.Set(uint32(value))
	}
}

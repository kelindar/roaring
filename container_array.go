package roaring

import (
	"unsafe"
)

// arr converts the container to an []uint16
func (c *container) arr() []uint16 {
	if len(c.Data) == 0 {
		return nil
	}

	return unsafe.Slice((*uint16)(unsafe.Pointer(&c.Data[0])), len(c.Data)/2)
}

// arrFind performs optimized binary search in array container
// Returns (index, found) where index is the insertion point if not found
func (c *container) arrFind(value uint16) (int, bool) {
	array := c.arr()
	n := len(array)

	// Quick bounds check for early exit
	if n == 0 {
		return 0, false
	}
	if value < array[0] {
		return 0, false
	}
	if value > array[n-1] {
		return n, false
	}

	// Optimized binary search with fewer comparisons
	left, right := 0, n
	for left < right {
		mid := left + (right-left)/2 // avoid overflow
		if array[mid] < value {
			left = mid + 1
		} else {
			right = mid
		}
	}

	return left, left < n && array[left] == value
}

// arrSet sets a value in an array container
func (c *container) arrSet(value uint16) bool {
	idx, exists := c.arrFind(value)
	if exists {
		return false // Already exists
	}

	// Insert at position idx more efficiently
	array := c.arr()
	oldLen := len(array)
	c.Data = append(c.Data, 0, 0) // Add space for new uint16
	newArray := c.arr()

	// Move elements to the right using bulk copy
	if idx < oldLen {
		copy(newArray[idx+1:], array[idx:])
	}

	newArray[idx] = value
	c.Size++
	return true
}

// arrDel removes a value from an array container
func (c *container) arrDel(value uint16) bool {
	idx, exists := c.arrFind(value)
	if !exists {
		return false
	}

	// Remove element at index idx
	array := c.arr()
	copy(array[idx:], array[idx+1:])
	c.Data = c.Data[:len(c.Data)-2] // Shrink by one uint16
	c.Size--
	return true
}

// arrHas checks if a value exists in an array container
func (c *container) arrHas(value uint16) bool {
	_, exists := c.arrFind(value)
	return exists
}

// arrOptimize tries to optimize the container
func (c *container) arrOptimize() {
	switch {
	case c.arrIsDense():
		c.arrToRun()
	case c.Size > arrMinSize:
		c.arrToBmp()
	}
}

// arrIsDense quickly estimates if converting to run container would be beneficial
func (c *container) arrIsDense() bool {
	array := c.arr()
	if len(array) < 128 {
		return false
	}

	lo, hi := array[0], array[len(array)-1]
	span := int(hi - lo + 1)
	size := len(array)

	// Quick density filters
	density := float64(size) / float64(span)
	switch {
	case density < 0.1: // Very sparse
		return false
	case density > 0.8: // Very dense
		return true
	}

	// Estimate number of runs using density
	runs := size
	if gap := float64(span) / float64(size); gap < 2.0 {
		runs = int(float64(size) * (1.0 - density*0.7))
	}

	// Check if estimated conversion meets our criteria
	sizeAsArr := size * 2
	sizeAsRun := runs*4 + 2
	return sizeAsRun < sizeAsArr*3/4 && runs <= size/3
}

// arrToRun attempts to convert array to run in a single pass
func (c *container) arrToRun() bool {
	array := c.arr()
	runs := make([]run, 0, len(array)/4) // estimate runs needed

	// Single iteration: build runs AND count them
	i0 := array[0]
	i1 := array[0]

	for i := 1; i < len(array); i++ {
		if array[i] == i1+1 {
			// Continue current run
			i1 = array[i]
		} else {
			// End current run and start new one
			runs = append(runs, run{i0, i1})
			i0 = array[i]
			i1 = array[i]
		}
	}

	// Add the final run
	runs = append(runs, run{i0, i1})

	// Check conversion criteria with the actual run count
	numRuns := len(runs)
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

	// Use bulk setting for better performance
	for _, value := range src {
		dst.Set(uint32(value))
	}
}

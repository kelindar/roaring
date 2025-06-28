// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

package roaring

func (c *container) runFind(value uint16) (idx [2]int, ok bool) {
	n := len(c.Data) >> 1
	switch {
	case n == 0:
		return [2]int{0, 0}, false
	case value < c.Data[0]:
		return [2]int{0, 0}, false
	case value > c.Data[(n-1)*2+1]:
		return [2]int{n, n}, false
	}

	// binary phase: shrink window to ≤4 runs
	lo, hi := 0, n
	for hi-lo > 4 {
		mid := (lo + hi) >> 1
		start := c.Data[mid*2]
		if value < start {
			hi = mid
			continue
		}
		end := c.Data[mid*2+1]
		if value <= end { // hit
			return [2]int{mid, mid}, true
		}
		lo = mid + 1
	}

	// linear phase inside one cache line
	for i := lo; i < hi; i++ {
		switch {
		case value < c.Data[i*2]:
			return [2]int{i, i}, false
		case value <= c.Data[i*2+1]:
			return [2]int{i, i}, true
		}
	}

	// value is greater than end of hi-1 but ≤ lastEnd (already checked)
	return [2]int{hi, hi}, false
}

// runSet sets a value in a run container
func (c *container) runSet(value uint16) bool {
	search, found := c.runFind(value)
	if found {
		return false // Value already exists
	}

	idx := search[1]
	numRuns := len(c.Data) / 2

	// Check boundary cases for merging/extending
	canMergeLeft := idx > 0 && numRuns > 0 && c.Data[(idx-1)*2+1]+1 == value
	canMergeRight := idx < numRuns && numRuns > 0 && c.Data[idx*2]-1 == value

	switch {
	case canMergeLeft && canMergeRight:
		c.Data[(idx-1)*2+1] = c.Data[idx*2+1]
		c.runRemoveRunAt(idx)
	case canMergeLeft:
		c.Data[(idx-1)*2+1] = value
	case canMergeRight:
		c.Data[idx*2] = value
	default:
		c.runInsertRunAt(idx, value, value)
	}

	c.Size++
	return true
}

// runDel removes a value from a run container
func (c *container) runDel(value uint16) bool {
	search, found := c.runFind(value)
	if !found {
		return false
	}

	idx := search[0]
	start := c.Data[idx*2]
	end := c.Data[idx*2+1]

	switch {
	case start == end:
		c.runRemoveRunAt(idx)
	case value == start:
		c.Data[idx*2] = value + 1
	case value == end:
		c.Data[idx*2+1] = value - 1
	default:
		c.Data[idx*2+1] = value - 1
		c.runInsertRunAt(idx+1, value+1, end)
	}

	c.Size--
	return true
}

// runHas checks if a value exists in a run container
func (c *container) runHas(value uint16) bool {
	_, found := c.runFind(value)
	return found
}

// runInsertRunAt inserts a new run at the specified index
func (c *container) runInsertRunAt(index int, start, end uint16) {
	numRuns := len(c.Data) / 2
	newLen := (numRuns + 1) * 2

	// Try to avoid allocation if we have enough capacity
	if cap(c.Data) >= newLen {
		c.Data = c.Data[:newLen]
		if index < numRuns {
			copy(c.Data[(index+1)*2:], c.Data[index*2:numRuns*2])
		}
	} else {
		// Need to allocate new slice with extra capacity for future insertions
		extraCapacity := max(16, numRuns) // Add 50% extra capacity or minimum 8 runs
		newData := make([]uint16, newLen, newLen+extraCapacity)

		// Copy existing runs with efficient bulk operations
		copy(newData, c.Data[:index*2])
		if index < numRuns {
			copy(newData[(index+1)*2:], c.Data[index*2:])
		}
		c.Data = newData
	}

	c.Data[index*2] = start
	c.Data[index*2+1] = end
}

// runRemoveRunAt removes the run at the specified index
func (c *container) runRemoveRunAt(index int) {
	numRuns := len(c.Data) / 2
	if index < 0 || index >= numRuns {
		return
	}

	if numRuns == 1 {
		c.Data = c.Data[:0] // Keep capacity but set length to 0
		return
	}

	// Move runs in-place to avoid allocation
	copy(c.Data[index*2:], c.Data[(index+1)*2:])
	c.Data = c.Data[:(numRuns-1)*2] // Shrink slice length
}

// runOptimize tries to optimize the container
func (c *container) runOptimize() {
	if c.Type != typeRun || c.Size == 0 {
		return
	}

	numRuns := len(c.Data) / 2
	avgRunLength := float64(c.Size) / float64(numRuns)
	compressionVsBitmap := float64(numRuns*4+2) / float64(8192)
	runDensity := float64(numRuns) / float64(c.Size)

	switch {
	case numRuns > runMaxSize:
		c.runToBmp()
	case c.Size <= 4096 && runDensity > 0.5:
		c.runToArray()
	case c.Size > 32768 && compressionVsBitmap > 0.8:
		c.runToBmp()
	case avgRunLength < 2.0:
		c.runToArray()
	}
}

// runToArray converts this container from run to array
func (c *container) runToArray() {
	numRuns := len(c.Data) / 2
	srcData := c.Data

	// Create new array data
	c.Data = make([]uint16, c.Size)
	c.Type = typeArray
	dst := c.Data

	// Copy all values to the array
	idx := 0
	for i := 0; i < numRuns; i++ {
		start, end := srcData[i*2], srcData[i*2+1]
		for value := start; value <= end; value++ {
			dst[idx] = value
			idx++
			if value == end {
				break // Prevent uint16 overflow when end is 65535
			}
		}
	}
}

// runToBmp converts this container from run to bitmap
func (c *container) runToBmp() {
	numRuns := len(c.Data) / 2
	srcData := c.Data

	// Create bitmap data (65536 bits = 8192 bytes = 4096 uint16s)
	c.Data = make([]uint16, 4096)
	c.Type = typeBitmap
	dst := c.bmp()

	for i := 0; i < numRuns; i++ {
		start, end := srcData[i*2], srcData[i*2+1]
		for v := start; v <= end; v++ {
			dst.Set(uint32(v))
			if v == end {
				break // Prevent uint16 overflow when end is 65535
			}
		}
	}
}

// runMin returns the smallest value in a run container
func (c *container) runMin() (uint16, bool) {
	if len(c.Data) == 0 {
		return 0, false
	}
	return c.Data[0], true // First run's start
}

// runMax returns the largest value in a run container
func (c *container) runMax() (uint16, bool) {
	if len(c.Data) == 0 {
		return 0, false
	}
	return c.Data[len(c.Data)-1], true // Last run's end
}

// runMinZero returns the smallest unset value in a run container
func (c *container) runMinZero() (uint16, bool) {
	if len(c.Data) == 0 {
		return 0, true // Empty container, 0 is unset
	}

	numRuns := len(c.Data) / 2

	// Check if 0 is unset (before first run)
	if c.Data[0] > 0 {
		return 0, true
	}

	// Find first gap between runs
	for i := 0; i < numRuns-1; i++ {
		currentEnd := c.Data[i*2+1]
		nextStart := c.Data[(i+1)*2]

		if nextStart > currentEnd+1 {
			return currentEnd + 1, true
		}
	}

	// Check if there's a gap after the last run
	lastEnd := c.Data[(numRuns-1)*2+1]
	if lastEnd < 65535 {
		return lastEnd + 1, true
	}

	return 0, false // No gaps found, all values 0-65535 are covered
}

// runMaxZero returns the largest unset value in a run container
func (c *container) runMaxZero() (uint16, bool) {
	if len(c.Data) == 0 {
		return 65535, true // Empty container, 65535 is unset
	}

	numRuns := len(c.Data) / 2

	// Check if 65535 is unset (after last run)
	lastEnd := c.Data[(numRuns-1)*2+1]
	if lastEnd < 65535 {
		return 65535, true
	}

	// Find last gap between runs (search backwards)
	for i := numRuns - 1; i > 0; i-- {
		currentStart := c.Data[i*2]
		prevEnd := c.Data[(i-1)*2+1]

		if currentStart > prevEnd+1 {
			return currentStart - 1, true
		}
	}

	// Check if there's a gap before the first run
	if c.Data[0] > 0 {
		return c.Data[0] - 1, true
	}

	return 0, false // No gaps found, all values 0-65535 are covered
}

package roaring

import "unsafe"

// run converts the container to a []run
func (c *container) run() []run {
	if len(c.Data) == 0 {
		return nil
	}

	return unsafe.Slice((*run)(unsafe.Pointer(&c.Data[0])), len(c.Data)/2)
}

// runFind performs binary search to find a value in the run container
func (c *container) runFind(value uint16) ([2]int, bool) {
	runs := c.run()
	if len(runs) == 0 {
		return [2]int{0, 0}, false
	}

	// Fast path for small containers - linear search is faster than binary search
	if len(runs) <= 4 {
		for i, run := range runs {
			if value >= run[0] && value <= run[1] {
				return [2]int{i, i}, true
			}
			if value < run[0] {
				return [2]int{i, i}, false
			}
		}
		return [2]int{len(runs), len(runs)}, false
	}

	// Binary search for larger containers
	left, right := 0, len(runs)-1
	for left <= right {
		mid := (left + right) / 2
		run := runs[mid]
		switch {
		case value >= run[0] && value <= run[1]:
			return [2]int{mid, mid}, true
		case value < run[0]:
			right = mid - 1
		default:
			left = mid + 1
		}
	}

	return [2]int{left, left}, false
}

// runSet sets a value in a run container
func (c *container) runSet(value uint16) bool {
	search, found := c.runFind(value)
	if found {
		return false // Value already exists
	}

	runs := c.run()
	idx := search[1]

	// Check boundary cases for merging/extending
	canMergeLeft := idx > 0 && len(runs) > 0 && runs[idx-1][1]+1 == value
	canMergeRight := idx < len(runs) && len(runs) > 0 && runs[idx][0]-1 == value

	switch {
	case canMergeLeft && canMergeRight:
		runs[idx-1][1] = runs[idx][1]
		c.runRemoveRunAt(idx)
	case canMergeLeft:
		runs[idx-1][1] = value
	case canMergeRight:
		runs[idx][0] = value
	default:
		c.runInsertRunAt(idx, run{value, value})
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

	runs := c.run()
	idx := search[0]
	r := runs[idx]

	switch {
	case r[0] == r[1]:
		c.runRemoveRunAt(idx)
	case value == r[0]:
		runs[idx][0] = value + 1
	case value == r[1]:
		runs[idx][1] = value - 1
	default:
		leftRun := run{r[0], value - 1}
		rightRun := run{value + 1, r[1]}
		runs[idx] = leftRun
		c.runInsertRunAt(idx+1, rightRun)
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
func (c *container) runInsertRunAt(index int, newRun run) {
	runs := c.run()
	oldLen := len(runs)
	newCapacity := (oldLen + 1) * 2

	// Try to avoid allocation if we have enough capacity
	if cap(c.Data) >= newCapacity {
		// Extend slice without reallocation
		c.Data = c.Data[:newCapacity]
		newRuns := c.run()

		// Move existing runs to make space
		if index < oldLen {
			copy(newRuns[index+1:], runs[index:])
		}
		newRuns[index] = newRun
	} else {
		// Need to allocate new slice with extra capacity for future insertions
		extraCapacity := max(8, oldLen/2) // Add 25% extra capacity or minimum 8
		c.Data = make([]uint16, newCapacity, newCapacity+extraCapacity*2)
		newRuns := c.run()

		// Copy existing runs with efficient bulk operations
		if index > 0 {
			copy(newRuns[:index], runs[:index])
		}
		newRuns[index] = newRun
		if index < oldLen {
			copy(newRuns[index+1:], runs[index:])
		}
	}
}

// runRemoveRunAt removes the run at the specified index
func (c *container) runRemoveRunAt(index int) {
	runs := c.run()
	if index < 0 || index >= len(runs) {
		return
	}

	oldLen := len(runs)
	if oldLen == 1 {
		c.Data = c.Data[:0] // Keep capacity but set length to 0
		return
	}

	// Move runs in-place to avoid allocation
	copy(runs[index:], runs[index+1:])
	c.Data = c.Data[:(oldLen-1)*2] // Shrink slice length
}

// runOptimize tries to optimize the container
func (c *container) runOptimize() {
	if c.Type != typeRun || c.Size == 0 {
		return
	}

	numRuns := len(c.run())
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
	src := c.run()

	// Create new array data
	c.Data = make([]uint16, c.Size)
	c.Type = typeArray
	dst := c.arr()

	// Copy all values to the array
	idx := 0
	for _, r := range src {
		for value := r[0]; value <= r[1]; value++ {
			dst[idx] = value
			idx++
			if value == r[1] {
				break // Prevent uint16 overflow when r[1] is 65535
			}
		}
	}
}

// runToBmp converts this container from run to bitmap
func (c *container) runToBmp() {
	src := c.run()

	// Create bitmap data (65536 bits = 8192 bytes = 4096 uint16s)
	c.Data = make([]uint16, 4096)
	c.Type = typeBitmap
	dst := c.bmp()

	for _, r := range src {
		for i := r[0]; i <= r[1]; i++ {
			dst.Set(uint32(i))
			if i == r[1] {
				break // Prevent uint16 overflow when r[1] is 65535
			}
		}
	}
}

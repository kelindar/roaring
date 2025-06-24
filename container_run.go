package roaring

import "unsafe"

// run converts the container to a []run
func (c *container) run() []run {
	if len(c.Data) == 0 {
		return nil
	}

	return unsafe.Slice((*run)(unsafe.Pointer(&c.Data[0])), len(c.Data)/4)
}

// runFind performs binary search to find a value in the run container
func (c *container) runFind(value uint16) ([2]int, bool) {
	runs := c.run()
	if len(runs) == 0 {
		return [2]int{-1, -1}, false
	}

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
	canMergeLeft := idx > 0 && runs[idx-1][1]+1 == value
	canMergeRight := idx < len(runs) && runs[idx][0]-1 == value

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
	
	// Pre-allocate with some extra capacity to reduce future reallocations 
	newCapacity := (oldLen+1)*4 + min(oldLen*2, 64)
	newData := make([]byte, (oldLen+1)*4, newCapacity)
	newRuns := unsafe.Slice((*run)(unsafe.Pointer(&newData[0])), oldLen+1)
	
	// Copy existing runs with efficient bulk operations
	if index > 0 {
		copy(newRuns[:index], runs[:index])
	}
	newRuns[index] = newRun
	if index < oldLen {
		copy(newRuns[index+1:], runs[index:])
	}
	
	c.Data = newData
}

// runRemoveRunAt removes the run at the specified index
func (c *container) runRemoveRunAt(index int) {
	runs := c.run()
	if index < 0 || index >= len(runs) {
		return
	}
	
	oldLen := len(runs)
	if oldLen == 1 {
		c.Data = nil
		return
	}

	// Reallocate data for smaller run array
	c.Data = make([]byte, (oldLen-1)*4)
	newRuns := c.run()
	
	// Copy runs efficiently, skipping the removed index
	if index > 0 {
		copy(newRuns[:index], runs[:index])
	}
	if index < oldLen-1 {
		copy(newRuns[index:], runs[index+1:])
	}
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
	case numRuns > 2048:
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
	c.Data = make([]byte, c.Size*2)
	c.Type = typeArray
	dst := c.arr()

	// Copy all values to the array
	for _, r := range src {
		for value := r[0]; value <= r[1]; value++ {
			dst = append(dst, value)
			if value == r[1] {
				break // Prevent uint16 overflow when r[1] is 65535
			}
		}
	}
}

// runToBmp converts this container from run to bitmap
func (c *container) runToBmp() {
	src := c.run()

	// Create bitmap data (65536 bits = 8192 bytes)
	c.Data = make([]byte, 8192)
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

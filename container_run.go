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
	c.runOptimize()
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
	newRuns := make([]run, len(runs)+1)
	copy(newRuns[:index], runs[:index])
	newRuns[index] = newRun
	copy(newRuns[index+1:], runs[index:])

	c.Data = make([]byte, len(newRuns)*4)
	c.Type = typeRun
	copy(c.run(), newRuns)
}

// runRemoveRunAt removes the run at the specified index
func (c *container) runRemoveRunAt(index int) {
	runs := c.run()
	if index < 0 || index >= len(runs) {
		return
	}

	newRuns := make([]run, len(runs)-1)
	copy(newRuns[:index], runs[:index])
	copy(newRuns[index:], runs[index+1:])

	if len(newRuns) == 0 {
		c.Data = nil
	} else {
		c.Data = make([]byte, len(newRuns)*4)
		copy(c.run(), newRuns)
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
	runs := c.run()
	var values []uint16

	for _, r := range runs {
		for value := r[0]; value <= r[1]; value++ {
			values = append(values, value)
			if value == r[1] {
				break // Prevent uint16 overflow when r[1] is 65535
			}
		}
	}

	c.Data = make([]byte, len(values)*2)
	c.Type = typeArray
	c.Size = uint32(len(values))
	copy(c.arr(), values)
}

// runToBmp converts this container from run to bitmap
func (c *container) runToBmp() {
	runs := c.run()
	c.Data = make([]byte, 8192)
	c.Type = typeBitmap
	bm := c.bmp()

	for _, r := range runs {
		for i := r[0]; i <= r[1]; i++ {
			bm.Set(uint32(i))
			if i == r[1] {
				break // Prevent uint16 overflow when r[1] is 65535
			}
		}
	}

	c.Size = uint32(bm.Count())
}

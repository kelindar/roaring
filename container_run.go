package roaring

import "unsafe"

// run converts the container to a []run
func (c *container) run() []run {
	if len(c.Data) == 0 {
		return nil
	}

	return unsafe.Slice((*run)(unsafe.Pointer(&c.Data[0])), len(c.Data)/4)
}

// runSet sets a value in a run container using binary search and efficient boundary cases
func (c *container) runSet(value uint16) bool {
	runs := c.run()

	if len(runs) == 0 {
		// Empty container, add first run
		c.runInsertRunAt(0, run{value, value})
		c.Size++ // Update cardinality for the new value
		return true
	}

	// Binary search to find the relevant run or insertion point
	left, right := 0, len(runs)-1

	for left <= right {
		mid := (left + right) / 2
		r := runs[mid]

		if value >= r[0] && value <= r[1] {
			// Value already exists in this run
			return false
		} else if value < r[0] {
			right = mid - 1
		} else {
			left = mid + 1
		}
	}

	// At this point, 'left' is the insertion index
	insertIndex := left

	// Check boundary cases for merging/extending
	canMergeLeft := insertIndex > 0 && runs[insertIndex-1][1]+1 == value
	canMergeRight := insertIndex < len(runs) && runs[insertIndex][0]-1 == value

	if canMergeLeft && canMergeRight {
		// Merge two adjacent runs: extend left run and remove right run
		runs[insertIndex-1][1] = runs[insertIndex][1]
		c.runRemoveRunAt(insertIndex)
	} else if canMergeLeft {
		// Extend the previous run to the right
		runs[insertIndex-1][1] = value
	} else if canMergeRight {
		// Extend the next run to the left
		runs[insertIndex][0] = value
	} else {
		// Insert new single-value run
		c.runInsertRunAt(insertIndex, run{value, value})
	}

	// Update cardinality after modification
	c.Size++

	// Execute three integer comparisons for conversion check
	numRuns := len(c.run())
	cardinality := int(c.Size)

	// Convert to array if: small cardinality and few runs
	if cardinality <= 4096 && numRuns >= cardinality/2 {
		c.runToArray()
		return true
	}

	// Convert to bitmap if: too many runs or high density
	if numRuns > 2048 || cardinality > 32768 {
		c.runToBmp()
		return true
	}

	return true
}

// runDel removes a value from a run container using binary search
func (c *container) runDel(value uint16) bool {
	runs := c.run()

	if len(runs) == 0 {
		return false // Empty container
	}

	// Binary search to find the run containing the value
	left, right := 0, len(runs)-1
	runIndex := -1

	for left <= right {
		mid := (left + right) / 2
		r := runs[mid]

		if value >= r[0] && value <= r[1] {
			// Found the run containing the value
			runIndex = mid
			break
		} else if value < r[0] {
			right = mid - 1
		} else {
			left = mid + 1
		}
	}

	if runIndex == -1 {
		return false // Value not found
	}

	// Handle boundary cases for removal
	r := runs[runIndex]

	if r[0] == r[1] {
		// Run contains only this value, remove entire run
		c.runRemoveRunAt(runIndex)
	} else if value == r[0] {
		// Value is at start of run, increment start
		runs[runIndex][0] = value + 1
	} else if value == r[1] {
		// Value is at end of run, decrement end
		runs[runIndex][1] = value - 1
	} else {
		// Value is in middle of run, split into two runs
		leftRun := run{r[0], value - 1}
		rightRun := run{value + 1, r[1]}

		// Replace current run with left run
		runs[runIndex] = leftRun

		// Insert right run after current position
		c.runInsertRunAt(runIndex+1, rightRun)
	}

	// Update cardinality after modification
	c.Size--

	return true
}

// runHas checks if a value exists in a run container using binary search
func (c *container) runHas(value uint16) bool {
	runs := c.run()

	if len(runs) == 0 {
		return false
	}

	// Binary search to find the run containing the value
	left, right := 0, len(runs)-1

	for left <= right {
		mid := (left + right) / 2
		r := runs[mid]

		if value >= r[0] && value <= r[1] {
			return true // Found the value in this run
		} else if value < r[0] {
			right = mid - 1
		} else {
			left = mid + 1
		}
	}

	return false // Value not found
}

// runInsertRunAt inserts a new run at the specified index
func (c *container) runInsertRunAt(index int, newRun run) {
	runs := c.run()

	// Create new slice with space for one more run
	newRuns := make([]run, len(runs)+1)

	// Copy runs before insertion point
	copy(newRuns[:index], runs[:index])

	// Insert new run
	newRuns[index] = newRun

	// Copy runs after insertion point
	copy(newRuns[index+1:], runs[index:])

	// Update container data
	c.Data = make([]byte, len(newRuns)*4)
	c.Type = typeRun
	finalRuns := c.run()
	copy(finalRuns, newRuns)
}

// runRemoveRunAt removes the run at the specified index
func (c *container) runRemoveRunAt(index int) {
	runs := c.run()

	if index < 0 || index >= len(runs) {
		return
	}

	// Create new slice without the run at index
	newRuns := make([]run, len(runs)-1)

	// Copy runs before removal point
	copy(newRuns[:index], runs[:index])

	// Copy runs after removal point
	copy(newRuns[index:], runs[index+1:])

	// Update container data
	if len(newRuns) == 0 {
		c.Data = nil
	} else {
		c.Data = make([]byte, len(newRuns)*4)
		finalRuns := c.run()
		copy(finalRuns, newRuns)
	}
}

// runToArray converts this container from run to array
func (c *container) runToArray() {
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
	array := c.arr()
	copy(array, values)
}

// runToBmp converts this container from run to bitmap
func (c *container) runToBmp() {
	runs := c.run()

	// Create bitmap data (65536 bits = 8192 bytes)
	c.Data = make([]byte, 8192)
	c.Type = typeBitmap
	bm := c.bmp()

	// Set all bits from the runs
	for _, r := range runs {
		for i := r[0]; i <= r[1]; i++ {
			bm.Set(uint32(i))
			if i == r[1] {
				break // Prevent uint16 overflow when r[1] is 65535
			}
		}
	}

	// Update cardinality from bitmap
	c.Size = uint16(bm.Count())
}

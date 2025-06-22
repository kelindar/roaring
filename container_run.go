package roaring

// runSet sets a value in a run container using binary search and efficient boundary cases
func (c *container) runSet(value uint16) bool {
	runs := c.run()

	if len(runs) == 0 {
		// Empty container, add first run
		c.runInsertRunAt(0, run{value, value})
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
		c.bitmapConvertFromRun()
		c.arrayConvertFromBitmap()
		return true
	}

	// Convert to bitmap if: too many runs or high density
	if numRuns > 2048 || cardinality > 32768 {
		c.bitmapConvertFromRun()
		return true
	}

	return true
}

// runRemove removes a value from a run container using binary search
func (c *container) runRemove(value uint16) bool {
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

	// Execute three integer comparisons for conversion check
	numRuns := len(c.run())
	cardinality := int(c.Size)

	// Convert to array if: small cardinality and few runs
	if cardinality <= 4096 && numRuns >= cardinality/2 {
		c.bitmapConvertFromRun()
		c.arrayConvertFromBitmap()
		return true
	}

	// Convert to bitmap if: too many runs or high density
	if numRuns > 2048 || cardinality > 32768 {
		c.bitmapConvertFromRun()
		return true
	}

	return true
}

// runContains checks if a value exists in a run container using binary search
func (c *container) runContains(value uint16) bool {
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

// runShouldConvert returns true if run container should convert to bitmap
func (c *container) runShouldConvert() bool {
	runs := c.run()
	numRuns := len(runs)
	cardinality := int(c.Size)

	// Convert to bitmap if we have too many runs (run container becomes inefficient)
	if numRuns > 100 {
		return true
	}

	// Convert to bitmap if density is high (similar to array threshold)
	if cardinality > 4096 {
		return true
	}

	return false
}

// runConvertFromBitmap converts this container from bitmap to run
func (c *container) runConvertFromBitmap() {
	bm := c.bitmap()
	cardinality := c.Size // Preserve cardinality
	var runs []run

	// Find consecutive ranges in the bitmap
	var currentStart uint16 = 0
	var inRun bool = false

	for i := uint32(0); i < 65536; i++ {
		value := uint16(i)
		if bm.Contains(i) {
			if !inRun {
				// Start of new run
				currentStart = value
				inRun = true
			}
			// Continue run
		} else {
			if inRun {
				// End of current run
				runs = append(runs, run{currentStart, value - 1})
				inRun = false
			}
		}
	}

	// Handle case where last run extends to the end
	if inRun {
		runs = append(runs, run{currentStart, 65535})
	}

	// Create new run data
	c.Data = make([]byte, len(runs)*4) // 4 bytes per run (2 uint16s)
	c.Type = typeRun
	c.Size = cardinality // Restore cardinality
	newRuns := c.run()
	copy(newRuns, runs)
}

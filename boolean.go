package roaring

import (
	"math/bits"
)

// And performs bitwise AND operation with other bitmap(s)
func (rb *Bitmap) And(other *Bitmap, extra ...*Bitmap) {
	// Handle nil inputs efficiently
	if other == nil {
		rb.Clear()
		return
	}

	// Create list of valid bitmaps
	bitmaps := make([]*Bitmap, 0, len(extra)+1)
	bitmaps = append(bitmaps, other)
	for _, bm := range extra {
		if bm != nil && len(bm.indices) > 0 {
			bitmaps = append(bitmaps, bm)
		}
	}

	// If no valid bitmaps or empty result, clear
	if len(bitmaps) == 0 || len(rb.indices) == 0 {
		rb.Clear()
		return
	}

	// Single bitmap optimization
	if len(bitmaps) == 1 {
		rb.andSingle(bitmaps[0])
		return
	}

	// Multiple bitmaps - use iterative approach
	for _, bm := range bitmaps {
		if len(rb.indices) == 0 {
			break // Early exit
		}
		rb.andSingle(bm)
	}
}

// andSingle performs AND with a single bitmap efficiently
func (rb *Bitmap) andSingle(other *Bitmap) {
	if other == nil || len(other.indices) == 0 {
		rb.Clear()
		return
	}

	// Find intersection of container ranges
	if len(rb.indices) == 0 || len(other.indices) == 0 {
		rb.Clear()
		return
	}

	minStart := max(rb.indices[0], other.indices[0])
	minEnd := min(rb.indices[len(rb.indices)-1], other.indices[len(other.indices)-1])

	if minStart > minEnd {
		rb.Clear()
		return
	}

	// Track containers that become empty for batch removal
	emptyContainers := make([]uint16, 0, 8)

	// Use two pointers to efficiently iterate through sorted indices
	i, j := 0, 0
	for i < len(rb.indices) && j < len(other.indices) {
		hi1, hi2 := rb.indices[i], other.indices[j]

		switch {
		case hi1 == hi2:
			// Both bitmaps have containers at this index - perform AND
			c1, c2 := &rb.containers[i], &other.containers[j]
			if !rb.andContainers(c1, c2) {
				emptyContainers = append(emptyContainers, hi1)
			}
			i++
			j++

		case hi1 < hi2:
			// rb has container but other doesn't - remove it
			emptyContainers = append(emptyContainers, hi1)
			i++

		case hi1 > hi2:
			// other has container but rb doesn't - skip
			j++
		}
	}

	// Remove any remaining containers in rb that don't exist in other
	for i < len(rb.indices) {
		emptyContainers = append(emptyContainers, rb.indices[i])
		i++
	}

	// Batch remove empty containers
	for _, hi := range emptyContainers {
		rb.ctrDel(hi)
	}
}

// andContainers performs efficient AND between two containers
func (rb *Bitmap) andContainers(c1, c2 *container) bool {
	// Ensure we own c1's data before modifying it (COW protection)
	c1.cowEnsureOwned()

	// Use most efficient algorithm based on container types
	switch {
	case c1.Type == typeArray && c2.Type == typeArray:
		return rb.andArrayArray(c1, c2)
	case c1.Type == typeArray && c2.Type == typeBitmap:
		return rb.andArrayBitmap(c1, c2)
	case c1.Type == typeBitmap && c2.Type == typeArray:
		return rb.andBitmapArray(c1, c2)
	case c1.Type == typeBitmap && c2.Type == typeBitmap:
		return rb.andBitmapBitmap(c1, c2)
	case c1.Type == typeRun:
		return rb.andRunContainer(c1, c2)
	case c2.Type == typeRun:
		return rb.andContainerRun(c1, c2)
	default:
		return false
	}
}

// andArrayArray performs AND between two array containers
func (rb *Bitmap) andArrayArray(c1, c2 *container) bool {
	arr1 := c1.arr()
	arr2 := c2.arr()
	if len(arr1) == 0 || len(arr2) == 0 {
		return false
	}

	out := arr1[:0]
	i, j := 0, 0
	for i < len(arr1) && j < len(arr2) {
		switch {
		case arr1[i] == arr2[j]:
			out = append(out, arr1[i])
			i++
			j++
		case arr1[i] < arr2[j]:
			i++
		case arr1[i] > arr2[j]:
			j++
		}
	}

	c1.Data = out
	c1.Size = uint32(len(out))
	return true
}

// andArrayBitmap performs AND between array and bitmap containers
func (rb *Bitmap) andArrayBitmap(c1, c2 *container) bool {
	arr := c1.arr()
	bmp := c2.bmp()
	result := arr[:0]

	for _, val := range arr {
		if bmp.Contains(uint32(val)) {
			result = append(result, val)
		}
	}

	if len(result) == 0 {
		return false
	}

	c1.Data = result
	c1.Size = uint32(len(result))
	return true
}

// andBitmapArray performs AND between bitmap and array containers
func (rb *Bitmap) andBitmapArray(c1, c2 *container) bool {
	// Convert bitmap to array with only intersecting values
	bmp := c1.bmp()
	arr := c2.arr()

	// Save original bitmap state before clearing
	original := make([]uint64, len(bmp))
	copy(original, bmp)

	// Clear bitmap first
	for i := range bmp {
		bmp[i] = 0
	}

	count := 0
	for _, val := range arr {
		// Check against original bitmap state
		blkIdx := val >> 6
		bitIdx := val & 63
		if int(blkIdx) < len(original) {
			if original[blkIdx]&(uint64(1)<<bitIdx) != 0 {
				// Set bit in cleared bitmap
				bmp[blkIdx] |= uint64(1) << bitIdx
				count++
			}
		}
	}

	if count == 0 {
		return false
	}

	c1.Size = uint32(count)

	// Convert to array if small enough
	if count <= arrMinSize {
		rb.bitmapToArray(c1)
	}

	return true
}

// andBitmapBitmap performs AND between two bitmap containers
func (rb *Bitmap) andBitmapBitmap(c1, c2 *container) bool {
	bmp1 := c1.bmp()
	bmp2 := c2.bmp()
	if bmp1 == nil || bmp2 == nil {
		return false
	}

	// Ensure we process the shorter bitmap
	minLen := len(bmp1)
	if len(bmp2) < minLen {
		minLen = len(bmp2)
	}

	count := 0
	// Process word by word for better performance
	for i := 0; i < minLen; i++ {
		word := bmp1[i] & bmp2[i]
		bmp1[i] = word
		if word != 0 {
			count += bits.OnesCount64(word)
		}
	}

	// Clear remaining words in bmp1 if it's longer
	for i := minLen; i < len(bmp1); i++ {
		bmp1[i] = 0
	}

	if count == 0 {
		return false
	}

	c1.Size = uint32(count)

	// Convert to array if small enough
	if count <= arrMinSize {
		rb.bitmapToArray(c1)
	}

	return true
}

// andRunContainer performs AND between run container and other container
func (rb *Bitmap) andRunContainer(c1, c2 *container) bool {
	runs := c1.run()
	newRuns := make([]run, 0, len(runs))

	for _, r := range runs {
		start, end := r[0], r[1]

		// Check each value in run against other container
		currentStart := uint16(0)
		inRun := false

		for val := start; val <= end; val++ {
			if c2.contains(val) {
				if !inRun {
					currentStart = val
					inRun = true
				}
			} else if inRun {
				newRuns = append(newRuns, run{currentStart, val - 1})
				inRun = false
			}

			if val == end {
				break // Prevent overflow
			}
		}

		// Handle final run
		if inRun {
			newRuns = append(newRuns, run{currentStart, end})
		}
	}

	if len(newRuns) == 0 {
		return false
	}

	// Update container
	c1.Data = make([]uint16, len(newRuns)*2)
	newRunsSlice := c1.run()
	copy(newRunsSlice, newRuns)

	// Recalculate size
	c1.Size = 0
	for _, r := range newRuns {
		c1.Size += uint32(r[1] - r[0] + 1)
	}

	return true
}

// andContainerRun performs AND between container and run container
func (rb *Bitmap) andContainerRun(c1, c2 *container) bool {
	runs := c2.run()

	switch c1.Type {
	case typeArray:
		arr := c1.arr()
		result := arr[:0]

		for _, val := range arr {
			// Check if value is in any run
			for _, r := range runs {
				if val >= r[0] && val <= r[1] {
					result = append(result, val)
					break
				}
			}
		}

		if len(result) == 0 {
			return false
		}

		c1.Data = result
		c1.Size = uint32(len(result))
		return true

	case typeBitmap:
		bmp := c1.bmp()

		// Save original bitmap state before clearing
		original := make([]uint64, len(bmp))
		copy(original, bmp)

		// Clear bitmap
		for i := range bmp {
			bmp[i] = 0
		}

		count := 0
		// Set bits only for values in runs
		for _, r := range runs {
			for val := r[0]; val <= r[1]; val++ {
				blkIdx := val >> 6
				bitIdx := val & 63
				if int(blkIdx) < len(original) {
					if original[blkIdx]&(uint64(1)<<bitIdx) != 0 {
						bmp[blkIdx] |= uint64(1) << bitIdx
						count++
					}
				}
				if val == r[1] {
					break
				}
			}
		}

		if count == 0 {
			return false
		}

		c1.Size = uint32(count)
		return true
	}

	return false
}

// bitmapToArray converts bitmap container to array if efficient
func (rb *Bitmap) bitmapToArray(c *container) {
	if c.Type != typeBitmap || c.Size > arrMinSize {
		return
	}

	// Ensure we own the data before modifying (COW protection)
	c.cowEnsureOwned()

	bmp := c.bmp()
	arr := make([]uint16, 0, c.Size)

	for i, word := range bmp {
		if word != 0 {
			base := uint16(i * 64)
			for j := 0; j < 64; j++ {
				if word&(1<<j) != 0 {
					arr = append(arr, base+uint16(j))
				}
			}
		}
	}

	c.Data = arr
	c.Type = typeArray
}

// AndNot performs bitwise AND NOT operation with other bitmap(s)
func (rb *Bitmap) AndNot(other *Bitmap, extra ...*Bitmap) {
	panic("not implemented")
}

// Or performs bitwise OR operation with other bitmap(s)
func (rb *Bitmap) Or(other *Bitmap, extra ...*Bitmap) {
	panic("not implemented")
}

// Xor performs bitwise XOR operation with other bitmap(s)
func (rb *Bitmap) Xor(other *Bitmap, extra ...*Bitmap) {
	panic("not implemented")
}

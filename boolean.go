package roaring

import (
	"math/bits"
)

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the larger of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// maxUint8 returns the larger of two uint8 values
func maxUint8(a, b uint8) uint8 {
	if a > b {
		return a
	}
	return b
}

// minUint8 returns the smaller of two uint8 values
func minUint8(a, b uint8) uint8 {
	if a < b {
		return a
	}
	return b
}

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
		if bm != nil && bm.count > 0 {
			bitmaps = append(bitmaps, bm)
		}
	}

	// If no valid bitmaps or empty result, clear
	if len(bitmaps) == 0 || rb.count == 0 {
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
		if rb.count == 0 {
			break // Early exit
		}
		rb.andSingle(bm)
	}
}

// andSingle performs AND with a single bitmap efficiently
func (rb *Bitmap) andSingle(other *Bitmap) {
	if other.count == 0 {
		rb.Clear()
		return
	}

	// Find intersection of container ranges using span optimization
	minStart := maxUint8(rb.span[0], other.span[0])
	minEnd := minUint8(rb.span[1], other.span[1])

	if minStart > minEnd {
		rb.Clear()
		return
	}

	// Pre-allocate emptyContainers with better capacity estimation
	// Most containers will likely survive the AND operation, estimate conservatively
	maxPossibleEmpty := int(rb.count)
	if maxPossibleEmpty > 64 {
		maxPossibleEmpty = 64 // Cap to reasonable limit
	}
	emptyContainers := make([]uint16, 0, maxPossibleEmpty)

	// Process only intersecting containers
	for i := int(minStart); i <= int(minEnd); i++ {
		block := rb.blocks[i]
		otherBlock := other.blocks[i]

		if block == nil || otherBlock == nil {
			// No intersection at this block level
			if block != nil {
				for j := int(block.span[0]); j <= int(block.span[1]); j++ {
					if c := block.content[j]; c != nil {
						emptyContainers = append(emptyContainers, c.Key)
					}
				}
			}
			continue
		}

		// Find intersection within blocks
		blockMinStart := maxUint8(block.span[0], otherBlock.span[0])
		blockMinEnd := minUint8(block.span[1], otherBlock.span[1])

		if blockMinStart > blockMinEnd {
			// No intersection at container level - remove entire block
			for j := int(block.span[0]); j <= int(block.span[1]); j++ {
				if c := block.content[j]; c != nil {
					emptyContainers = append(emptyContainers, c.Key)
				}
			}
			continue
		}

		// Process containers within intersecting range
		for j := int(blockMinStart); j <= int(blockMinEnd); j++ {
			c1 := block.content[j]
			c2 := otherBlock.content[j]

			if c1 == nil || c2 == nil {
				if c1 != nil {
					emptyContainers = append(emptyContainers, c1.Key)
				}
				continue
			}

			// Perform container AND
			if !rb.andContainers(c1, c2) {
				emptyContainers = append(emptyContainers, c1.Key)
			}
		}

		// Remove containers outside intersection range
		for j := int(block.span[0]); j < int(blockMinStart); j++ {
			if c := block.content[j]; c != nil {
				emptyContainers = append(emptyContainers, c.Key)
			}
		}
		for j := int(blockMinEnd) + 1; j <= int(block.span[1]); j++ {
			if c := block.content[j]; c != nil {
				emptyContainers = append(emptyContainers, c.Key)
			}
		}
	}

	// Remove containers outside intersection range
	for i := int(rb.span[0]); i < int(minStart); i++ {
		if block := rb.blocks[i]; block != nil {
			for j := int(block.span[0]); j <= int(block.span[1]); j++ {
				if c := block.content[j]; c != nil {
					emptyContainers = append(emptyContainers, c.Key)
				}
			}
		}
	}
	for i := int(minEnd) + 1; i <= int(rb.span[1]); i++ {
		if block := rb.blocks[i]; block != nil {
			for j := int(block.span[0]); j <= int(block.span[1]); j++ {
				if c := block.content[j]; c != nil {
					emptyContainers = append(emptyContainers, c.Key)
				}
			}
		}
	}

	// Batch remove empty containers
	for _, hi := range emptyContainers {
		rb.removeContainer(hi)
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
		c1.Size = 0
		return false
	}

	// Optimize for very small arrays - linear search can be faster
	if len(arr1) <= 8 && len(arr2) <= 8 {
		result := make([]uint16, 0, min(len(arr1), len(arr2)))
		for _, v1 := range arr1 {
			for _, v2 := range arr2 {
				if v1 == v2 {
					result = append(result, v1)
					break
				}
			}
		}
		c1.Data = result
		c1.Size = uint32(len(result))
		return len(result) > 0
	}

	// Two-pointer approach for larger arrays
	result := make([]uint16, 0, min(len(arr1), len(arr2)))
	i, j := 0, 0
	for i < len(arr1) && j < len(arr2) {
		switch {
		case arr1[i] == arr2[j]:
			result = append(result, arr1[i])
			i++
			j++
		case arr1[i] < arr2[j]:
			i++
		case arr1[i] > arr2[j]:
			j++
		}
	}

	c1.Data = result
	c1.Size = uint32(len(result))
	return len(result) > 0
}

// andArrayBitmap performs AND between array and bitmap containers
func (rb *Bitmap) andArrayBitmap(c1, c2 *container) bool {
	arr := c1.arr()
	bmp := c2.bmp()
	
	if len(arr) == 0 || len(bmp) == 0 {
		c1.Size = 0
		return false
	}
	
	// Pre-allocate result with better size estimation
	resultCapacity := len(arr)
	if resultCapacity > 64 {
		resultCapacity = 64 // Cap for memory efficiency
	}
	result := make([]uint16, 0, resultCapacity)

	// Direct bit manipulation is faster than Contains() method calls
	for _, val := range arr {
		word := val >> 6       // Which 64-bit word
		bit := val & 63        // Which bit within the word
		if int(word) < len(bmp) && (bmp[word]&(uint64(1)<<bit)) != 0 {
			result = append(result, val)
		}
	}

	// Replace the data
	c1.Data = result
	c1.Size = uint32(len(result))
	return len(result) > 0
}

// andBitmapArray performs AND between bitmap and array containers
func (rb *Bitmap) andBitmapArray(c1, c2 *container) bool {
	originalBmp := c1.bmp()
	arr := c2.arr()

	if len(arr) == 0 || len(originalBmp) == 0 {
		c1.Size = 0
		return false
	}

	// Create intersection array directly - more efficient than bitmap operations
	result := make([]uint16, 0, len(arr))
	
	// Direct bit manipulation for better performance
	for _, val := range arr {
		word := val >> 6       // Which 64-bit word
		bit := val & 63        // Which bit within the word
		if int(word) < len(originalBmp) && (originalBmp[word]&(uint64(1)<<bit)) != 0 {
			result = append(result, val)
		}
	}

	if len(result) == 0 {
		c1.Size = 0
		return false
	}

	// Convert container to array type for efficiency
	c1.Data = result
	c1.Type = typeArray
	c1.Size = uint32(len(result))
	return true
}

// andBitmapBitmap performs AND between two bitmap containers
func (rb *Bitmap) andBitmapBitmap(c1, c2 *container) bool {
	bmp1 := c1.bmp()
	bmp2 := c2.bmp()
	if bmp1 == nil || bmp2 == nil {
		c1.Size = 0
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

	c1.Size = uint32(count)

	if count == 0 {
		return false
	}

	// Convert to array if small enough for better space efficiency
	if count <= arrMinSize {
		rb.bitmapToArray(c1)
	}

	return true
}

// andRunContainer performs AND between run container and other container
func (rb *Bitmap) andRunContainer(c1, c2 *container) bool {
	runs := c1.run()
	// Pre-allocate with reasonable capacity to avoid growth
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
		c1.Size = 0
		return false
	}

	// Reuse existing data slice if possible
	if cap(c1.Data) >= len(newRuns)*2 {
		c1.Data = c1.Data[:len(newRuns)*2]
	} else {
		c1.Data = make([]uint16, len(newRuns)*2)
	}
	
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
		result := make([]uint16, 0, len(arr)) // Create new slice to avoid corruption

		for _, val := range arr {
			// Check if value is in any run
			for _, r := range runs {
				if val >= r[0] && val <= r[1] {
					result = append(result, val)
					break
				}
			}
		}

		c1.Data = result
		c1.Size = uint32(len(result))
		return len(result) > 0

	case typeBitmap:
		bmp := c1.bmp()
		
		// Create a copy to check original values, then clear the original
		original := make([]uint64, len(bmp))
		copy(original, bmp)
		
		// Clear bitmap
		for i := range bmp {
			bmp[i] = 0
		}

		count := 0
		// Set bits for intersecting values
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

		c1.Size = uint32(count)
		return count > 0
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

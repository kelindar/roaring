package roaring

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
		if bm != nil && len(bm.containers) > 0 {
			bitmaps = append(bitmaps, bm)
		}
	}

	// If no valid bitmaps or empty result, clear
	if len(bitmaps) == 0 || len(rb.containers) == 0 {
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
		if len(rb.containers) == 0 {
			break // Early exit
		}
		rb.andSingle(bm)
	}
}

// andSingle performs AND with a single bitmap efficiently
func (rb *Bitmap) andSingle(other *Bitmap) {
	if other == nil || len(other.containers) == 0 {
		rb.Clear()
		return
	}

	// If this bitmap is empty, result is empty
	if len(rb.containers) == 0 {
		return
	}

	// Track containers that become empty for batch removal
	emptyIndices := make([]int, 0, 8)

	// Iterate through all containers in this bitmap
	for i := range rb.containers {
		c1 := &rb.containers[i]
		hi := rb.index[i]

		// Check if other bitmap has a container at this key
		idx, exists := find16(other.index, hi)
		if !exists {
			// Other bitmap doesn't have this container - mark for removal
			emptyIndices = append(emptyIndices, i)
			continue
		}

		// Both bitmaps have containers at this index - perform AND
		c2 := &other.containers[idx]
		if !rb.andContainers(c1, c2) {
			emptyIndices = append(emptyIndices, i)
		}
	}

	// Batch remove empty containers (in reverse order to maintain indices)
	for i := len(emptyIndices) - 1; i >= 0; i-- {
		rb.ctrDel(emptyIndices[i])
	}
}

// andContainers performs efficient AND between two containers
func (rb *Bitmap) andContainers(c1, c2 *container) bool {
	c1.cowEnsureOwned()

	// Use most efficient algorithm based on container types
	switch {
	case c1.Type == typeArray && c2.Type == typeArray:
		return rb.arrAndArr(c1, c2)
	case c1.Type == typeArray && c2.Type == typeBitmap:
		return rb.arrAndBmp(c1, c2)
	case c1.Type == typeBitmap && c2.Type == typeArray:
		return rb.bmpAndArr(c1, c2)
	case c1.Type == typeBitmap && c2.Type == typeBitmap:
		return rb.bmpAndBmp(c1, c2)
	case c1.Type == typeRun:
		return rb.andRunContainer(c1, c2)
	case c2.Type == typeRun:
		return rb.andContainerRun(c1, c2)
	default:
		return false
	}
}

// arrAndArr performs AND between two array containers
func (rb *Bitmap) arrAndArr(c1, c2 *container) bool {
	a, b := c1.Data, c2.Data
	i, j, k := 0, 0, 0
	for i < len(a) && j < len(b) {
		av, bv := a[i], b[j]
		switch {
		case av == bv:
			a[k] = av
			k++
			i++
			j++
		case av < bv:
			i++
		default: // av > bv
			j++
		}
	}

	c1.Data = a[:k]
	c1.Size = uint32(len(c1.Data))
	return true
}

// arrAndBmp performs AND between array and bitmap containers
func (rb *Bitmap) arrAndBmp(c1, c2 *container) bool {
	a, b := c1.Data, c2.bmp()
	out := a[:0]

	for _, val := range a {
		if b.Contains(uint32(val)) {
			out = append(out, val)
		}
	}

	c1.Data = out
	c1.Size = uint32(len(out))
	return true
}

// bmpAndArr performs AND between bitmap and array containers
func (rb *Bitmap) bmpAndArr(c1, c2 *container) bool {
	a, b := c1.bmp(), c2.Data
	out := make([]uint16, 0, len(b))

	for _, val := range b {
		if a.Contains(uint32(val)) {
			out = append(out, val)
		}
	}

	c1.Data = out
	c1.Size = uint32(len(out))
	return true
}

// bmpAndBmp performs AND between two bitmap containers
func (rb *Bitmap) bmpAndBmp(c1, c2 *container) bool {
	bmp1 := c1.bmp()
	bmp2 := c2.bmp()
	if bmp1 == nil || bmp2 == nil {
		return false
	}

	// Perform AND operation and update container size
	bmp1.And(bmp2)
	c1.Size = uint32(bmp1.Count())

	// Convert to array if small enough
	if c1.Size <= arrMinSize {
		rb.bitmapToArray(c1)
	}
	return true
}

// andRunContainer performs AND between run container and other container
func (rb *Bitmap) andRunContainer(c1, c2 *container) bool {
	numRuns := len(c1.Data) / 2
	newData := make([]uint16, 0, len(c1.Data))
	var newSize uint32

	for i := 0; i < numRuns; i++ {
		start, end := c1.Data[i*2], c1.Data[i*2+1]

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
				newData = append(newData, currentStart, val-1)
				newSize += uint32(val-1) - uint32(currentStart) + 1
				inRun = false
			}

			if val == end {
				break // Prevent overflow
			}
		}

		// Handle final run
		if inRun {
			newData = append(newData, currentStart, end)
			newSize += uint32(end) - uint32(currentStart) + 1
		}
	}

	if len(newData) == 0 {
		c1.Data = c1.Data[:0]
		c1.Size = 0
		return false
	}

	// Update container
	c1.Data = newData
	c1.Size = newSize
	return true
}

// andContainerRun performs AND between container and run container
func (rb *Bitmap) andContainerRun(c1, c2 *container) bool {
	numRuns := len(c2.Data) / 2

	switch c1.Type {
	case typeArray:
		arr := c1.Data
		result := arr[:0]

		for _, val := range arr {
			// Check if value is in any run
			for i := 0; i < numRuns; i++ {
				start, end := c2.Data[i*2], c2.Data[i*2+1]
				if val >= start && val <= end {
					result = append(result, val)
					break
				}
			}
		}

		c1.Data = result
		c1.Size = uint32(len(result))
		if c1.Size == 0 {
			return false
		}
		return true

	case typeBitmap:
		bmp := c1.bmp()
		newData := make([]uint16, 0, c1.Size)

		for i := 0; i < numRuns; i++ {
			start, end := c2.Data[i*2], c2.Data[i*2+1]
			for v := start; v <= end; v++ {
				if bmp.Contains(uint32(v)) {
					newData = append(newData, v)
				}
				if v == end {
					break
				}
			}
		}

		if len(newData) == 0 {
			c1.Data = c1.Data[:0]
			c1.Size = 0
			return false
		}

		c1.Data = newData
		c1.Size = uint32(len(newData))
		c1.Type = typeArray
		c1.optimize()
		return c1.Size > 0
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

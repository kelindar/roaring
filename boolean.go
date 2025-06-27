package roaring

// and performs AND with a single bitmap efficiently
func (rb *Bitmap) and(other *Bitmap) {
	switch {
	case other == nil || len(other.containers) == 0:
		rb.Clear()
		return
	case len(rb.containers) == 0:
		return
	}

	// Iterate through all containers in this bitmap
	rb.scratch = rb.scratch[:0]
	for i := range rb.containers {
		c1 := &rb.containers[i]
		idx, exists := find16(other.index, rb.index[i])
		switch {
		case !exists:
			rb.scratch = append(rb.scratch, uint32(i))
		case !c1.and(&other.containers[idx]):
			rb.scratch = append(rb.scratch, uint32(i))
		}
	}

	// Batch remove empty containers (in reverse order to maintain indices)
	for i := len(rb.scratch) - 1; i >= 0; i-- {
		rb.ctrDel(int(rb.scratch[i]))
	}
}

// arrAndArr performs AND between two array containers
func arrAndArr(c1, c2 *container) bool {
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
func arrAndBmp(c1, c2 *container) bool {
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

// arrAndRun performs AND between array and run containers
func arrAndRun(c1, c2 *container) bool {
	a, b := c1.Data, c2.Data
	out := a[:0]

	for _, val := range a {
		for i := 0; i < len(b)/2; i += 2 {
			if val >= b[i*2] && val <= b[i*2+1] {
				out = append(out, val)
				break
			}
		}
	}

	c1.Data = out
	c1.Size = uint32(len(out))
	return c1.Size > 0
}

// bmpAndArr performs AND between bitmap and array containers
func bmpAndArr(c1, c2 *container) bool {
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
func bmpAndBmp(c1, c2 *container) bool {
	bmp1 := c1.bmp()
	bmp2 := c2.bmp()
	if bmp1 == nil || bmp2 == nil {
		return false
	}

	// Perform AND operation and update container size
	bmp1.And(bmp2)
	c1.Size = uint32(bmp1.Count())
	return true
}

// andContainerRun performs AND between container and run container
func bmpAndRun(c1, c2 *container) bool {
	numRuns := len(c2.Data) / 2
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

// runAndCtr performs AND between run container and other container
func runAndCtr(c1, c2 *container) bool {
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

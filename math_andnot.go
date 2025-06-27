package roaring

// andNot performs AND NOT with a single bitmap efficiently
func (rb *Bitmap) andNot(other *Bitmap) {
	switch {
	case other == nil || len(other.containers) == 0:
		return // No change needed - A AND NOT ∅ = A
	case len(rb.containers) == 0:
		return // Empty bitmap AND NOT anything = empty
	}

	// Remove elements that are in other bitmap
	rb.scratch = rb.scratch[:0]
	for i := range rb.containers {
		c1 := &rb.containers[i]
		idx, exists := find16(other.index, rb.index[i])
		switch {
		case !exists:
			// Container not in other bitmap - keep as is
			continue
		case !rb.ctrAndNot(c1, &other.containers[idx]):
			// Container became empty - mark for removal
			rb.scratch = append(rb.scratch, uint16(i))
		}
	}

	// Batch remove empty containers (in reverse order to maintain indices)
	for i := len(rb.scratch) - 1; i >= 0; i-- {
		rb.ctrDel(int(rb.scratch[i]))
	}
}

// ctrAndNot performs efficient AND NOT between two containers
func (rb *Bitmap) ctrAndNot(c1, c2 *container) bool {
	c1.fork()
	switch c1.Type {
	case typeArray:
		switch c2.Type {
		case typeArray:
			return rb.arrAndNotArr(c1, c2)
		case typeBitmap:
			return rb.arrAndNotBmp(c1, c2)
		case typeRun:
			return rb.arrAndNotRun(c1, c2)
		}
	case typeBitmap:
		switch c2.Type {
		case typeArray:
			return rb.bmpAndNotArr(c1, c2)
		case typeBitmap:
			return rb.bmpAndNotBmp(c1, c2)
		case typeRun:
			return rb.bmpAndNotRun(c1, c2)
		}
	case typeRun:
		switch c2.Type {
		case typeArray:
			return rb.runAndNotArr(c1, c2)
		case typeBitmap:
			return rb.runAndNotBmp(c1, c2)
		case typeRun:
			return rb.runAndNotRun(c1, c2)
		}
	}
	return false
}

// arrAndNotArr performs AND NOT between two array containers
func (rb *Bitmap) arrAndNotArr(c1, c2 *container) bool {
	a, b := c1.Data, c2.Data
	out := a[:0]
	i, j := 0, 0

	for i < len(a) && j < len(b) {
		av, bv := a[i], b[j]
		switch {
		case av == bv:
			// Element in both - exclude from result
			i++
			j++
		case av < bv:
			// Only in first array - keep it
			out = append(out, av)
			i++
		default: // av > bv
			// Only in second array - skip it
			j++
		}
	}

	// Add remaining elements from first array
	for i < len(a) {
		out = append(out, a[i])
		i++
	}

	c1.Data = out
	c1.Size = uint32(len(out))
	return c1.Size > 0
}

// arrAndNotBmp performs AND NOT between array and bitmap containers
func (rb *Bitmap) arrAndNotBmp(c1, c2 *container) bool {
	a, b := c1.Data, c2.bmp()
	out := a[:0]

	for _, val := range a {
		if !b.Contains(uint32(val)) {
			out = append(out, val)
		}
	}

	c1.Data = out
	c1.Size = uint32(len(out))
	return c1.Size > 0
}

// arrAndNotRun performs AND NOT between array and run containers
func (rb *Bitmap) arrAndNotRun(c1, c2 *container) bool {
	a, runs := c1.Data, c2.Data
	out := a[:0]

	for _, val := range a {
		// Check if value is in any run
		inRun := false
		for i := 0; i < len(runs); i += 2 {
			if val >= runs[i] && val <= runs[i+1] {
				inRun = true
				break
			}
		}
		if !inRun {
			out = append(out, val)
		}
	}

	c1.Data = out
	c1.Size = uint32(len(out))
	return c1.Size > 0
}

// bmpAndNotArr performs AND NOT between bitmap and array containers
func (rb *Bitmap) bmpAndNotArr(c1, c2 *container) bool {
	bmp := c1.bmp()
	for _, val := range c2.Data {
		if bmp.Contains(uint32(val)) {
			bmp.Remove(uint32(val))
			c1.Size--
		}
	}
	return c1.Size > 0
}

// bmpAndNotBmp performs AND NOT between two bitmap containers
func (rb *Bitmap) bmpAndNotBmp(c1, c2 *container) bool {
	a, b := c1.bmp(), c2.bmp()
	if b == nil {
		return c1.Size > 0
	}

	a.AndNot(b)
	c1.Size = uint32(a.Count())
	return c1.Size > 0
}

// bmpAndNotRun performs AND NOT between bitmap and run containers
func (rb *Bitmap) bmpAndNotRun(c1, c2 *container) bool {
	bmp := c1.bmp()
	runs := c2.Data

	for i := 0; i < len(runs); i += 2 {
		start, end := runs[i], runs[i+1]
		for v := start; v <= end; v++ {
			if bmp.Contains(uint32(v)) {
				bmp.Remove(uint32(v))
				c1.Size--
			}
			if v == end {
				break // Prevent overflow
			}
		}
	}
	return c1.Size > 0
}

// runAndNotArr performs AND NOT between run and array containers
func (rb *Bitmap) runAndNotArr(c1, c2 *container) bool {
	runs, arr := c1.Data, c2.Data
	out := rb.scratch[:0]
	size := uint32(0)

	for i := 0; i < len(runs); i += 2 {
		start, end := runs[i], runs[i+1]

		// For each run, exclude elements that are in the array
		currStart := start
		for _, val := range arr {
			if val < currStart || val > end {
				continue // Value not in current run
			}

			// Add run segment before this value
			if currStart < val {
				out = append(out, currStart, val-1)
				size += uint32(val-1) - uint32(currStart) + 1
			}
			currStart = val + 1
		}

		// Add remaining part of run
		if currStart <= end {
			out = append(out, currStart, end)
			size += uint32(end) - uint32(currStart) + 1
		}
	}

	c1.Data = append(c1.Data[:0], out...)
	c1.Size = size
	rb.scratch = out
	return size > 0
}

// runAndNotBmp performs AND NOT between run and bitmap containers
func (rb *Bitmap) runAndNotBmp(c1, c2 *container) bool {
	runs, bmp := c1.Data, c2.bmp()
	out := rb.scratch[:0]
	size := uint32(0)

	for i := 0; i < len(runs); i += 2 {
		start, end := runs[i], runs[i+1]
		currStart := start

		for v := start; v <= end; v++ {
			if bmp.Contains(uint32(v)) {
				// Found element to exclude - add run before it
				if currStart < v {
					out = append(out, currStart, v-1)
					size += uint32(v-1) - uint32(currStart) + 1
				}
				currStart = v + 1
			}
			if v == end {
				break // Prevent overflow
			}
		}

		// Add remaining part of run
		if currStart <= end {
			out = append(out, currStart, end)
			size += uint32(end) - uint32(currStart) + 1
		}
	}

	c1.Data = append(c1.Data[:0], out...)
	c1.Size = size
	rb.scratch = out
	return size > 0
}

// runAndNotRun performs AND NOT between two run containers
func (rb *Bitmap) runAndNotRun(c1, c2 *container) bool {
	a, b := c1.Data, c2.Data
	out := rb.scratch[:0]
	size := uint32(0)
	i, j := 0, 0

	for i < len(a) {
		s1, e1 := a[i], a[i+1]

		// Find overlapping runs in second container
		currStart := s1
		for j < len(b) && b[j] <= e1 {
			s2, e2 := b[j], b[j+1]

			// Check for overlap
			if s2 <= e1 && e2 >= currStart {
				// Add segment before overlap
				if currStart < s2 {
					out = append(out, currStart, s2-1)
					size += uint32(s2-1) - uint32(currStart) + 1
				}

				// Move past this overlap
				if e2 >= e1 {
					// Second run extends past first run
					currStart = e1 + 1
					break
				} else {
					// Second run is contained in first run
					currStart = e2 + 1
				}
			}

			if e2 < e1 {
				j += 2
			} else {
				break
			}
		}

		// Add remaining part of first run
		if currStart <= e1 {
			out = append(out, currStart, e1)
			size += uint32(e1) - uint32(currStart) + 1
		}

		i += 2
	}

	c1.Data = append(c1.Data[:0], out...)
	c1.Size = size
	rb.scratch = out
	return size > 0
}

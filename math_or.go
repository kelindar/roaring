// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

package roaring

// or performs OR with a single bitmap efficiently
func (rb *Bitmap) or(other *Bitmap) {
	switch {
	case other == nil || len(other.containers) == 0:
		return // No change needed
	case len(rb.containers) == 0:
		// Copy all containers from other
		rb.containers = make([]container, len(other.containers))
		rb.index = make([]uint16, len(other.index))
		for i := range other.containers {
			other.containers[i].Shared = true
		}
		copy(rb.containers, other.containers)
		copy(rb.index, other.index)
		return
	}

	// Update the common prefix in place before allocating a merged index.
	i, j := 0, 0
	for i < len(rb.index) && j < len(other.index) && rb.index[i] == other.index[j] {
		rb.ctrOr(&rb.containers[i], &other.containers[j])
		i++
		j++
	}
	if j == len(other.index) {
		return
	}
	newContainers := make([]container, 0, len(rb.containers)+len(other.containers))
	newIndex := make([]uint16, 0, len(rb.index)+len(other.index))
	newContainers = append(newContainers, rb.containers[:i]...)
	newIndex = append(newIndex, rb.index[:i]...)

	for i < len(rb.containers) && j < len(other.containers) {
		hi1, hi2 := rb.index[i], other.index[j]
		switch {
		case hi1 < hi2:
			// Only in left bitmap
			newContainers = append(newContainers, rb.containers[i])
			newIndex = append(newIndex, hi1)
			i++
		case hi1 > hi2:
			// Only in right bitmap
			other.containers[j].Shared = true
			newContainers = append(newContainers, other.containers[j])
			newIndex = append(newIndex, hi2)
			j++
		default:
			// In both bitmaps - merge them
			c1 := &rb.containers[i]
			c2 := &other.containers[j]
			rb.ctrOr(c1, c2)
			newContainers = append(newContainers, *c1)
			newIndex = append(newIndex, hi1)
			i++
			j++
		}
	}

	// Add remaining containers from left
	for i < len(rb.containers) {
		newContainers = append(newContainers, rb.containers[i])
		newIndex = append(newIndex, rb.index[i])
		i++
	}

	// Add remaining containers from right
	for j < len(other.containers) {
		other.containers[j].Shared = true
		newContainers = append(newContainers, other.containers[j])
		newIndex = append(newIndex, other.index[j])
		j++
	}

	rb.containers = newContainers
	rb.index = newIndex
}

// ctrOr performs efficient OR between two containers
func (rb *Bitmap) ctrOr(c1, c2 *container) {
	c1.fork()
	switch c1.Type {
	case typeArray:
		switch c2.Type {
		case typeArray:
			rb.arrOrArr(c1, c2)
		case typeBitmap:
			rb.arrOrBmp(c1, c2)
		case typeRun:
			rb.arrOrRun(c1, c2)
		}
	case typeBitmap:
		switch c2.Type {
		case typeArray:
			rb.bmpOrArr(c1, c2)
		case typeBitmap:
			rb.bmpOrBmp(c1, c2)
		case typeRun:
			rb.bmpOrRun(c1, c2)
		}
	case typeRun:
		switch c2.Type {
		case typeArray:
			rb.runOrArr(c1, c2)
		case typeBitmap:
			rb.runOrBmp(c1, c2)
		case typeRun:
			rb.runOrRun(c1, c2)
		}
	}
}

// arrOrArr performs OR between two array containers
func (rb *Bitmap) arrOrArr(c1, c2 *container) {
	a, b := c1.Data, c2.Data
	out := rb.scratch[:0]
	if cap(out) < max(len(a), len(b)) {
		out = make([]uint16, 0, len(a)+len(b))
	}
	i, j := 0, 0

	if len(rb.index) >= branchlessAt {
		for i < len(a) && j < len(b) {
			av, bv := uint32(a[i]), uint32(b[j])
			less, greater := int((av-bv)>>31), int((bv-av)>>31)
			out = append(out, uint16(min(av, bv)))
			i += 1 - greater
			j += 1 - less
		}
	} else {
		for i < len(a) && j < len(b) {
			av, bv := a[i], b[j]
			switch {
			case av == bv:
				out = append(out, av)
				i++
				j++
			case av < bv:
				out = append(out, av)
				i++
			default: // av > bv
				out = append(out, bv)
				j++
			}
		}
	}

	// Add remaining elements
	for i < len(a) {
		out = append(out, a[i])
		i++
	}
	for j < len(b) {
		out = append(out, b[j])
		j++
	}

	rb.scratch = c1.Data[:0]
	c1.Data = out
	c1.Size = uint32(len(c1.Data))
}

// arrOrBmp performs OR between array and bitmap containers
func (rb *Bitmap) arrOrBmp(c1, c2 *container) {
	// Convert to bitmap for efficient OR
	c1.arrToBmp()
	rb.bmpOrBmp(c1, c2)
}

// arrOrRun performs OR between array and run containers
func (rb *Bitmap) arrOrRun(c1, c2 *container) {
	runs := c2.Data
	out := rb.scratch[:0]

	// Expand runs and merge with array
	runIdx := 0
	for _, val := range c1.Data {
		// Add runs that come before this value
		for runIdx*2+1 < len(runs) && runs[runIdx*2+1] < val {
			start, end := uint32(runs[runIdx*2]), uint32(runs[runIdx*2+1])
			for v := start; v <= end; v++ {
				out = append(out, uint16(v))
			}
			runIdx++
		}

		// Check if value is covered by current run
		if runIdx*2+1 < len(runs) && val >= runs[runIdx*2] && val <= runs[runIdx*2+1] {
			// Value is covered by run, skip it
			continue
		}

		out = append(out, val)
	}

	// Add remaining runs
	for runIdx*2+1 < len(runs) {
		start, end := uint32(runs[runIdx*2]), uint32(runs[runIdx*2+1])
		for v := start; v <= end; v++ {
			out = append(out, uint16(v))
		}
		runIdx++
	}

	c1.Data = append(c1.Data[:0], out...)
	c1.Size = uint32(len(c1.Data))
	c1.Type = typeArray
	rb.scratch = out
}

// bmpOrArr performs OR between bitmap and array containers
func (rb *Bitmap) bmpOrArr(c1, c2 *container) {
	bmp := c1.bmp()
	for _, val := range c2.Data {
		if !bmp.Contains(uint32(val)) {
			bmp.Set(uint32(val))
			c1.Size++
		}
	}
}

// bmpOrBmp performs OR between two bitmap containers
func (rb *Bitmap) bmpOrBmp(c1, c2 *container) {
	a, b := c1.bmp(), c2.bmp()
	if b == nil {
		return
	}

	a.Or(b)
	c1.Size = uint32(a.Count())
}

// bmpOrRun performs OR between bitmap and run containers
func (rb *Bitmap) bmpOrRun(c1, c2 *container) {
	bmp := c1.bmp()
	runs := c2.Data

	for i := 0; i < len(runs); i += 2 {
		start, end := uint32(runs[i]), uint32(runs[i+1])
		for v := start; v <= end; v++ {
			if !bmp.Contains(v) {
				bmp.Set(v)
				c1.Size++
			}
		}
	}
}

// runOrArr performs OR between run and array containers
func (rb *Bitmap) runOrArr(c1, c2 *container) {
	c1.runToArray()
	rb.arrOrArr(c1, c2)
	c1.optimize()
}

// runOrBmp performs OR between run and bitmap containers
func (rb *Bitmap) runOrBmp(c1, c2 *container) {
	c1.runToBmp()
	rb.bmpOrBmp(c1, c2)
}

// runOrRun performs OR between two run containers
func (rb *Bitmap) runOrRun(c1, c2 *container) {
	a, b := c1.Data, c2.Data
	out := rb.scratch[:0]
	if cap(out) < max(len(a), len(b)) {
		out = make([]uint16, 0, len(a)+len(b))
	}
	size := uint32(0)
	for len(a) > 0 || len(b) > 0 {
		var start, end uint16
		switch {
		case len(b) == 0:
			start, end = a[0], a[1]
			a = a[2:]
		case len(a) > 0 && a[0] <= b[0]:
			start, end = a[0], a[1]
			a = a[2:]
		default:
			start, end = b[0], b[1]
			b = b[2:]
		}
		if n := len(out); n > 0 && uint32(start) <= uint32(out[n-1])+1 {
			if end > out[n-1] {
				size += uint32(end) - uint32(out[n-1])
				out[n-1] = end
			}
		} else {
			out = append(out, start, end)
			size += uint32(end) - uint32(start) + 1
		}
	}
	rb.scratch = c1.Data[:0]
	c1.Data = out
	c1.Size = size
}

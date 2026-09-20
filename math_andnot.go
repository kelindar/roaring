// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

package roaring

// andNot performs AND NOT with a single bitmap efficiently
func (rb *Bitmap) andNot(other *Bitmap) {
	switch {
	case other == nil || len(other.containers) == 0:
		return // No change needed - A AND NOT ∅ = A
	case len(rb.containers) == 0:
		return // Empty bitmap AND NOT anything = empty
	}

	write, idx := 0, 0
	for read := range rb.containers {
		key := rb.index[read]
		c1 := &rb.containers[read]
		for idx < len(other.index) && other.index[idx] < key {
			idx++
		}
		exists := idx < len(other.index) && other.index[idx] == key
		switch {
		case !exists:
			// Keep containers that do not exist in the subtrahend.
		case !rb.ctrAndNot(c1, &other.containers[idx]):
			continue
		}

		if write != read {
			rb.containers[write] = rb.containers[read]
			rb.index[write] = key
		}
		write++
	}
	clearContainerTail(rb.containers, write)
	rb.containers = rb.containers[:write]
	rb.index = rb.index[:write]
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
	i, j, k := 0, 0, 0

	if len(rb.index) >= branchlessAt {
		for i < len(a) && j < len(b) {
			av, bv := uint32(a[i]), uint32(b[j])
			less, greater := int((av-bv)>>31), int((bv-av)>>31)
			a[k] = uint16(av)
			k += less
			i += 1 - greater
			j += 1 - less
		}
	} else {
		for i < len(a) && j < len(b) {
			av, bv := a[i], b[j]
			switch {
			case av == bv:
				i++
				j++
			case av < bv:
				a[k] = av
				k++
				i++
			default:
				j++
			}
		}
	}

	k += copy(a[k:], a[i:])

	c1.Data = a[:k]
	c1.Size = uint32(k)
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
	runAt := 0

	for _, val := range a {
		for runAt < len(runs) && runs[runAt+1] < val {
			runAt += 2
		}

		if runAt >= len(runs) || val < runs[runAt] {
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
		start, end := uint32(runs[i]), uint32(runs[i+1])
		for v := start; v <= end; v++ {
			if bmp.Contains(v) {
				bmp.Remove(v)
				c1.Size--
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
	arrAt := 0

	for i := 0; i < len(runs); i += 2 {
		start, end := uint32(runs[i]), uint32(runs[i+1])
		currStart := start

		for arrAt < len(arr) && uint32(arr[arrAt]) < currStart {
			arrAt++
		}

		for arrAt < len(arr) {
			val := arr[arrAt]
			val32 := uint32(val)
			if val32 > end {
				break
			}

			if currStart < val32 {
				out = append(out, uint16(currStart), uint16(val32-1))
				size += (val32 - 1) - currStart + 1
			}
			currStart = val32 + 1
			arrAt++
		}

		if currStart <= end {
			out = append(out, uint16(currStart), uint16(end))
			size += end - currStart + 1
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
		start, end := uint32(runs[i]), uint32(runs[i+1])
		currStart := start

		for v := start; v <= end; v++ {
			if bmp.Contains(v) {
				// Found element to exclude - add run before it
				if currStart < v {
					out = append(out, uint16(currStart), uint16(v-1))
					size += (v - 1) - currStart + 1
				}
				currStart = v + 1
			}
		}

		// Add remaining part of run
		if currStart <= end {
			out = append(out, uint16(currStart), uint16(end))
			size += end - currStart + 1
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
		s1, e1 := uint32(a[i]), uint32(a[i+1])

		// Find overlapping runs in second container
		currStart := s1
		for j < len(b) && uint32(b[j]) <= e1 {
			s2, e2 := uint32(b[j]), uint32(b[j+1])

			// Check for overlap
			if s2 <= e1 && e2 >= currStart {
				// Add segment before overlap
				if currStart < s2 {
					out = append(out, uint16(currStart), uint16(s2-1))
					size += (s2 - 1) - currStart + 1
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
			out = append(out, uint16(currStart), uint16(e1))
			size += e1 - currStart + 1
		}

		i += 2
	}

	c1.Data = append(c1.Data[:0], out...)
	c1.Size = size
	rb.scratch = out
	return size > 0
}

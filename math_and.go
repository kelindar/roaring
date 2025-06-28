// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

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
			rb.scratch = append(rb.scratch, uint16(i))
		case !rb.ctrAnd(c1, &other.containers[idx]):
			rb.scratch = append(rb.scratch, uint16(i))
		}
	}

	// Batch remove empty containers (in reverse order to maintain indices)
	for i := len(rb.scratch) - 1; i >= 0; i-- {
		rb.ctrDel(int(rb.scratch[i]))
	}
}

// and performs efficient AND between two containers
func (rb *Bitmap) ctrAnd(c1, c2 *container) bool {
	c1.fork()
	switch c1.Type {
	case typeArray:
		switch c2.Type {
		case typeArray:
			return rb.arrAndArr(c1, c2)
		case typeBitmap:
			return rb.arrAndBmp(c1, c2)
		case typeRun:
			return rb.arrAndRun(c1, c2)
		}
	case typeBitmap:
		switch c2.Type {
		case typeArray:
			return rb.bmpAndArr(c1, c2)
		case typeBitmap:
			return rb.bmpAndBmp(c1, c2)
		case typeRun:
			return rb.bmpAndRun(c1, c2)
		}
	case typeRun:
		switch c2.Type {
		case typeArray:
			return rb.runAndArr(c1, c2)
		case typeBitmap:
			return rb.runAndBmp(c1, c2)
		case typeRun:
			return rb.runAndRun(c1, c2)
		}
	}
	return false
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
	return c1.Size > 0
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
	return c1.Size > 0
}

// arrAndRun performs AND between array and run containers
func (rb *Bitmap) arrAndRun(c1, c2 *container) bool {
	a, b := c1.Data, c2.Data
	out := a[:0]
	i, j := 0, 0

	for i < len(a) && j < len(b) {
		val := a[i]
		start, end := b[j], b[j+1]
		switch {
		case val < start:
			i++
		case val > end:
			j += 2
		default: // val >= start && val <= end
			out = append(out, val)
			i++
		}
	}

	c1.Data = out
	c1.Size = uint32(len(out))
	return c1.Size > 0
}

// bmpAndArr performs AND between bitmap and array containers
func (rb *Bitmap) bmpAndArr(c1, c2 *container) bool {
	a, b := c1.bmp(), c2.Data
	out := rb.scratch[:0]

	for _, val := range b {
		if a.Contains(uint32(val)) {
			out = append(out, val)
		}
	}

	c1.Data = append(c1.Data[:0], out...)
	c1.Size = uint32(len(c1.Data))
	c1.Type = typeArray
	return c1.Size > 0
}

// bmpAndBmp performs AND between two bitmap containers
func (rb *Bitmap) bmpAndBmp(c1, c2 *container) bool {
	a, b := c1.bmp(), c2.bmp()
	if a == nil || b == nil {
		return false
	}

	a.And(b)
	c1.Size = uint32(a.Count())
	return c1.Size > 0
}

// bmpAndRun performs AND between bitmap and run containers
func (rb *Bitmap) bmpAndRun(c1, c2 *container) bool {
	a, b := c1.bmp(), c2.Data
	n := len(b) / 2
	if n == 0 {
		c1.Size = 0
		return false
	}

	count, run := 0, 0
	a.Filter(func(x uint32) bool {
		for run < n && x > uint32(b[run*2+1]) {
			run++
		}

		if run < n && x >= uint32(b[run*2]) && x <= uint32(b[run*2+1]) {
			count++
			return true
		}
		return false
	})

	c1.Size = uint32(count)
	return c1.Size > 0
}

// runAndArr performs AND between run and array containers
func (rb *Bitmap) runAndArr(c1, c2 *container) bool {
	a, b := c1.Data, c2.Data
	out := rb.scratch[:0]
	i, j := 0, 0

	for i < len(a) && j < len(b) {
		start, end := a[i], a[i+1]
		for j < len(b) && b[j] < start {
			j++
		}

		for j < len(b) && b[j] <= end {
			out = append(out, b[j])
			j++
		}
		i += 2
	}

	c1.Data = append(c1.Data[:0], out...)
	c1.Size = uint32(len(out))
	c1.Type = typeArray
	return c1.Size > 0
}

// runAndRun performs AND between two run containers
func (rb *Bitmap) runAndRun(c1, c2 *container) bool {
	a, b := c1.Data, c2.Data
	out := rb.scratch[:0]
	i, j := 0, 0
	size := uint32(0)

	for i < len(a) && j < len(b) {
		s1, e1 := uint32(a[i]), uint32(a[i+1])
		s2, e2 := uint32(b[j]), uint32(b[j+1])

		is, ie := s1, e1
		if s2 > is {
			is = s2
		}
		if e2 < ie {
			ie = e2
		}

		if is <= ie {
			out = append(out, uint16(is), uint16(ie))
			size += ie - is + 1
		}

		switch {
		case e1 < e2:
			i += 2
		case e2 < e1:
			j += 2
		default:
			i += 2
			j += 2
		}
	}

	c1.Data = append(c1.Data[:0], out...)
	c1.Size = size
	rb.scratch = out
	return size > 0
}

// runAndBmp performs AND between run and bitmap containers
func (rb *Bitmap) runAndBmp(c1, c2 *container) bool {
	a, b := c1.bmp(), c2.bmp()
	if a == nil || b == nil {
		return false
	}

	a.And(b)

	c1.Size = uint32(a.Count())
	return c1.Size > 0
}

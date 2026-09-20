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
	if len(rb.containers) == 1 && len(other.containers) == 1 {
		if rb.index[0] != other.index[0] {
			rb.Clear()
			return
		}
		c1, c2 := &rb.containers[0], &other.containers[0]
		var keep bool
		switch {
		case c1.Type == typeArray && c2.Type == typeArray:
			c1.fork()
			keep = rb.arrAndArr(c1, c2)
		default:
			keep = rb.ctrAnd(c1, c2)
		}
		if !keep {
			rb.Clear()
		}
		return
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
			continue
		case !rb.ctrAnd(c1, &other.containers[idx]):
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
	if len(rb.index) >= branchlessAt {
		for i < len(a) && j < len(b) {
			av, bv := uint32(a[i]), uint32(b[j])
			less, greater := int((av-bv)>>31), int((bv-av)>>31)
			a[k] = uint16(av)
			k += 1 - less - greater
			i += 1 - greater
			j += 1 - less
		}
	} else {
		if len(a) > 0 && len(b) > 0 {
			left, right := a, b
			av, bv := left[0], right[0]
		main:
			for {
				if bv < av {
					for {
						if len(right) <= 1 {
							break main
						}
						right = right[1:]
						bv = right[0]
						if bv >= av {
							break
						}
					}
				}
				if av < bv {
					for {
						if len(left) <= 1 {
							break main
						}
						left = left[1:]
						av = left[0]
						if av >= bv {
							break
						}
					}
				} else {
					a[k] = av
					k++
					if len(left) <= 1 {
						break
					}
					left = left[1:]
					av = left[0]
					if len(right) <= 1 {
						break
					}
					right = right[1:]
					bv = right[0]
				}
			}
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

		is, ie := max(s1, s2), min(e1, e2)

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
	return size > 0
}

// runAndBmp performs AND between run and bitmap containers
func (rb *Bitmap) runAndBmp(c1, c2 *container) bool {
	c1.runToBmp()
	return rb.bmpAndBmp(c1, c2)
}

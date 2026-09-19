// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

package roaring

// xor performs XOR with a single bitmap efficiently
func (rb *Bitmap) xor(other *Bitmap) {
	switch {
	case rb == other:
		rb.Clear()
		return
	case other == nil || len(other.containers) == 0:
		return // No change needed
	case len(rb.containers) == 0:
		// Copy all containers from other since A XOR B = B when A is empty
		rb.containers = make([]container, len(other.containers))
		rb.index = make([]uint16, len(other.index))
		for i := range other.containers {
			other.containers[i].Shared = true
		}
		copy(rb.containers, other.containers)
		copy(rb.index, other.index)
		return
	}

	// Compact matching keys in place before allocating a merged index.
	i, j, write := 0, 0, 0
	for i < len(rb.index) && j < len(other.index) && rb.index[i] == other.index[j] {
		if rb.ctrXor(&rb.containers[i], &other.containers[j]) {
			rb.containers[write] = rb.containers[i]
			rb.index[write] = rb.index[i]
			write++
		}
		i++
		j++
	}
	if j == len(other.index) {
		copy(rb.index[write:], rb.index[i:])
		write += copy(rb.containers[write:], rb.containers[i:])
		clearContainerTail(rb.containers, write)
		rb.containers = rb.containers[:write]
		rb.index = rb.index[:write]
		return
	}
	newContainers := make([]container, 0, len(rb.containers)+len(other.containers))
	newIndex := make([]uint16, 0, len(rb.index)+len(other.index))
	newContainers = append(newContainers, rb.containers[:write]...)
	newIndex = append(newIndex, rb.index[:write]...)

	for i < len(rb.containers) && j < len(other.containers) {
		hi1, hi2 := rb.index[i], other.index[j]
		switch {
		case hi1 < hi2:
			// Only in left bitmap - keep as is
			newContainers = append(newContainers, rb.containers[i])
			newIndex = append(newIndex, hi1)
			i++
		case hi1 > hi2:
			// Only in right bitmap - copy it
			other.containers[j].Shared = true
			newContainers = append(newContainers, other.containers[j])
			newIndex = append(newIndex, hi2)
			j++
		default:
			// In both bitmaps - XOR them
			c1 := &rb.containers[i]
			c2 := &other.containers[j]
			if rb.ctrXor(c1, c2) {
				// Only add if result is non-empty
				newContainers = append(newContainers, *c1)
				newIndex = append(newIndex, hi1)
			}
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

// ctrXor performs efficient XOR between two containers
func (rb *Bitmap) ctrXor(c1, c2 *container) bool {
	c1.fork()
	switch c1.Type {
	case typeArray:
		switch c2.Type {
		case typeArray:
			return rb.arrXorArr(c1, c2)
		case typeBitmap:
			return rb.arrXorBmp(c1, c2)
		case typeRun:
			return rb.arrXorRun(c1, c2)
		}
	case typeBitmap:
		switch c2.Type {
		case typeArray:
			return rb.bmpXorArr(c1, c2)
		case typeBitmap:
			return rb.bmpXorBmp(c1, c2)
		case typeRun:
			return rb.bmpXorRun(c1, c2)
		}
	case typeRun:
		switch c2.Type {
		case typeArray:
			return rb.runXorArr(c1, c2)
		case typeBitmap:
			return rb.runXorBmp(c1, c2)
		case typeRun:
			return rb.runXorRun(c1, c2)
		}
	}
	return false
}

// arrXorArr performs XOR between two array containers
func (rb *Bitmap) arrXorArr(c1, c2 *container) bool {
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
			out = out[:len(out)-1+less+greater]
			i += 1 - greater
			j += 1 - less
		}
	} else {
		for i < len(a) && j < len(b) {
			av, bv := a[i], b[j]
			switch {
			case av == bv:
				// Same element in both - exclude from XOR
				i++
				j++
			case av < bv:
				// Only in first array
				out = append(out, av)
				i++
			default: // av > bv
				// Only in second array
				out = append(out, bv)
				j++
			}
		}
	}

	// Add remaining elements from first array
	for i < len(a) {
		out = append(out, a[i])
		i++
	}
	// Add remaining elements from second array
	for j < len(b) {
		out = append(out, b[j])
		j++
	}

	rb.scratch = c1.Data[:0]
	c1.Data = out
	c1.Size = uint32(len(c1.Data))
	return c1.Size > 0
}

// arrXorBmp performs XOR between array and bitmap containers
func (rb *Bitmap) arrXorBmp(c1, c2 *container) bool {
	// Convert to bitmap for efficient XOR
	c1.arrToBmp()
	return rb.bmpXorBmp(c1, c2)
}

// arrXorRun performs XOR between array and run containers
func (rb *Bitmap) arrXorRun(c1, c2 *container) bool {
	a := c1.Data
	runs := c2.Data
	out := rb.scratch[:0]
	arrAt := 0

	for i := 0; i < len(runs); i += 2 {
		start, end := uint32(runs[i]), uint32(runs[i+1])

		for arrAt < len(a) && uint32(a[arrAt]) < start {
			out = append(out, a[arrAt])
			arrAt++
		}

		curr := start
		for arrAt < len(a) {
			val := uint32(a[arrAt])
			if val > end {
				break
			}

			for curr < val {
				out = append(out, uint16(curr))
				curr++
			}
			curr = val + 1
			arrAt++
		}

		for curr <= end {
			out = append(out, uint16(curr))
			curr++
		}
	}

	for arrAt < len(a) {
		out = append(out, a[arrAt])
		arrAt++
	}

	c1.Data = append(c1.Data[:0], out...)
	c1.Size = uint32(len(c1.Data))
	c1.Type = typeArray
	rb.scratch = out
	return c1.Size > 0
}

// bmpXorArr performs XOR between bitmap and array containers
func (rb *Bitmap) bmpXorArr(c1, c2 *container) bool {
	bmp := c1.bmp()
	for _, val := range c2.Data {
		if bmp.Contains(uint32(val)) {
			bmp.Remove(uint32(val))
			c1.Size--
		} else {
			bmp.Set(uint32(val))
			c1.Size++
		}
	}
	return c1.Size > 0
}

// bmpXorBmp performs XOR between two bitmap containers
func (rb *Bitmap) bmpXorBmp(c1, c2 *container) bool {
	a, b := c1.bmp(), c2.bmp()
	if b == nil {
		return c1.Size > 0
	}

	a.Xor(b)
	c1.Size = uint32(a.Count())
	return c1.Size > 0
}

// bmpXorRun performs XOR between bitmap and run containers
func (rb *Bitmap) bmpXorRun(c1, c2 *container) bool {
	bmp := c1.bmp()
	runs := c2.Data

	for i := 0; i < len(runs); i += 2 {
		start, end := uint32(runs[i]), uint32(runs[i+1])
		for v := start; v <= end; v++ {
			if bmp.Contains(v) {
				bmp.Remove(v)
				c1.Size--
			} else {
				bmp.Set(v)
				c1.Size++
			}
		}
	}
	return c1.Size > 0
}

// runXorArr performs XOR between run and array containers
func (rb *Bitmap) runXorArr(c1, c2 *container) bool {
	// Convert to array for simpler XOR, then optimize
	c1.runToArray()
	result := rb.arrXorArr(c1, c2)
	c1.optimize()
	return result
}

// runXorBmp performs XOR between run and bitmap containers
func (rb *Bitmap) runXorBmp(c1, c2 *container) bool {
	// Convert run to bitmap and XOR
	c1.runToBmp()
	return rb.bmpXorBmp(c1, c2)
}

// runXorRun performs XOR between two run containers
func (rb *Bitmap) runXorRun(c1, c2 *container) bool {
	// For simplicity, convert both to arrays, XOR, then optimize
	c1.runToArray()

	// Create temporary array from second run container
	runs := c2.Data
	var tempArray []uint16
	for i := 0; i < len(runs); i += 2 {
		start, end := uint32(runs[i]), uint32(runs[i+1])
		for v := start; v <= end; v++ {
			tempArray = append(tempArray, uint16(v))
		}
	}

	temp := &container{
		Type: typeArray,
		Data: tempArray,
		Size: uint32(len(tempArray)),
	}

	result := rb.arrXorArr(c1, temp)
	c1.optimize()
	return result
}

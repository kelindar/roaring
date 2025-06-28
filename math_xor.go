// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

package roaring

// xor performs XOR with a single bitmap efficiently
func (rb *Bitmap) xor(other *Bitmap) {
	switch {
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

	// Merge containers from both bitmaps using XOR logic
	i, j := 0, 0
	var newContainers []container
	var newIndex []uint16

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
	i, j := 0, 0

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

	c1.Data = append(c1.Data[:0], out...)
	c1.Size = uint32(len(c1.Data))
	rb.scratch = out
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
	runs := c2.Data
	out := rb.scratch[:0]

	for _, val := range c1.Data {
		// Check if value is in any run
		inRun := false
		for i := 0; i < len(runs); i += 2 {
			if uint32(val) >= uint32(runs[i]) && uint32(val) <= uint32(runs[i+1]) {
				inRun = true
				break
			}
		}
		if !inRun {
			out = append(out, val)
		}
	}

	// Add values from runs that are not in array
	for i := 0; i < len(runs); i += 2 {
		start, end := uint32(runs[i]), uint32(runs[i+1])
		for v := start; v <= end; v++ {
			// Check if value is in array
			_, found := find16(c1.Data, uint16(v))
			if !found {
				out = append(out, uint16(v))
			}
		}
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

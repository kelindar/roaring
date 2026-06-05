// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

package roaring

import "math/bits"

// Range calls the given function for each value in the bitmap
func (rb *Bitmap) Range(fn func(x uint32) bool) {
	for i := range rb.containers {
		c := &rb.containers[i]
		base := uint32(rb.index[i]) << 16

		switch c.Type {
		case typeArray:
			data := c.Data
			for j := 0; j < len(data); j++ {
				if !fn(base | uint32(data[j])) {
					return
				}
			}

		case typeBitmap:
			if !c.bmpRange(func(value uint32) bool {
				return fn(base | value)
			}) {
				return
			}

		case typeRun:
			numRuns := len(c.Data) / 2
			for i := 0; i < numRuns; i++ {
				start, end := uint32(c.Data[i*2]), uint32(c.Data[i*2+1])
				for curr := start; curr <= end; curr++ {
					if !fn(base | curr) {
						return
					}
				}
			}
		}
	}
}

// Filter iterates over the bitmap elements and calls a predicate provided for each
// containing element. If the predicate returns false, the bitmap at the element's
// position is set to zero.
func (rb *Bitmap) Filter(f func(x uint32) bool) {
	rb.scratch = rb.scratch[:0]

	for i := range rb.containers {
		c := &rb.containers[i]
		base := uint32(rb.index[i]) << 16

		switch c.Type {
		case typeArray:
			c.fork()
			data := c.Data
			out := data[:0]
			for _, value := range data {
				if f(base | uint32(value)) {
					out = append(out, value)
				}
			}
			c.Data = out
			c.Size = uint32(len(out))

		case typeBitmap:
			c.fork()
			count := uint32(0)
			bmp := c.bmp()
			bmp.Filter(func(value uint32) bool {
				keep := f(base | value)
				if keep {
					count++
				}
				return keep
			})
			c.Size = count
			c.optimize()

		case typeRun:
			c.fork()
			runs := c.Data
			out := runs[:0]
			size := uint32(0)

			for i := 0; i < len(runs); i += 2 {
				start, end := uint32(runs[i]), uint32(runs[i+1])
				var keepStart, keepEnd uint32
				inRun := false

				for curr := start; curr <= end; curr++ {
					if f(base | curr) {
						if !inRun {
							keepStart = curr
							inRun = true
						}
						keepEnd = curr
						continue
					}

					if inRun {
						out = append(out, uint16(keepStart), uint16(keepEnd))
						size += keepEnd - keepStart + 1
						inRun = false
					}
				}

				if inRun {
					out = append(out, uint16(keepStart), uint16(keepEnd))
					size += keepEnd - keepStart + 1
				}
			}
			c.Data = out
			c.Size = size
			c.optimize()
		}

		if c.isEmpty() {
			rb.scratch = append(rb.scratch, uint16(i))
		}
	}

	for i := len(rb.scratch) - 1; i >= 0; i-- {
		rb.ctrDel(int(rb.scratch[i]))
	}
}

// Iterate iterates over all of the bits set to one in this bitmap.
func (c *container) bmpRange(fn func(x uint32) bool) bool {
	dst := c.bmp()
	for blkAt := 0; blkAt < len(dst); blkAt++ {
		blk := dst[blkAt]
		offset := uint32(blkAt << 6)
		for blk != 0 {
			bit := uint32(bits.TrailingZeros64(blk))
			if !fn(offset + bit) {
				return false
			}
			blk &= blk - 1
		}
	}
	return true
}

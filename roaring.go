package roaring

import "unsafe"

// Bitmap represents a roaring bitmap for uint32 values
type Bitmap struct {
	containers []container // Containers in sorted order by key
	index      []uint16    // Container keys for cache-efficient searching
	scratch    []uint32
}

// New creates a new empty roaring bitmap
func New() *Bitmap {
	return &Bitmap{}
}

// Set sets the bit x in the bitmap and grows it if necessary.
func (rb *Bitmap) Set(x uint32) {
	hi, lo := uint16(x>>16), uint16(x&0xFFFF)
	idx, exists := rb.ctrFind(hi)
	if !exists {
		rb.ctrAdd(hi, idx, &container{
			Type: typeArray,
			Size: 0,
			Data: make([]uint16, 0, 64),
		})
	}
	rb.containers[idx].set(lo)
}

// Remove removes the bit x from the bitmap
func (rb *Bitmap) Remove(x uint32) {
	hi, lo := uint16(x>>16), uint16(x&0xFFFF)
	idx, exists := rb.ctrFind(hi)
	if !exists || !rb.containers[idx].remove(lo) {
		return
	}

	if rb.containers[idx].isEmpty() {
		rb.ctrDel(idx)
	}
}

// Contains checks whether a value is contained in the bitmap
func (rb *Bitmap) Contains(x uint32) bool {
	hi, lo := uint16(x>>16), uint16(x&0xFFFF)
	idx, exists := rb.ctrFind(hi)
	if !exists {
		return false
	}

	return rb.containers[idx].contains(lo)
}

// Count returns the total number of bits set to 1 in the bitmap
func (rb *Bitmap) Count() int {
	count := 0
	for i := range rb.containers {
		count += int(rb.containers[i].Size)
	}
	return count
}

// Clear clears the bitmap
func (rb *Bitmap) Clear() {
	rb.containers = rb.containers[:0]
	rb.index = rb.index[:0]
}

// Optimize optimizes all containers to use the most efficient representation
func (rb *Bitmap) Optimize() {
	for i := range rb.containers {
		rb.containers[i].optimize()
	}
}

// Clone clones the bitmap
func (rb *Bitmap) Clone(into *Bitmap) *Bitmap {
	if into == nil {
		into = &Bitmap{}
	}

	// Clone containers
	into.containers = make([]container, len(rb.containers))

	for i := range rb.containers {
		// Mark original as shared and copy with shared data
		rb.containers[i].Shared = true
		into.containers[i] = container{
			Type:   rb.containers[i].Type,
			Call:   rb.containers[i].Call,
			Size:   rb.containers[i].Size,
			Data:   rb.containers[i].Data, // Share the same underlying slice
			Shared: true,
		}
	}

	// Clone index
	into.index = make([]uint16, len(rb.index))
	copy(into.index, rb.index)

	return into
}

// ---------------------------------------- Container ----------------------------------------

// ctrFind finds the container for the given high bits (read-only, no creation)
// Returns (index, found) where index is the insertion point if not found
func (rb *Bitmap) ctrFind(hi uint16) (int, bool) {
	const blockSize = 64

	index, n := rb.index, len(rb.index)
	switch {
	case n == 0:
		return 0, false
	case hi < index[0]:
		return 0, false
	case hi > index[n-1]:
		return n, false
	case hi == index[0]:
		return 0, true
	case hi == index[n-1]:
		return n - 1, true
	case n <= 16:
		// Simple linear search for small arrays
		for i, key := range index {
			switch {
			case key == hi:
				return i, true
			case key > hi:
				return i, false
			}
		}
		return n, false
	default:
		// Binary search for the correct block
		numBlocks := (n + blockSize - 1) / blockSize
		left, right := 0, numBlocks-1
		for left <= right {
			mid := left + (right-left)>>1
			blockStart := mid * blockSize
			blockEnd := blockStart + blockSize
			if blockEnd > n {
				blockEnd = n
			}

			switch {
			case hi < index[blockStart]:
				right = mid - 1
			case hi > index[blockEnd-1]:
				left = mid + 1
			default:
				// Use optimized block search
				var result int64 = -1
				_find16(unsafe.Pointer(&index[blockStart]), hi, unsafe.Pointer(&result), uint64(blockEnd-blockStart))
				if result >= 0 {
					return int(result), true
				}

				return left * blockSize, false

				//return searchBlock(index, blockStart, blockEnd, hi)
			}
		}

		return left * blockSize, false
	}
}

// searchBlock performs an optimized linear search within a block
// Returns (index, found) for the key within the block range
func searchBlock(keys []uint16, start, end int, target uint16) (int, bool) {
	for i := start; i < end; {
		remaining := end - i
		switch {
		case remaining >= 4:
			if keys[i] >= target {
				return i, keys[i] == target
			}
			if keys[i+1] >= target {
				return i + 1, keys[i+1] == target
			}
			if keys[i+2] >= target {
				return i + 2, keys[i+2] == target
			}
			if keys[i+3] >= target {
				return i + 3, keys[i+3] == target
			}
			i += 4
		case remaining >= 2:
			if keys[i] >= target {
				return i, keys[i] == target
			}
			if keys[i+1] >= target {
				return i + 1, keys[i+1] == target
			}
			i += 2
		default:
			if keys[i] >= target {
				return i, keys[i] == target
			}
			i++
		}
	}
	return end, false
}

// ctrAdd inserts a container at the given position
func (rb *Bitmap) ctrAdd(hi uint16, pos int, c *container) {
	// Insert new container at position to maintain order
	rb.containers = append(rb.containers, container{})
	if pos < len(rb.containers)-1 {
		copy(rb.containers[pos+1:], rb.containers[pos:len(rb.containers)-1])
	}
	rb.containers[pos] = *c

	// Keep index in sync
	rb.index = append(rb.index, 0)
	if pos < len(rb.index)-1 {
		copy(rb.index[pos+1:], rb.index[pos:len(rb.index)-1])
	}
	rb.index[pos] = hi
}

// ctrDel removes the container at the given position
func (rb *Bitmap) ctrDel(pos int) {
	if pos < 0 || pos >= len(rb.containers) {
		return
	}

	// Remove container by shifting slice
	copy(rb.containers[pos:], rb.containers[pos+1:])
	rb.containers = rb.containers[:len(rb.containers)-1]

	// Keep index in sync
	copy(rb.index[pos:], rb.index[pos+1:])
	rb.index = rb.index[:len(rb.index)-1]
}

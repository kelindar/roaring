package roaring

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
	case n <= 8:
		for i, key := range index {
			switch {
			case key == hi:
				return i, true
			case key > hi:
				return i, false
			}
		}
		return n, false

	// Galloping search for larger arrays
	default:
		pos := 1
		for pos < n && index[pos] < hi {
			pos <<= 1 // Double the position
		}

		// Binary search in the found range
		left := pos >> 1
		right := pos
		if right >= n {
			right = n - 1
		}

		for left <= right {
			mid := left + (right-left)>>1
			switch {
			case index[mid] == hi:
				return mid, true
			case index[mid] < hi:
				left = mid + 1
			default:
				right = mid - 1
			}
		}

		return left, false
	}
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

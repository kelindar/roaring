package roaring

// Bitmap represents a roaring bitmap for uint32 values
type Bitmap struct {
	indices    []uint16    // Sorted high 16-bit keys for containers
	containers []container // Containers corresponding to indices
	scratch    []uint32
}

// New creates a new empty roaring bitmap
func New() *Bitmap {
	return &Bitmap{}
}

// Set sets the bit x in the bitmap and grows it if necessary.
func (rb *Bitmap) Set(x uint32) {
	hi, lo := uint16(x>>16), uint16(x&0xFFFF)
	c, exists := rb.ctrFind(hi)
	if !exists {
		newContainer := &container{
			Type: typeArray,
			Size: 0,
			Data: make([]uint16, 0, 64),
		}
		rb.ctrAdd(hi, newContainer)
		// Get the stored container (not the local pointer)
		c, _ = rb.ctrFind(hi)
	}

	c.set(lo)
}

// Remove removes the bit x from the bitmap
func (rb *Bitmap) Remove(x uint32) {
	hi, lo := uint16(x>>16), uint16(x&0xFFFF)
	c, exists := rb.ctrFind(hi)
	if !exists || !c.remove(lo) {
		return
	}

	if c.isEmpty() {
		rb.ctrDel(hi)
	}
}

// Contains checks whether a value is contained in the bitmap
func (rb *Bitmap) Contains(x uint32) bool {
	hi, lo := uint16(x>>16), uint16(x&0xFFFF)
	c, exists := rb.ctrFind(hi)
	if !exists {
		return false
	}

	return c.contains(lo)
}

// Count returns the total number of bits set to 1 in the bitmap
func (rb *Bitmap) Count() int {
	count := 0
	rb.iterateContainers(func(base uint32, c *container) {
		count += int(c.Size)
	})
	return count
}

// Clear clears the bitmap
func (rb *Bitmap) Clear() {
	rb.indices = rb.indices[:0]
	rb.containers = rb.containers[:0]
}

// Optimize optimizes all containers to use the most efficient representation
func (rb *Bitmap) Optimize() {
	rb.iterateContainers(func(base uint32, c *container) {
		c.optimize()
	})
}

// Clone clones the bitmap
func (rb *Bitmap) Clone(into *Bitmap) *Bitmap {
	if into == nil {
		into = &Bitmap{}
	}

	// Clone containers directly
	into.indices = make([]uint16, len(rb.indices))
	into.containers = make([]container, len(rb.containers))

	copy(into.indices, rb.indices)
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

	return into
}

// ---------------------------------------- Container ----------------------------------------

// ctrFind finds the container for the given high bits using binary search
func (rb *Bitmap) ctrFind(hi uint16) (*container, bool) {
	pos, found := rb.findContainer(hi)
	if !found {
		return nil, false
	}
	return &rb.containers[pos], true
}

// findContainer performs binary search to find container position
func (rb *Bitmap) findContainer(hi uint16) (int, bool) {
	if len(rb.indices) == 0 {
		return 0, false
	}

	// Quick bounds check
	if hi < rb.indices[0] {
		return 0, false
	}
	if hi > rb.indices[len(rb.indices)-1] {
		return len(rb.indices), false
	}

	// Binary search
	left, right := 0, len(rb.indices)
	for left < right {
		mid := left + (right-left)/2
		switch {
		case rb.indices[mid] < hi:
			left = mid + 1
		case rb.indices[mid] > hi:
			right = mid
		default:
			return mid, true
		}
	}

	return left, false
}

// ctrAdd sets a container at the given high bits
func (rb *Bitmap) ctrAdd(hi uint16, c *container) {
	pos, found := rb.findContainer(hi)
	if found {
		rb.containers[pos] = *c
		return
	}

	// Insert new container at correct position to maintain sorted order
	rb.indices = append(rb.indices, 0)
	rb.containers = append(rb.containers, container{})

	// Shift elements to make room
	if pos < len(rb.indices)-1 {
		copy(rb.indices[pos+1:], rb.indices[pos:len(rb.indices)-1])
		copy(rb.containers[pos+1:], rb.containers[pos:len(rb.containers)-1])
	}

	// Insert new elements
	rb.indices[pos] = hi
	rb.containers[pos] = *c
}

// ctrDel removes the container at the given high bits
func (rb *Bitmap) ctrDel(hi uint16) {
	pos, found := rb.findContainer(hi)
	if !found {
		return
	}

	// Remove element by shifting slices
	copy(rb.indices[pos:], rb.indices[pos+1:])
	copy(rb.containers[pos:], rb.containers[pos+1:])

	// Shrink slices
	rb.indices = rb.indices[:len(rb.indices)-1]
	rb.containers = rb.containers[:len(rb.containers)-1]
}

// iterateContainers iterates over all containers in the bitmap
func (rb *Bitmap) iterateContainers(fn func(base uint32, c *container)) {
	for i, hi := range rb.indices {
		base := uint32(hi) << 16
		fn(base, &rb.containers[i])
	}
}

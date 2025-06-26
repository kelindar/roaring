package roaring

// Bitmap represents a roaring bitmap for uint32 values
type Bitmap struct {
	containers []container // Containers in sorted order by key
	scratch    []uint32
}

// New creates a new empty roaring bitmap
func New() *Bitmap {
	return &Bitmap{}
}

// Set sets the bit x in the bitmap and grows it if necessary.
func (rb *Bitmap) Set(x uint32) {
	hi, lo := uint16(x>>16), uint16(x&0xFFFF)
	c := rb.ctrLoad(hi)
	c.set(lo)
}

// ctrFind finds the container for the given high bits (read-only, no creation)
// Returns (index, found) where index is the insertion point if not found
func (rb *Bitmap) ctrFind(hi uint16) (int, bool) {
	containers := rb.containers
	n := len(containers)

	// Quick bounds check for early exit
	switch {
	case n == 0:
		return 0, false
	case hi < containers[0].Key:
		return 0, false
	case hi > containers[n-1].Key:
		return n, false
	case hi == containers[0].Key:
		return 0, true
	case hi == containers[n-1].Key:
		return n - 1, true
	}

	// Linear search for small arrays (cache-friendly, fewer branches)
	if n <= 8 {
		for i, c := range containers {
			switch {
			case c.Key == hi:
				return i, true
			case c.Key > hi:
				return i, false
			}
		}
		return n, false
	}

	// Galloping search for larger arrays
	// First, find the range using exponential search
	pos := 1
	for pos < n && containers[pos].Key < hi {
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
		case containers[mid].Key == hi:
			return mid, true
		case containers[mid].Key < hi:
			left = mid + 1
		default:
			right = mid - 1
		}
	}

	return left, false
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
			Key:    rb.containers[i].Key,
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

// ctrLoad finds the container for the given high bits, creating it if it doesn't exist
func (rb *Bitmap) ctrLoad(hi uint16) *container {
	containers := rb.containers
	n := len(containers)

	// Quick bounds check for early exit
	switch {
	case n == 0:
		// Create first container
		rb.ctrAdd(hi, 0, &container{
			Key:  hi,
			Type: typeArray,
			Size: 0,
			Data: make([]uint16, 0, 64),
		})
		return &rb.containers[0]
	case hi < containers[0].Key:
		// Insert at beginning
		rb.ctrAdd(hi, 0, &container{
			Key:  hi,
			Type: typeArray,
			Size: 0,
			Data: make([]uint16, 0, 64),
		})
		return &rb.containers[0]
	case hi > containers[n-1].Key:
		// Insert at end
		rb.ctrAdd(hi, n, &container{
			Key:  hi,
			Type: typeArray,
			Size: 0,
			Data: make([]uint16, 0, 64),
		})
		return &rb.containers[n]
	case hi == containers[0].Key:
		return &containers[0]
	case hi == containers[n-1].Key:
		return &containers[n-1]
	}

	// Linear search for small arrays (cache-friendly, fewer branches)
	if n <= 8 {
		for i, c := range containers {
			switch {
			case c.Key == hi:
				return &rb.containers[i]
			case c.Key > hi:
				// Insert at position i
				rb.ctrAdd(hi, i, &container{
					Key:  hi,
					Type: typeArray,
					Size: 0,
					Data: make([]uint16, 0, 64),
				})
				return &rb.containers[i]
			}
		}
		// Insert at end
		rb.ctrAdd(hi, n, &container{
			Key:  hi,
			Type: typeArray,
			Size: 0,
			Data: make([]uint16, 0, 64),
		})
		return &rb.containers[n]
	}

	// Galloping search for larger arrays
	// First, find the range using exponential search
	pos := 1
	for pos < n && containers[pos].Key < hi {
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
		case containers[mid].Key == hi:
			return &rb.containers[mid]
		case containers[mid].Key < hi:
			left = mid + 1
		default:
			right = mid - 1
		}
	}

	// Not found, create at insertion point (left)
	rb.ctrAdd(hi, left, &container{
		Key:  hi,
		Type: typeArray,
		Size: 0,
		Data: make([]uint16, 0, 64),
	})
	return &rb.containers[left]
}

// ctrAdd inserts a container at the given position
func (rb *Bitmap) ctrAdd(hi uint16, pos int, c *container) {
	// Insert new container at position to maintain order
	rb.containers = append(rb.containers, container{})
	if pos < len(rb.containers)-1 {
		copy(rb.containers[pos+1:], rb.containers[pos:len(rb.containers)-1])
	}
	rb.containers[pos] = *c
}

// ctrDel removes the container at the given position
func (rb *Bitmap) ctrDel(pos int) {
	if pos < 0 || pos >= len(rb.containers) {
		return
	}

	// Remove container by shifting slice
	copy(rb.containers[pos:], rb.containers[pos+1:])
	rb.containers = rb.containers[:len(rb.containers)-1]
}

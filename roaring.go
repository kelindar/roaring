package roaring

import "github.com/kelindar/bitmap"

// Bitmap represents a roaring bitmap for uint32 values
type Bitmap struct {
	index      bitmap.Bitmap // Bitmap tracking which container keys exist
	containers []container   // Containers in order of their keys
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
	rb.index.Clear()
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

	// Clone index bitmap and containers
	into.index = make(bitmap.Bitmap, len(rb.index))
	copy(into.index, rb.index)
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

	return into
}

// ---------------------------------------- Container ----------------------------------------

// ctrFind finds the container for the given high bits using the bitmap index
func (rb *Bitmap) ctrFind(hi uint16) (*container, bool) {
	if !rb.index.Contains(uint32(hi)) {
		return nil, false
	}

	// Find position in containers slice by counting set bits before hi
	pos := rb.index.CountTo(uint32(hi))
	if pos >= len(rb.containers) {
		return nil, false
	}

	return &rb.containers[pos], true
}

// ctrAdd sets a container at the given high bits
func (rb *Bitmap) ctrAdd(hi uint16, c *container) {
	key := uint32(hi)

	if rb.index.Contains(key) {
		// Update existing container
		pos := rb.index.CountTo(key)
		rb.containers[pos] = *c
		return
	}

	// Insert new container at correct position to maintain order
	pos := rb.index.CountTo(key)
	rb.index.Set(key)

	// Insert new container at position
	rb.containers = append(rb.containers, container{})
	if pos < len(rb.containers)-1 {
		copy(rb.containers[pos+1:], rb.containers[pos:len(rb.containers)-1])
	}
	rb.containers[pos] = *c
}

// ctrDel removes the container at the given high bits
func (rb *Bitmap) ctrDel(hi uint16) {
	key := uint32(hi)

	if !rb.index.Contains(key) {
		return
	}

	// Find position and remove
	pos := rb.index.CountTo(key)
	rb.index.Remove(key)

	// Remove container by shifting slice
	copy(rb.containers[pos:], rb.containers[pos+1:])
	rb.containers = rb.containers[:len(rb.containers)-1]
}

// iterateContainers iterates over all containers in the bitmap
func (rb *Bitmap) iterateContainers(fn func(base uint32, c *container)) {
	pos := 0
	rb.index.Range(func(key uint32) {
		base := key << 16
		fn(base, &rb.containers[pos])
		pos++
	})
}

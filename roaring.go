package roaring

import (
	"io"
	"sort"
)

// containerPair represents a (key, container) pair in sorted order
type containerPair struct {
	key       uint16
	container *container
}

// Bitmap represents a roaring bitmap for uint32 values
type Bitmap struct {
	containers []containerPair // sorted by key for binary search
	// Cache for last accessed container to avoid binary search
	lastIdx int
	lastHi  uint16
}

// New creates a new empty roaring bitmap
func New() *Bitmap {
	return &Bitmap{
		containers: make([]containerPair, 0, 8), // start small, grow as needed
		lastIdx:    -1,                          // invalid index initially
	}
}

// findContainer finds the container for the given high bits using binary search
// Returns (index, found) where index is either the found position or insertion point
func (rb *Bitmap) findContainer(hi uint16) (int, bool) {
	// Check cache first
	if rb.lastIdx >= 0 && rb.lastIdx < len(rb.containers) && rb.containers[rb.lastIdx].key == hi {
		return rb.lastIdx, true
	}

	// Binary search for the container
	idx := sort.Search(len(rb.containers), func(i int) bool {
		return rb.containers[i].key >= hi
	})

	found := idx < len(rb.containers) && rb.containers[idx].key == hi
	if found {
		rb.lastIdx = idx
		rb.lastHi = hi
	}
	return idx, found
}

// getContainer returns the container for the given high bits
func (rb *Bitmap) getContainer(hi uint16) (*container, bool) {
	idx, found := rb.findContainer(hi)
	if found {
		return rb.containers[idx].container, true
	}
	return nil, false
}

// setContainer sets a container at the given high bits
func (rb *Bitmap) setContainer(hi uint16, c *container) {
	idx, found := rb.findContainer(hi)
	if found {
		// Replace existing container
		rb.containers[idx].container = c
		rb.lastIdx = idx
		rb.lastHi = hi
	} else {
		// Insert new container at the correct position
		rb.containers = append(rb.containers, containerPair{})
		copy(rb.containers[idx+1:], rb.containers[idx:])
		rb.containers[idx] = containerPair{key: hi, container: c}
		rb.lastIdx = idx
		rb.lastHi = hi
	}
}

// removeContainer removes the container at the given high bits
func (rb *Bitmap) removeContainer(hi uint16) {
	idx, found := rb.findContainer(hi)
	if found {
		// Remove container by shifting slice
		copy(rb.containers[idx:], rb.containers[idx+1:])
		rb.containers = rb.containers[:len(rb.containers)-1]

		// Invalidate cache
		rb.lastIdx = -1
	}
}

// Set sets the bit x in the bitmap and grows it if necessary.
func (rb *Bitmap) Set(x uint32) {
	hi, lo := uint16(x>>16), uint16(x&0xFFFF)
	c, exists := rb.getContainer(hi)
	if !exists {
		c = &container{
			Type: typeArray,
			Size: 0,                     // Start with zero cardinality
			Data: make([]uint16, 0, 64), // Start empty with some capacity (64 uint16s = 128 bytes)
		}
		rb.setContainer(hi, c)
	}

	c.set(lo)
}

// Remove removes the bit x from the bitmap, but does not shrink it.
func (rb *Bitmap) Remove(x uint32) {
	hi, lo := uint16(x>>16), uint16(x&0xFFFF)
	c, exists := rb.getContainer(hi)
	if !exists || !c.remove(lo) {
		return
	}

	if c.isEmpty() {
		rb.removeContainer(hi)
	}
}

// Contains checks whether a value is contained in the bitmap or not.
func (rb *Bitmap) Contains(x uint32) bool {
	hi, lo := uint16(x>>16), uint16(x&0xFFFF)
	c, exists := rb.getContainer(hi)
	if !exists {
		return false
	}

	return c.contains(lo)
}

// Count returns the total number of bits set to 1 in the bitmap
func (rb *Bitmap) Count() int {
	count := 0
	for i := range rb.containers {
		count += rb.containers[i].container.cardinality()
	}
	return count
}

// Clear clears the bitmap and resizes it to zero.
func (rb *Bitmap) Clear() {
	rb.containers = rb.containers[:0] // reuse underlying array
	rb.lastIdx = -1
}

// Optimize optimizes all containers to use the most efficient representation.
// This can significantly reduce memory usage, especially after bulk operations.
func (rb *Bitmap) Optimize() {
	for i := range rb.containers {
		rb.containers[i].container.optimize()
	}
}

// Grow grows the bitmap size until we reach the desired bit.
func (rb *Bitmap) Grow(desiredBit uint32) {
	panic("not implemented")
}

// Clone clones the bitmap
func (rb *Bitmap) Clone(into *Bitmap) *Bitmap {
	panic("not implemented")
}

// And performs bitwise AND operation with other bitmap(s)
func (rb *Bitmap) And(other *Bitmap, extra ...*Bitmap) {
	panic("not implemented")
}

// AndNot performs bitwise AND NOT operation with other bitmap(s)
func (rb *Bitmap) AndNot(other *Bitmap, extra ...*Bitmap) {
	panic("not implemented")
}

// Or performs bitwise OR operation with other bitmap(s)
func (rb *Bitmap) Or(other *Bitmap, extra ...*Bitmap) {
	panic("not implemented")
}

// Xor performs bitwise XOR operation with other bitmap(s)
func (rb *Bitmap) Xor(other *Bitmap, extra ...*Bitmap) {
	panic("not implemented")
}

// Range calls the given function for each value in the bitmap
func (rb *Bitmap) Range(fn func(x uint32)) {
	panic("not implemented")
}

// Filter iterates over the bitmap elements and calls a predicate provided for each
// containing element. If the predicate returns false, the bitmap at the element's
// position is set to zero.
func (rb *Bitmap) Filter(f func(x uint32) bool) {
	panic("not implemented")
}

// ToBytes converts the bitmap to a byte slice
func (rb *Bitmap) ToBytes() []byte {
	panic("not implemented")
}

// WriteTo writes the bitmap to a writer
func (rb *Bitmap) WriteTo(w io.Writer) (int64, error) {
	panic("not implemented")
}

// ReadFrom reads the bitmap from a reader
func (rb *Bitmap) ReadFrom(r io.Reader) (int64, error) {
	panic("not implemented")
}

// FromBytes creates a roaring bitmap from a byte buffer
func FromBytes(buffer []byte) *Bitmap {
	panic("not implemented")
}

// ReadFrom reads a roaring bitmap from an io.Reader
func ReadFrom(r io.Reader) (*Bitmap, error) {
	panic("not implemented")
}

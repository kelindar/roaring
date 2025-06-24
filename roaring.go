package roaring

import "io"

// Bitmap represents a roaring bitmap for uint32 values
type Bitmap struct {
	containers map[uint16]*container
	// Cache for last accessed container to avoid map lookups
	lastHi    uint16
	lastContainer *container
	// Secondary cache for second-most-recent container
	prevHi    uint16
	prevContainer *container
	// Tertiary cache for third-most-recent container
	thirdHi   uint16
	thirdContainer *container
}

// New creates a new empty roaring bitmap
func New() *Bitmap {
	return &Bitmap{
		containers: make(map[uint16]*container),
	}
}

// getContainer returns the container for the given high bits, using multi-level cache for performance
func (rb *Bitmap) getContainer(hi uint16) (*container, bool) {
	// Check primary cache first
	if rb.lastContainer != nil && rb.lastHi == hi {
		return rb.lastContainer, true
	}
	
	// Check secondary cache
	if rb.prevContainer != nil && rb.prevHi == hi {
		// Promote to primary cache
		rb.thirdHi, rb.prevHi, rb.lastHi = rb.prevHi, rb.lastHi, hi
		rb.thirdContainer, rb.prevContainer, rb.lastContainer = rb.prevContainer, rb.lastContainer, rb.prevContainer
		return rb.lastContainer, true
	}
	
	// Check tertiary cache
	if rb.thirdContainer != nil && rb.thirdHi == hi {
		// Promote to primary cache
		rb.thirdHi, rb.prevHi, rb.lastHi = rb.prevHi, rb.lastHi, hi
		rb.thirdContainer, rb.prevContainer, rb.lastContainer = rb.prevContainer, rb.lastContainer, rb.thirdContainer
		return rb.lastContainer, true
	}
	
	// Fallback to map lookup
	c, exists := rb.containers[hi]
	if exists {
		// Update cache (shift down)
		rb.thirdHi, rb.prevHi, rb.lastHi = rb.prevHi, rb.lastHi, hi
		rb.thirdContainer, rb.prevContainer, rb.lastContainer = rb.prevContainer, rb.lastContainer, c
	}
	return c, exists
}

// setContainer sets a container and updates the cache
func (rb *Bitmap) setContainer(hi uint16, c *container) {
	rb.containers[hi] = c
	// Update cache (shift down)
	rb.thirdHi, rb.prevHi, rb.lastHi = rb.prevHi, rb.lastHi, hi
	rb.thirdContainer, rb.prevContainer, rb.lastContainer = rb.prevContainer, rb.lastContainer, c
}

// Set sets the bit x in the bitmap and grows it if necessary.
func (rb *Bitmap) Set(x uint32) {
	hi, lo := uint16(x>>16), uint16(x&0xFFFF)
	
	// Ultra fast path: check cache first without function call overhead
	if rb.lastContainer != nil && rb.lastHi == hi {
		rb.lastContainer.set(lo)
		return
	}
	if rb.prevContainer != nil && rb.prevHi == hi {
		// Promote to primary cache
		rb.thirdHi, rb.prevHi, rb.lastHi = rb.prevHi, rb.lastHi, hi
		rb.thirdContainer, rb.prevContainer, rb.lastContainer = rb.prevContainer, rb.lastContainer, rb.prevContainer
		rb.lastContainer.set(lo)
		return
	}
	if rb.thirdContainer != nil && rb.thirdHi == hi {
		// Promote to primary cache
		rb.thirdHi, rb.prevHi, rb.lastHi = rb.prevHi, rb.lastHi, hi
		rb.thirdContainer, rb.prevContainer, rb.lastContainer = rb.prevContainer, rb.lastContainer, rb.thirdContainer
		rb.lastContainer.set(lo)
		return
	}
	
	// Slow path: map lookup
	c, exists := rb.containers[hi]
	if !exists {
		// For random patterns, start with bitmap container more aggressively
		c = &container{
			Type: typeArray,
			Size: 0,               // Start with zero cardinality
			Data: make([]byte, 0, 256), // Larger initial capacity for random patterns
		}
		rb.containers[hi] = c
	}
	
	// Update cache (shift down)
	rb.thirdHi, rb.prevHi, rb.lastHi = rb.prevHi, rb.lastHi, hi
	rb.thirdContainer, rb.prevContainer, rb.lastContainer = rb.prevContainer, rb.lastContainer, c
	
	c.set(lo)
}

// Remove removes the bit x from the bitmap, but does not shrink it.
func (rb *Bitmap) Remove(x uint32) {
	hi, lo := uint16(x>>16), uint16(x&0xFFFF)
	c, exists := rb.getContainer(hi)
	if !exists || !c.remove(lo) {
		return
	}

	switch {
	case c.isEmpty():
		delete(rb.containers, hi)
		// Invalidate cache if we deleted the cached container
		if rb.lastContainer == c {
			rb.lastContainer = nil
		}
		if rb.prevContainer == c {
			rb.prevContainer = nil
		}
		if rb.thirdContainer == c {
			rb.thirdContainer = nil
		}
	default:
		rb.setContainer(hi, c)
	}
}

// Contains checks whether a value is contained in the bitmap or not.
func (rb *Bitmap) Contains(x uint32) bool {
	hi, lo := uint16(x>>16), uint16(x&0xFFFF)
	
	// Fast path: check cache first without function call overhead
	if rb.lastContainer != nil && rb.lastHi == hi {
		return rb.lastContainer.contains(lo)
	}
	if rb.prevContainer != nil && rb.prevHi == hi {
		// Promote to primary cache
		rb.thirdHi, rb.prevHi, rb.lastHi = rb.prevHi, rb.lastHi, hi
		rb.thirdContainer, rb.prevContainer, rb.lastContainer = rb.prevContainer, rb.lastContainer, rb.prevContainer
		return rb.lastContainer.contains(lo)
	}
	if rb.thirdContainer != nil && rb.thirdHi == hi {
		// Promote to primary cache
		rb.thirdHi, rb.prevHi, rb.lastHi = rb.prevHi, rb.lastHi, hi
		rb.thirdContainer, rb.prevContainer, rb.lastContainer = rb.prevContainer, rb.lastContainer, rb.thirdContainer
		return rb.lastContainer.contains(lo)
	}
	
	// Slow path: map lookup
	c, exists := rb.containers[hi]
	if !exists {
		return false
	}
	
	// Update cache (shift down)
	rb.thirdHi, rb.prevHi, rb.lastHi = rb.prevHi, rb.lastHi, hi
	rb.thirdContainer, rb.prevContainer, rb.lastContainer = rb.prevContainer, rb.lastContainer, c

	return c.contains(lo)
}

// Count returns the total number of bits set to 1 in the bitmap
func (rb *Bitmap) Count() int {
	count := 0
	for _, c := range rb.containers {
		count += c.cardinality()
	}
	return count
}

// Clear clears the bitmap and resizes it to zero.
func (rb *Bitmap) Clear() {
	rb.containers = make(map[uint16]*container)
	rb.lastContainer = nil
	rb.prevContainer = nil
	rb.thirdContainer = nil
}

// Optimize optimizes all containers to use the most efficient representation.
// This can significantly reduce memory usage, especially after bulk operations.
func (rb *Bitmap) Optimize() {
	for _, c := range rb.containers {
		c.optimize()
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

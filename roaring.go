package roaring

import (
	"io"
)

type cindex struct {
	span  [2]uint8
	count int
}

// cblock represents a block of containers for the two-level index
type cblock struct {
	content [256]*container // 256 slots for containers with same high 8 bits
	cindex
}

// Bitmap represents a roaring bitmap for uint32 values
type Bitmap struct {
	blocks [256]*cblock // Level 1: index by high 8 bits of container key
	cindex
}

// New creates a new empty roaring bitmap
func New() *Bitmap {
	return &Bitmap{}
}

// findContainer finds the container for the given high bits using two-level index
// Returns (container, found) where container is nil if not found
func (rb *Bitmap) findContainer(hi uint16) (*container, bool) {
	hi8, lo8 := uint8(hi>>8), uint8(hi&0xFF)

	// Look up in two-level index
	block := rb.blocks[hi8]
	if block == nil {
		return nil, false
	}

	if c := block.content[lo8]; c != nil {
		return c, true
	}
	return nil, false
}

// setContainer sets a container at the given high bits
func (rb *Bitmap) setContainer(hi uint16, c *container) {
	c.Key = hi // Set the key in the container
	hi8, lo8 := uint8(hi>>8), uint8(hi&0xFF)

	// Get or create the block
	block := rb.blocks[hi8]
	if block == nil {
		block = &cblock{cindex: cindex{span: [2]uint8{lo8, lo8}}}
		rb.blocks[hi8] = block
		rb.count++

		// Update bitmap-level span for new block
		if rb.count == 1 {
			// First block in bitmap
			rb.span[0] = hi8
			rb.span[1] = hi8
		} else {
			// Update bounds
			if hi8 < rb.span[0] {
				rb.span[0] = hi8
			}
			if hi8 > rb.span[1] {
				rb.span[1] = hi8
			}
		}
	}

	// Set the container in the block
	if block.content[lo8] == nil {
		block.count++
		// Update span for new container
		if block.count == 1 {
			// First container in block
			block.span[0] = lo8
			block.span[1] = lo8
		} else {
			// Update bounds
			if lo8 < block.span[0] {
				block.span[0] = lo8
			}
			if lo8 > block.span[1] {
				block.span[1] = lo8
			}
		}
	}
	block.content[lo8] = c
}

// removeContainer removes the container at the given high bits
func (rb *Bitmap) removeContainer(hi uint16) {
	hi8, lo8 := uint8(hi>>8), uint8(hi&0xFF)

	block := rb.blocks[hi8]
	if block == nil {
		return
	}

	if block.content[lo8] != nil {
		block.content[lo8] = nil
		block.count--

		// If block is empty, remove it
		if block.count == 0 {
			rb.blocks[hi8] = nil
			rb.count--

			// Update bitmap-level span if necessary
			if rb.count == 0 {
				// Bitmap is empty - bounds don't matter
				return
			} else if hi8 == rb.span[0] || hi8 == rb.span[1] {
				// Need to recalculate bitmap bounds
				rb.updateBitmapBounds()
			}
			return
		}

		// Update span if necessary
		if lo8 == block.span[0] || lo8 == block.span[1] {
			// Need to recalculate bounds
			block.updateBounds()
		}
	}
}

// updateBounds recalculates the min/max indices for a block
func (b *cblock) updateBounds() {
	if b.count == 0 {
		return
	}

	// Find new min
	for i := 0; i < 256; i++ {
		if b.content[i] != nil {
			b.span[0] = uint8(i)
			break
		}
	}

	// Find new max
	for i := 255; i >= 0; i-- {
		if b.content[i] != nil {
			b.span[1] = uint8(i)
			break
		}
	}
}

// updateBitmapBounds recalculates the min/max block indices for the bitmap
func (rb *Bitmap) updateBitmapBounds() {
	if rb.count == 0 {
		return
	}

	// Find new min block
	for i := 0; i < 256; i++ {
		if rb.blocks[i] != nil {
			rb.span[0] = uint8(i)
			break
		}
	}

	// Find new max block
	for i := 255; i >= 0; i-- {
		if rb.blocks[i] != nil {
			rb.span[1] = uint8(i)
			break
		}
	}
}

// Set sets the bit x in the bitmap and grows it if necessary.
func (rb *Bitmap) Set(x uint32) {
	hi, lo := uint16(x>>16), uint16(x&0xFFFF)
	c, exists := rb.findContainer(hi)
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
	c, exists := rb.findContainer(hi)
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
	c, exists := rb.findContainer(hi)
	if !exists {
		return false
	}

	return c.contains(lo)
}

// Count returns the total number of bits set to 1 in the bitmap
func (rb *Bitmap) Count() int {
	if rb.count == 0 {
		return 0
	}

	count := 0
	for i := int(rb.span[0]); i <= int(rb.span[1]); i++ {
		block := rb.blocks[i]
		if block != nil {
			for j := int(block.span[0]); j <= int(block.span[1]); j++ {
				c := block.content[j]
				if c != nil {
					count += c.cardinality()
				}
			}
		}
	}
	return count
}

// Clear clears the bitmap and resizes it to zero.
func (rb *Bitmap) Clear() {
	rb.blocks = [256]*cblock{} // Clear all blocks
	rb.count = 0
	rb.span[0] = 0
	rb.span[1] = 0
}

// Optimize optimizes all containers to use the most efficient representation.
// This can significantly reduce memory usage, especially after bulk operations.
func (rb *Bitmap) Optimize() {
	if rb.count == 0 {
		return
	}

	for i := int(rb.span[0]); i <= int(rb.span[1]); i++ {
		block := rb.blocks[i]
		if block != nil {
			for j := int(block.span[0]); j <= int(block.span[1]); j++ {
				c := block.content[j]
				if c != nil {
					c.optimize()
				}
			}
		}
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

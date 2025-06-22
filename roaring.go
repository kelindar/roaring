package roaring

import "io"

// Bitmap represents a roaring bitmap for uint32 values
type Bitmap struct {
	containers map[uint16]container
}

// New creates a new empty roaring bitmap
func New() *Bitmap {
	return &Bitmap{
		containers: make(map[uint16]container),
	}
}

// Set sets the bit x in the bitmap and grows it if necessary.
func (rb *Bitmap) Set(x uint32) {
	panic("not implemented")
}

// Remove removes the bit x from the bitmap, but does not shrink it.
func (rb *Bitmap) Remove(x uint32) {
	panic("not implemented")
}

// Contains checks whether a value is contained in the bitmap or not.
func (rb *Bitmap) Contains(x uint32) bool {
	panic("not implemented")
}

// Count returns the total number of bits set to 1 in the bitmap
func (rb *Bitmap) Count() int {
	panic("not implemented")
}

// Grow grows the bitmap size until we reach the desired bit.
func (rb *Bitmap) Grow(desiredBit uint32) {
	panic("not implemented")
}

// Clear clears the bitmap and resizes it to zero.
func (rb *Bitmap) Clear() {
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

// Clone clones the bitmap
func (rb *Bitmap) Clone(into *Bitmap) *Bitmap {
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

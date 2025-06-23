package roaring

import (
	"unsafe"

	"github.com/kelindar/bitmap"
)

type ctype byte

const (
	typeArray ctype = iota
	typeBitmap
	typeRun
)

type container struct {
	Type ctype  // Type of the container
	Size uint16 // Cardinality
	Data []byte // Data of the container
}

type run [2]uint16

// bitmap converts the container to a bitmap.Bitmap
func (c *container) bitmap() bitmap.Bitmap {
	if len(c.Data) == 0 {
		return nil
	}

	return bitmap.Bitmap(unsafe.Slice((*uint64)(unsafe.Pointer(&c.Data[0])), len(c.Data)/8))
}

// array converts the container to an []uint16
func (c *container) array() []uint16 {
	if len(c.Data) == 0 {
		return nil
	}

	return unsafe.Slice((*uint16)(unsafe.Pointer(&c.Data[0])), len(c.Data)/2)
}

// run converts the container to a []run
func (c *container) run() []run {
	if len(c.Data) == 0 {
		return nil
	}

	return unsafe.Slice((*run)(unsafe.Pointer(&c.Data[0])), len(c.Data)/4)
}

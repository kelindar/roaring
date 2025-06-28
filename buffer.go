package roaring

import (
	"sync"
	"unsafe"

	"github.com/kelindar/bitmap"
)

const bitmapSize = 4096

var pool = sync.Pool{
	New: func() any {
		return make([]uint16, 0, bitmapSize)
	},
}

func borrowArray() []uint16 {
	return pool.Get().([]uint16)
}

func borrowBitmap() bitmap.Bitmap {
	arr := borrowArray()
	if cap(arr) < bitmapSize {
		arr = make([]uint16, bitmapSize)
	}

	// Clear the memory to ensure clean bitmap
	out := asBitmap(arr[:bitmapSize])
	for i := range out {
		out[i] = 0
	}
	return out
}

func release(v any) {
	switch v := v.(type) {
	case []uint16:
		pool.Put(v[:0])
	case bitmap.Bitmap:
		pool.Put(asUint16s(v[:0]))
	}
}

func asBitmap(data []uint16) bitmap.Bitmap {
	if len(data) == 0 {
		return nil
	}

	return bitmap.Bitmap(unsafe.Slice((*uint64)(unsafe.Pointer(&data[0])), len(data)/4))
}

func asUint16s(data bitmap.Bitmap) []uint16 {
	return unsafe.Slice((*uint16)(unsafe.Pointer(&data[0])), len(data)*4)
}

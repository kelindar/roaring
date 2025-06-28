package roaring

import (
	"sync"
	"unsafe"

	"github.com/kelindar/bitmap"
)

var pool = sync.Pool{
	New: func() any {
		return make([]uint16, 4096)
	},
}

func borrowArray() []uint16 {
	return pool.Get().([]uint16)
}

func borrowBitmap() bitmap.Bitmap {
	return asBitmap(borrowArray())
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
	return unsafe.Slice((*uint16)(unsafe.Pointer(&data[0])), len(data))
}

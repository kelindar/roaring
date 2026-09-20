// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

package roaring

import (
	"bytes"
	"encoding/binary"
	"io"
	"unsafe"
)

var isLittleEndian = binary.NativeEndian.Uint16([]byte{1, 0}) == 1

const (
	bitmapSizeMask  = (1 << 14) - 1
	bitmapCardShift = 14
)

// ToBytes converts the bitmap to a byte slice
func (rb *Bitmap) ToBytes() []byte {
	var buf bytes.Buffer
	if _, err := rb.WriteTo(&buf); err != nil {
		panic(err)
	}

	return buf.Bytes()
}

// WriteTo writes the bitmap to a writer
func (rb *Bitmap) WriteTo(w io.Writer) (int64, error) {
	if buf, ok := w.(*bytes.Buffer); ok {
		buf.Grow(rb.serializedSize())
	}

	var n int64
	var header [7]byte

	// Write number of containers
	count := uint32(len(rb.containers))
	binary.LittleEndian.PutUint32(header[:4], count)
	written, err := writeBytes(w, header[:4])
	n += int64(written)
	if err != nil {
		return n, err
	}

	for i, c := range rb.containers {
		key := rb.index[i]
		binary.LittleEndian.PutUint16(header[:2], key)
		header[2] = byte(c.Type)

		// Prepare payload
		var payload []uint16
		var sizeBytes uint32
		switch c.Type {
		case typeArray:
			payload = c.Data[:len(c.Data)]
			sizeBytes = uint32(len(payload)) * 2
		case typeBitmap:
			payload = c.Data[:4096] // Bitmap containers always have a fixed size of 4096 uint16s
			// The payload size is fixed, so use the high bits to persist the
			// cardinality and avoid recounting every bitmap while decoding.
			sizeBytes = uint32(len(payload))*2 | c.Size<<bitmapCardShift
		case typeRun:
			payload = c.Data[:len(c.Data)]
			sizeBytes = uint32(len(payload)) * 2
		default:
			written, err = writeBytes(w, header[:3])
			n += int64(written)
			if err != nil {
				return n, err
			}
			return n, io.ErrUnexpectedEOF
		}

		binary.LittleEndian.PutUint32(header[3:], sizeBytes)
		written, err = writeBytes(w, header[:])
		n += int64(written)
		if err != nil {
			return n, err
		}

		// Write payload ([]uint16)
		written, err = writeUint16sCount(w, isLittleEndian, payload)
		n += int64(written)
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func (rb *Bitmap) serializedSize() int {
	size := 4
	for _, c := range rb.containers {
		payload := len(c.Data)
		if c.Type == typeBitmap {
			payload = 4096
		}
		size += 7 + payload*2
	}
	return size
}

// ReadFrom reads the bitmap from a reader
func (rb *Bitmap) ReadFrom(r io.Reader) (int64, error) {
	rb.Clear()
	rb.scratch = nil
	var n int64
	var header [7]byte

	// Read number of containers
	read, err := io.ReadFull(r, header[:4])
	n += int64(read)
	if err != nil {
		return n, err
	}
	count := binary.LittleEndian.Uint32(header[:4])
	if count > 1<<16 {
		return n, io.ErrUnexpectedEOF
	}
	if cap(rb.containers) < int(count) {
		rb.containers = make([]container, 0, count)
	}
	if cap(rb.index) < int(count) {
		rb.index = make([]uint16, 0, count)
	}
	type payloadRange struct {
		start int
		end   int
	}
	ranges := make([]payloadRange, 0, count)
	var storage []uint16

	for i := uint32(0); i < count; i++ {
		read, err = io.ReadFull(r, header[:])
		n += int64(read)
		if err != nil {
			return n, err
		}
		key := binary.LittleEndian.Uint16(header[:2])
		typ := ctype(header[2])
		sizeBytes := binary.LittleEndian.Uint32(header[3:])
		cardinality := uint32(0)
		if typ == typeBitmap {
			cardinality = sizeBytes >> bitmapCardShift
			sizeBytes &= bitmapSizeMask
		}

		payloadSize := int(sizeBytes)
		if payloadSize%2 != 0 {
			return n, io.ErrUnexpectedEOF
		}
		payloadCount := payloadSize / 2
		if cap(storage)-len(storage) < payloadCount {
			newCap := cap(storage) * 2
			if len(storage) == 0 && cap(storage) == 0 {
				newCap = payloadCount * int(count)
			}
			if newCap < len(storage)+payloadCount {
				newCap = len(storage) + payloadCount
			}
			if newCap > 1<<20 {
				newCap = 1 << 20
				if newCap < len(storage)+payloadCount {
					newCap = len(storage) + payloadCount
				}
			}
			if newCap == 0 {
				newCap = payloadCount
			}
			grown := make([]uint16, len(storage), newCap)
			copy(grown, storage)
			storage = grown
		}
		start := len(storage)
		payload, read, err := readUint16sReuse(r, isLittleEndian, payloadSize, storage[start:start])
		n += int64(read)
		if err != nil {
			return n, err
		}
		storage = storage[:start+len(payload)]
		ranges = append(ranges, payloadRange{start: start, end: len(storage)})

		switch typ {
		case typeArray:
			rb.ctrAdd(key, len(rb.containers), &container{
				Type: typ,
				Size: uint32(len(payload)),
			})
		case typeBitmap:
			if len(payload) != 4096 {
				return n, io.ErrUnexpectedEOF
			}

			// Older streams did not persist cardinality; retain the fallback.
			sz := cardinality
			if sz == 0 {
				sz = uint32(asBitmap(payload).Count())
			}
			rb.ctrAdd(key, len(rb.containers), &container{
				Type: typ,
				Size: sz,
			})
		case typeRun:
			if len(payload)%2 != 0 {
				return n, io.ErrUnexpectedEOF
			}

			// Calculate run cardinality
			sz := uint32(0)
			for i := 0; i+1 < len(payload); i += 2 {
				sz += uint32(payload[i+1]-payload[i]) + 1
			}
			rb.ctrAdd(key, len(rb.containers), &container{
				Type: typ,
				Size: sz,
			})
		default:
			return n, io.ErrUnexpectedEOF
		}
	}
	for i, span := range ranges {
		rb.containers[i].Data = storage[span.start:span.end:span.end]
	}
	return n, nil
}

// FromBytes creates a roaring bitmap from a byte buffer
func FromBytes(buffer []byte) *Bitmap {
	rb := New()
	n, err := rb.ReadFrom(bytes.NewReader(buffer))
	if err != nil && (err != io.EOF || n != 0) {
		panic(err)
	}
	return rb
}

// ReadFrom reads a roaring bitmap from an io.Reader
func ReadFrom(r io.Reader) (*Bitmap, error) {
	rb := New()
	n, err := rb.ReadFrom(r)
	if err != nil && (err != io.EOF || n != 0) {
		return nil, err
	}
	return rb, nil
}

// writeUint16s writes a slice of uint16s to a writer, converting it to []byte if
// the machine is little endian.
func writeUint16s(w io.Writer, isLittleEndian bool, data []uint16) error {
	_, err := writeUint16sCount(w, isLittleEndian, data)
	return err
}

func writeUint16sCount(w io.Writer, isLittleEndian bool, data []uint16) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}

	if isLittleEndian {
		buf := unsafe.Slice((*byte)(unsafe.Pointer(&data[0])), len(data)*2)
		return writeBytes(w, buf)
	}

	buf := make([]byte, len(data)*2)
	for i, value := range data {
		binary.LittleEndian.PutUint16(buf[i*2:], value)
	}
	return writeBytes(w, buf)
}

// readUint16s reads a slice of uint16s from a reader, converting it to []uint16 if
// the machine is little endian.
func readUint16s(r io.Reader, isLittleEndian bool, sizeBytes int) ([]uint16, error) {
	out, _, err := readUint16sCount(r, isLittleEndian, sizeBytes)
	return out, err
}

func readUint16sCount(r io.Reader, isLittleEndian bool, sizeBytes int) ([]uint16, int, error) {
	if sizeBytes < 0 || sizeBytes%2 != 0 {
		return nil, 0, io.ErrUnexpectedEOF
	}

	count := sizeBytes / 2
	out := make([]uint16, count)
	if count == 0 {
		return out, 0, nil
	}

	if isLittleEndian {
		buf := unsafe.Slice((*byte)(unsafe.Pointer(&out[0])), sizeBytes)
		read, err := io.ReadFull(r, buf)
		return out, read, err
	}

	buf := make([]byte, sizeBytes)
	read, err := io.ReadFull(r, buf)
	if err != nil {
		return out, read, err
	}
	for i := range out {
		out[i] = binary.LittleEndian.Uint16(buf[i*2:])
	}
	return out, read, nil
}

func readUint16sReuse(r io.Reader, isLittleEndian bool, sizeBytes int, out []uint16) ([]uint16, int, error) {
	if sizeBytes < 0 || sizeBytes%2 != 0 {
		return nil, 0, io.ErrUnexpectedEOF
	}

	count := sizeBytes / 2
	if cap(out) < count {
		out = make([]uint16, count)
	} else {
		out = out[:count]
	}
	if count == 0 {
		return out, 0, nil
	}

	if isLittleEndian {
		buf := unsafe.Slice((*byte)(unsafe.Pointer(&out[0])), sizeBytes)
		read, err := io.ReadFull(r, buf)
		return out, read, err
	}

	buf := make([]byte, sizeBytes)
	read, err := io.ReadFull(r, buf)
	if err != nil {
		return out, read, err
	}
	for i := range out {
		out[i] = binary.LittleEndian.Uint16(buf[i*2:])
	}
	return out, read, nil
}

func writeBytes(w io.Writer, data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}

	written, err := w.Write(data)
	if written < 0 || written > len(data) {
		return 0, io.ErrShortWrite
	}
	if err != nil {
		return written, err
	}
	if written != len(data) {
		return written, io.ErrShortWrite
	}
	return written, nil
}

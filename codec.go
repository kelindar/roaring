package roaring

import (
	"bytes"
	"encoding/binary"
	"io"
	"math/bits"
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
	var n int64
	for i, c := range rb.containers {
		key := rb.index[i]

		// Write key (uint16)
		if err := binary.Write(w, binary.LittleEndian, key); err != nil {
			return n, err
		}
		n += 2

		// Write type (byte)
		if err := binary.Write(w, binary.LittleEndian, c.Type); err != nil {
			return n, err
		}
		n += 1

		// Prepare payload
		var payload []uint16
		var sizeBytes uint32
		switch c.Type {
		case typeArray:
			payload = c.Data[:len(c.Data)]
			sizeBytes = uint32(len(payload)) * 2
		case typeBitmap:
			payload = c.Data[:4096]
			sizeBytes = 4096 * 2
		case typeRun:
			payload = c.Data[:len(c.Data)]
			sizeBytes = uint32(len(payload)) * 2
		default:
			return n, io.ErrUnexpectedEOF
		}

		// Write size (uint32)
		if err := binary.Write(w, binary.LittleEndian, sizeBytes); err != nil {
			return n, err
		}
		n += 4

		// Write payload ([]uint16)
		if err := binary.Write(w, binary.LittleEndian, payload); err != nil {
			return n, err
		}
		n += int64(sizeBytes)
	}
	return n, nil
}

// ReadFrom reads the bitmap from a reader
func (rb *Bitmap) ReadFrom(r io.Reader) (int64, error) {
	rb.Clear()
	var n int64
	for {
		var key uint16
		if err := binary.Read(r, binary.LittleEndian, &key); err != nil {
			if err == io.EOF {
				break
			}
			return n, err
		}
		n += 2

		var typ ctype
		if err := binary.Read(r, binary.LittleEndian, &typ); err != nil {
			return n, err
		}
		n += 1

		var sizeBytes uint32
		if err := binary.Read(r, binary.LittleEndian, &sizeBytes); err != nil {
			return n, err
		}
		n += 4

		count := sizeBytes / 2
		payload := make([]uint16, count)
		if err := binary.Read(r, binary.LittleEndian, payload); err != nil {
			return n, err
		}
		n += int64(sizeBytes)

		switch typ {
		case typeArray:
			rb.ctrAdd(key, len(rb.containers), &container{
				Type: typ,
				Size: uint32(len(payload)),
				Data: payload,
			})
		case typeBitmap:
			// Count bits set for Size
			sz := uint32(0)
			for _, v := range payload {
				sz += uint32(bits.OnesCount16(v))
			}
			rb.ctrAdd(key, len(rb.containers), &container{
				Type: typ,
				Size: sz,
				Data: payload,
			})
		case typeRun:
			// Calculate run cardinality
			sz := uint32(0)
			for i := 0; i+1 < len(payload); i += 2 {
				sz += uint32(payload[i+1]-payload[i]) + 1
			}
			rb.ctrAdd(key, len(rb.containers), &container{
				Type: typ,
				Size: sz,
				Data: payload,
			})
		default:
			return n, io.ErrUnexpectedEOF
		}
	}
	return n, nil
}

// FromBytes creates a roaring bitmap from a byte buffer
func FromBytes(buffer []byte) *Bitmap {
	rb := New()
	_, err := rb.ReadFrom(bytes.NewReader(buffer))
	if err != nil && err != io.EOF {
		panic(err)
	}
	return rb
}

// ReadFrom reads a roaring bitmap from an io.Reader
func ReadFrom(r io.Reader) (*Bitmap, error) {
	rb := New()
	_, err := rb.ReadFrom(r)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return rb, nil
}

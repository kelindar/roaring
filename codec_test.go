// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root

package roaring

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

var errTestWriter = errors.New("write failed")

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) {
	return 0, errTestWriter
}

type shortWriter struct {
	remaining int
}

func (w *shortWriter) Write(p []byte) (int, error) {
	if w.remaining == 0 {
		return 0, io.ErrShortWrite
	}
	n := len(p)
	if n > w.remaining {
		n = w.remaining
	}
	w.remaining -= n
	return n, nil
}

func makeTestBitmap() *Bitmap {
	rb := New()

	// Array container
	rb.Set(1)
	rb.Set(5)
	rb.Set(10)

	// Bitmap container
	for i := 0xFFFF; i < 0xFFFF+0x5FFF; i += 3 {
		rb.Set(uint32(i))
	}

	// Run container
	for i := 131072; i < 131072+1000; i++ {
		rb.Set(uint32(i))
	}

	// Max uint32
	rb.Set(4294967295)

	rb.Optimize()
	return rb
}

func bitmapsEqual(t *testing.T, a, b *Bitmap) {
	t.Helper()
	assert.Equal(t, a.Count(), b.Count(), "Count mismatch")
	var av, bv []uint32
	a.Range(func(x uint32) bool { av = append(av, x); return true })
	b.Range(func(x uint32) bool { bv = append(bv, x); return true })
	assert.Equal(t, av, bv, "Values mismatch")
}

func TestCodec_ToBytes_FromBytes(t *testing.T) {
	rb := makeTestBitmap()
	data := rb.ToBytes()
	rb2 := FromBytes(data)
	bitmapsEqual(t, rb, rb2)
}

func TestCodec_WriteTo_ReadFrom_Methods(t *testing.T) {
	rb := makeTestBitmap()
	var buf bytes.Buffer
	_, err := rb.WriteTo(&buf)
	assert.NoError(t, err)

	rb2 := New()
	_, err = rb2.ReadFrom(bytes.NewReader(buf.Bytes()))
	assert.NoError(t, err)
	bitmapsEqual(t, rb, rb2)
}

func TestCodec_Package_ReadFrom(t *testing.T) {
	rb := makeTestBitmap()
	var buf bytes.Buffer
	_, err := rb.WriteTo(&buf)
	assert.NoError(t, err)

	rb2, err := ReadFrom(bytes.NewReader(buf.Bytes()))
	assert.NoError(t, err)
	bitmapsEqual(t, rb, rb2)
}

func TestCodec_EmptyBitmap(t *testing.T) {
	rb := New()
	data := rb.ToBytes()
	rb2 := FromBytes(data)
	bitmapsEqual(t, rb, rb2)
}

func TestCodec_SingleValue(t *testing.T) {
	rb := New()
	rb.Set(42)
	data := rb.ToBytes()
	rb2 := FromBytes(data)
	bitmapsEqual(t, rb, rb2)
}

func TestCodec_DenseBitmap(t *testing.T) {
	rb := New()
	for i := 0; i < 70000; i++ {
		rb.Set(uint32(i))
	}
	data := rb.ToBytes()
	rb2 := FromBytes(data)
	bitmapsEqual(t, rb, rb2)
}

func TestCodec_SparseRandom(t *testing.T) {
	rb := New()
	for i := 0; i < 1000; i++ {
		rb.Set(uint32(rand.Intn(1 << 24)))
	}
	data := rb.ToBytes()
	rb2 := FromBytes(data)
	bitmapsEqual(t, rb, rb2)
}

func TestCodec_BigEndian(t *testing.T) {
	data := []uint16{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	assert.NoError(t, writeUint16s(&bytes.Buffer{}, true, nil))

	var buf1 bytes.Buffer
	assert.NoError(t, writeUint16s(&buf1, true, data))

	var buf2 bytes.Buffer
	assert.NoError(t, writeUint16s(&buf2, false, data))

	assert.Equal(t, buf1.Bytes(), buf2.Bytes())

	out1, err := readUint16s(&buf1, true, len(data)*2)
	assert.NoError(t, err)
	assert.Equal(t, data, out1)

	out2, err := readUint16s(&buf2, false, len(data)*2)
	assert.NoError(t, err)
	assert.Equal(t, data, out2)
}

func TestCodecCounts(t *testing.T) {
	bm := bitmapOf(1, 2)
	encoded := bm.ToBytes()
	for cut := 0; cut < len(encoded); cut++ {
		n, err := bm.WriteTo(&shortWriter{remaining: cut})
		assert.Equal(t, int64(cut), n)
		assert.ErrorIs(t, err, io.ErrShortWrite)

		n, err = New().ReadFrom(bytes.NewReader(encoded[:cut]))
		assert.Equal(t, int64(cut), n)
		assert.Error(t, err)
	}
}
func TestCodecErrors(t *testing.T) {
	t.Run("write error", func(t *testing.T) {
		_, err := bitmapOf(1).WriteTo(errWriter{})
		assert.ErrorIs(t, err, errTestWriter)
	})

	tests := []struct {
		name string
		data []byte
	}{
		{"truncated", []byte{1, 0}},
		{"invalid type", codecRecord(ctype(99), 0, nil)},
		{"odd payload", codecRecord(typeArray, 1, []byte{1})},
		{"short bitmap", codecRecord(typeBitmap, 2, codecPayload(1))},
		{"malformed run", codecRecord(typeRun, 2, codecPayload(1))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rb := New()
			_, err := rb.ReadFrom(bytes.NewReader(tt.data))
			assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
		})
	}
}

func codecRecord(typ ctype, size uint32, payload []byte) []byte {
	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.LittleEndian, uint32(1))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(0))
	_ = buf.WriteByte(byte(typ))
	_ = binary.Write(&buf, binary.LittleEndian, size)
	_, _ = buf.Write(payload)
	return buf.Bytes()
}

func codecPayload(values ...uint16) []byte {
	out := make([]byte, len(values)*2)
	for i, v := range values {
		binary.LittleEndian.PutUint16(out[i*2:], v)
	}
	return out
}

func TestCodecTruncated(t *testing.T) {
	encoded := bitmapOf(1, 2).ToBytes()
	for cut := 1; cut < len(encoded); cut++ {
		_, err := ReadFrom(bytes.NewReader(encoded[:cut]))
		assert.Error(t, err)
		assert.Panics(t, func() { FromBytes(encoded[:cut]) })
	}
	_, err := ReadFrom(bytes.NewReader(nil))
	assert.NoError(t, err)
	assert.NotPanics(t, func() { FromBytes(nil) })
}

package roaring

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func makeTestBitmap() *Bitmap {
	rb := New()
	// Array container
	rb.Set(1)
	rb.Set(5)
	rb.Set(10)
	// Bitmap container
	for i := 65536; i < 65536+5000; i += 3 {
		rb.Set(uint32(i))
	}
	// Run container
	for i := 131072; i < 131072+100; i++ {
		rb.Set(uint32(i))
	}
	// Max uint32
	rb.Set(4294967295)
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

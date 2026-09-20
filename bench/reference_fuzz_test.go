package main

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/RoaringBitmap/roaring"
	rb "github.com/kelindar/roaring"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func FuzzReference(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0, 0, 0, 0, 1, 0, 0, 0})
	f.Add([]byte{1, 0xff, 0xff, 0xff, 0xff, 2, 0, 0, 1})
	f.Add([]byte{3, 0, 0, 1, 0, 4, 0xff, 0xff, 0, 0, 5, 0, 0, 0, 1})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 4096 {
			data = data[:4096]
		}

		ours := rb.New()
		reference := roaring.NewBitmap()
		other := rb.New()
		otherReference := roaring.NewBitmap()
		third := rb.New()
		thirdReference := roaring.NewBitmap()

		for i := 0; i < len(data); i += 5 {
			value := fuzzValue(data, i+1)
			switch data[i] % 25 {
			case 0:
				ours.Set(value)
				reference.Add(value)
			case 1:
				ours.Remove(value)
				reference.Remove(value)
			case 2:
				other.Set(value)
				otherReference.Add(value)
			case 3:
				other.Remove(value)
				otherReference.Remove(value)
			case 4:
				third.Set(value)
				thirdReference.Add(value)
			case 5:
				third.Remove(value)
				thirdReference.Remove(value)
			case 6:
				assert.Equal(t, reference.Contains(value), ours.Contains(value))
			case 7:
				ours.And(other)
				reference.And(otherReference.Clone())
			case 8:
				ours.Or(other)
				reference.Or(otherReference.Clone())
			case 9:
				ours.Xor(other)
				reference.Xor(otherReference.Clone())
			case 10:
				ours.AndNot(other)
				reference.AndNot(otherReference.Clone())
			case 11:
				ours.And(other, third)
				reference.And(otherReference.Clone())
				reference.And(thirdReference.Clone())
			case 12:
				ours.Or(other, nil, third)
				reference.Or(otherReference.Clone())
				reference.Or(thirdReference.Clone())
			case 13:
				ours.Xor(other, third)
				reference.Xor(otherReference.Clone())
				reference.Xor(thirdReference.Clone())
			case 14:
				ours.AndNot(other, third)
				reference.AndNot(otherReference.Clone())
				reference.AndNot(thirdReference.Clone())
			case 15:
				ours.Optimize()
				reference.RunOptimize()
			case 16:
				ours.Clear()
				reference.Clear()
			case 17:
				compareClone(t, ours, reference, value)
			case 18:
				compareExtrema(t, ours, reference)
			case 19:
				mask := fuzzValue(data, i+2)
				keep := func(candidate uint32) bool {
					return (candidate^value)&mask == 0
				}
				ours.Filter(keep)
				filterReference(reference, keep)
			case 20:
				compareCodec(t, ours, reference)
			case 21:
				compareRange(t, ours, reference, int(fuzzValue(data, i+1)&31))
			case 22:
				comparePackageReadFrom(t, ours, reference)
			case 23:
				ours.And(ours)
				reference.And(reference.Clone())
			case 24:
				ours.Xor(ours, other)
				reference.Xor(reference.Clone())
				reference.Xor(otherReference.Clone())
			}
			compareBitmaps(t, ours, reference)
			compareBitmaps(t, other, otherReference)
			compareBitmaps(t, third, thirdReference)
		}

		compareCodec(t, ours, reference)
	})
}

func fuzzValue(data []byte, offset int) uint32 {
	var encoded [4]byte
	if offset < len(data) {
		copy(encoded[:], data[offset:])
	}
	return binary.LittleEndian.Uint32(encoded[:])
}

func compareBitmaps(t *testing.T, ours *rb.Bitmap, reference *roaring.Bitmap) {
	t.Helper()
	assert.Equal(t, int(reference.GetCardinality()), ours.Count())
	assert.Equal(t, reference.IsEmpty(), ours.Count() == 0)

	values := make([]uint32, 0, ours.Count())
	ours.Range(func(value uint32) bool {
		values = append(values, value)
		return true
	})
	assert.Equal(t, reference.ToArray(), values)
}

func compareClone(t *testing.T, ours *rb.Bitmap, reference *roaring.Bitmap, value uint32) {
	t.Helper()
	clone := ours.Clone(nil)
	compareBitmaps(t, clone, reference)

	reused := rb.New()
	reused.Set(value ^ 0x9e3779b9)
	ours.Clone(reused)
	compareBitmaps(t, reused, reference)

	clone.Set(value ^ 0x85ebca6b)
	assert.True(t, clone.Contains(value^0x85ebca6b))
	compareBitmaps(t, ours, reference)
}

func compareExtrema(t *testing.T, ours *rb.Bitmap, reference *roaring.Bitmap) {
	t.Helper()
	values := reference.ToArray()

	min, minOK := ours.Min()
	max, maxOK := ours.Max()
	assert.Equal(t, len(values) > 0, minOK)
	assert.Equal(t, len(values) > 0, maxOK)
	if len(values) > 0 {
		assert.Equal(t, values[0], min)
		assert.Equal(t, values[len(values)-1], max)
	}

	wantZero, wantZeroOK := firstMissing(values)
	gotZero, gotZeroOK := ours.MinZero()
	assert.Equal(t, wantZeroOK, gotZeroOK)
	if wantZeroOK {
		assert.Equal(t, wantZero, gotZero)
	}
}

func firstMissing(values []uint32) (uint32, bool) {
	var candidate uint64
	for _, value := range values {
		if uint64(value) != candidate {
			return uint32(candidate), true
		}
		candidate++
	}
	if candidate <= uint64(^uint32(0)) {
		return uint32(candidate), true
	}
	return 0, false
}

func filterReference(reference *roaring.Bitmap, keep func(uint32) bool) {
	values := reference.ToArray()
	reference.Clear()
	for _, value := range values {
		if keep(value) {
			reference.Add(value)
		}
	}
}

func compareRange(t *testing.T, ours *rb.Bitmap, reference *roaring.Bitmap, limit int) {
	t.Helper()
	values := make([]uint32, 0)
	ours.Range(func(value uint32) bool {
		if len(values) >= limit {
			return false
		}
		values = append(values, value)
		return len(values) < limit
	})

	want := reference.ToArray()
	if len(want) > len(values) {
		want = want[:len(values)]
	}
	assert.Equal(t, want, values)
}

func compareCodec(t *testing.T, ours *rb.Bitmap, reference *roaring.Bitmap) {
	t.Helper()

	encoded := ours.ToBytes()
	decoded := rb.FromBytes(encoded)
	compareBitmaps(t, decoded, reference)

	var buffer bytes.Buffer
	_, err := ours.WriteTo(&buffer)
	require.NoError(t, err)
	decoded = rb.New()
	_, err = decoded.ReadFrom(bytes.NewReader(buffer.Bytes()))
	require.NoError(t, err)
	compareBitmaps(t, decoded, reference)
}

func comparePackageReadFrom(t *testing.T, ours *rb.Bitmap, reference *roaring.Bitmap) {
	t.Helper()
	encoded := ours.ToBytes()
	decoded, err := rb.ReadFrom(bytes.NewReader(encoded))
	require.NoError(t, err)
	compareBitmaps(t, decoded, reference)
}

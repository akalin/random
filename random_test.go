package random

import (
	"fmt"
	"testing"
)

type singleSource struct {
	i         uint32
	callCount uint32
}

func (src *singleSource) Uint32() uint32 {
	if src.callCount == 0 {
		src.callCount++
		return src.i
	}

	if src.callCount == 1 {
		src.callCount++
		return 0xffffffff
	}

	panic("called when callCount > 1")
}

func computeRange(i, n uint32) (uint32, uint32, uint32) {
	thresh := -n % n
	firstProd := uint64(i) << 32
	firstValidProd := uint64(i)<<32 + uint64(thresh)
	lastValidProd := uint64(i+1)<<32 - 1
	firstV := uint32((firstProd + uint64(n-1)) / uint64(n))
	firstValidV := uint32((firstValidProd + uint64(n-1)) / uint64(n))
	lastValidV := uint32(lastValidProd / uint64(n))
	return firstV, firstValidV, lastValidV
}

func expectUniformUint32Returns(t *testing.T, n, v, expected uint32) {
	t.Helper()
	src := singleSource{i: v}
	actual := UniformUint32(&src, n)
	if actual != expected {
		t.Errorf("(v=%d) expected %d, got %d", v, expected, actual)
	}
}

func expectUniformUint32Rejects(t *testing.T, n, v uint32) {
	t.Helper()
	src := singleSource{i: v}
	actual := UniformUint32(&src, n)
	if src.callCount != 2 {
		t.Errorf("(v=%d) expected %d, got %d (actual=%d)", v, 2, src.callCount, actual)
	}
}

func TestUniformUint32Range(t *testing.T) {
	var n uint32 = 5
	var prevLastValidV uint32
	for i := uint32(0); i < n; i++ {
		firstV, firstValidV, lastValidV := computeRange(i, n)
		if i == 0 {
			if firstV != 0 {
				t.Errorf("(i=0) expected firstV=0, got %d", firstV)
			}
		} else if prevLastValidV+1 != firstV {
			t.Errorf("(i=%d) expected firstV=%d to follow prevLastValidV=%d", i, firstV, prevLastValidV)
		}

		if i == n-1 {
			if lastValidV != 0xffffffff {
				t.Errorf("(i=%d) expected lastValidV=0xffffffff, got %d", i, lastValidV)
			}
		}

		for i := firstV; i < firstValidV; i++ {
			expectUniformUint32Rejects(t, n, i)
		}
		expectUniformUint32Returns(t, n, firstValidV, i)
		delta := (lastValidV - firstValidV) / 1000
		if delta == 0 {
			delta = 1
		}
		for j := uint64(firstValidV) + 1; j < uint64(lastValidV); j += uint64(delta) {
			expectUniformUint32Returns(t, n, uint32(j), i)
		}
		expectUniformUint32Returns(t, n, lastValidV, i)
		prevLastValidV = lastValidV
	}
}

func TestUniformUint32IsUniform(t *testing.T) {
	t.Skip()

	var n uint32 = 5
	buckets := make([]int, n)
	thresh := -n % n
	for i := uint64(0); i <= 0xffffffff; i++ {
		if i&0x0fffffff == 0 {
			fmt.Printf("i is %x\n", i)
		}
		if uint32(i*uint64(n)) < thresh {
			continue
		}
		src := singleSource{i: uint32(i)}
		actual := UniformUint32(&src, n)
		buckets[actual]++
	}
	expectedCount := buckets[0]
	for i := uint32(1); i < n; i++ {
		if buckets[i] != expectedCount {
			t.Errorf("(i = %d) expected %d, got %d", i, expectedCount, buckets[i])
		}
	}
}

func uniformUint32Simple(src Source, n uint32) uint32 {
	thresh := uint32(-n) % uint32(n)
	for {
		v := src.Uint32()
		prod := uint64(v) * uint64(n)
		low := uint32(prod)
		if low >= thresh {
			return uint32(prod >> 32)
		}
	}
}

type descendingSource struct {
	i uint32
}

func (src *descendingSource) Uint32() uint32 {
	src.i--
	return src.i
}

func TestUniformUint32MatchesSimple(t *testing.T) {
	var src1, src2 descendingSource
	var n uint32 = 0xffffffff
	for i := 0; i < 1000; i++ {
		expected := uniformUint32Simple(&src1, n)
		actual := UniformUint32(&src2, n)
		if expected != actual {
			t.Fatalf("(i = %d) expected %d, got %d", i, expected, actual)
		}
		if src1.i != src2.i {
			t.Fatalf("(i = %d) expected %d, got %d", i, src1.i, src2.i)
		}
	}
}

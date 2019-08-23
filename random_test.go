package random

import (
	"fmt"
	"testing"
)

type singleSource struct {
	i      uint32
	called bool
}

func (src *singleSource) Uint32() uint32 {
	if src.called {
		panic("already called")
	}
	src.called = true
	return src.i
}

func computeRange(i, n uint32) (uint32, uint32) {
	thresh := uint32(-n) % uint32(n)
	firstProd := uint64(i)<<32 + uint64(thresh)
	lastProd := uint64(i+1)<<32 - 1
	firstV := uint32((firstProd + uint64(n) - 1) / uint64(n))
	lastV := uint32(lastProd / uint64(n))
	return firstV, lastV
}

func expectUniformUint32(t *testing.T, n, v, expected uint32) {
	t.Helper()
	src := singleSource{i: v}
	actual := UniformUint32(&src, n)
	if actual != expected {
		t.Errorf("(v=%d) expected %d, got %d", v, expected, actual)
	}
}

func TestUniformUint32Range(t *testing.T) {
	var n uint32 = 5
	for i := uint32(0); i < n; i++ {
		firstV, lastV := computeRange(i, n)
		expectUniformUint32(t, n, firstV, i)
		expectUniformUint32(t, n, lastV, i)
	}
}

func TestUniformUint32IsUniform(t *testing.T) {
	t.Skip()

	var n uint32 = 5
	buckets := make([]int, n)
	thresh := uint32(-n) % uint32(n)
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

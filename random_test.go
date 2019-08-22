package random

import (
	"testing"
)

func uniformUint32_simple(src Source, n uint32) uint32 {
	thresh := uint32(-n) % uint32(n)
	for {
		v := src.Uint32()
		prod := uint64(v) * uint64(n)
		low := uint32(prod)
		if low > thresh {
			return uint32(prod >> 32)
		}
	}
}

type sequentialSource struct {
	i uint32
}

func (src *sequentialSource) Uint32() uint32 {
	v := src.i
	src.i++
	return v
}

func TestUintformUint32MatchesSimple(t *testing.T) {
	var src1, src2 sequentialSource
	var n uint32 = 129
	for i := 0; i < 10000; i++ {
		expected := uniformUint32_simple(&src1, n)
		actual := UniformUint32(&src2, n)
		if expected != actual {
			t.Fatalf("(i = %d) expected %d, got %d", i, expected, actual)
		}
	}
}

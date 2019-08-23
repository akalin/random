package random

import (
	"fmt"
	"testing"
)

// How can we test UniformUint32? Fortunately, it's not too difficult -- since the randomness is encapsulated
// in src.Uint32(), we just have to make sure that as the return value of src.Uint32() spans the entire range
// from 0 to 0xffffffff, that UniformUint32(src, n) returns an equal number of values for 0, 1, …, n-1 when
// it doesn't reject the value of src.Uint32().
//
// Roughly, the return value of src.Uint32() vs. the return value of UniformUint32() (for n not a power of 2) looks like:
//
//    src.Uint32(): 0 1 2 3 …           …         …                     … fffffffe ffffffff
// UniformUint32(): X 0 0 0 … 0 X 1 1 1 … 1 X 2 2 … n-2 n-2 n-2 n-1 n-1 …      n-1      n-1
//
// where X means the value for src.Uint32() is rejected. The exact calculation is done by computeRange() below.
//
// Therefore, to test uniformity, we can test:
//
//   1) that the return values of computeRange(i, n) for 0 ≤ i < n partition the entire range from 0 to 0xffffffff;
//   2) that the valid ranges returned by computeRange(i, n) have the same size for 0 ≤ i < n;
//   3) that UniformUint32() returns i when in the valid range returned by computeRange(i, n), and rejects the value
//      otherwise.

// computeRange, given n > 0 and i < n, returns three uint32s
//
//   firstV ≤ firstValidV ≤ lastValidV
//
// such that:
//
//   - if firstV ≤ src.Uint32() < firstValidV, UniformUint32() rejects the sample, and
//   - if firstValidV ≤ src.Uint32() ≤ firstValidV, UniformUint32() returns i.
//
// TODO: Verify that firstValidV - firstV is at most 1.
//
// TODO: Refer to test below.
func computeRange(i, n uint32) (firstV uint32, firstValidV uint32, lastValidV uint32) {
	if n == 0 {
		panic("n must be non-zero")
	}
	if i >= n {
		panic("i must be < n")
	}
	thresh := -n % n
	firstProd := uint64(i) << 32
	firstValidProd := firstProd + uint64(thresh)
	lastValidProd := uint64(i+1)<<32 - 1
	firstV = uint32((firstProd + uint64(n-1)) / uint64(n))
	firstValidV = uint32((firstValidProd + uint64(n-1)) / uint64(n))
	lastValidV = uint32(lastValidProd / uint64(n))
	return firstV, firstValidV, lastValidV
}

func assertEqualUint32(t *testing.T, expected, actual uint32) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected %d, got %d", expected, actual)
	}
}

func assertLessEqualUint32(t *testing.T, a, b uint32) {
	t.Helper()
	if a > b {
		t.Fatalf("expected %d ≤ %d", a, b)
	}
}

// TestComputeRange tests that the values returned by computeRange partition the entire 32-bit range,
// and that the valid ranges for each i have the same number of values.
func TestComputeRange(t *testing.T) {
	var n uint32 = 5
	// From https://graphics.stanford.edu/~seander/bithacks.html#DetermineIfPowerOf2
	nIsPowerOfTwo := n&(n-1) == 0
	var prevLastValidV uint32
	var validRange uint32
	for i := uint32(0); i < n; i++ {
		firstV, firstValidV, lastValidV := computeRange(i, n)
		if nIsPowerOfTwo {
			assertEqualUint32(t, firstV, firstValidV)
		} else {
			assertLessEqualUint32(t, firstV, firstValidV)
		}
		assertLessEqualUint32(t, firstValidV, lastValidV)

		if i == 0 {
			assertEqualUint32(t, 0, firstV)

		} else {
			assertEqualUint32(t, prevLastValidV+1, firstV)
			assertEqualUint32(t, validRange, lastValidV-firstValidV+1)
		}

		if i == n-1 {
			assertEqualUint32(t, 0xffffffff, lastValidV)
		}
		prevLastValidV = lastValidV
		validRange = lastValidV - firstValidV + 1
	}
}

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
	for i := uint32(0); i < n; i++ {
		firstV, firstValidV, lastValidV := computeRange(i, n)

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

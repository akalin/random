package random

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// uniformUint returns a uniformly-distributed number in the range 0 to n-1 (inclusive). n must be non-zero, and
// must fit in numBits bits. numBits must be at least 1 and less than 32.
//
// This is a more general version of UniformUint32 for testing.
func uniformUint(src Source, n, numBits uint32) uint32 {
	if n == 0 {
		panic("n must be non-zero in call to UniformUint32")
	}

	if n >= 1<<numBits {
		panic("n must fit in numBits bits")
	}

	if numBits >= 32 {
		panic("numBits must be less than 32")
	}

	// Mask used to mask off all but the lower numBits bits
	// of v and low.
	mask := uint32(1)<<numBits - 1

	v := src.Uint32() & mask
	prod := uint64(v) * uint64(n)
	low := uint32(prod) & mask
	if low >= n {
		return uint32(prod >> numBits)
	}

	threshold := (1 << numBits) % n
	if low >= threshold {
		return uint32(prod >> numBits)
	}

	for {
		v = src.Uint32() & mask
		prod = uint64(v) * uint64(n)
		low := uint32(prod) & mask
		if low >= threshold {
			return uint32(prod >> numBits)
		}
	}
}

// singleSource is a source that returns only a single value, used for testing.
type singleSource struct {
	v         uint32
	callCount uint32
}

// Uint32() returns the stored value of v for the first call, then 0xffffffff for the second call
// (which should always be accepted), then panics on subsequent calls. Then callers can look at
// the call count to determine what value was actually used.
func (src *singleSource) Uint32() uint32 {
	if src.callCount == 0 {
		src.callCount++
		return src.v
	}

	if src.callCount == 1 {
		src.callCount++
		return 0xffffffff
	}

	panic("called when callCount > 1")
}

// testUniformUint loops through all numBits-bit values and checks to make sure that
// uniformUint() returns the values 0 to n-1 an equal number of times, filtering out
// the case where the first value is rejected.
func testUniformUint(t *testing.T, n, numBits uint32) {
	buckets := make([]int, n)
	for v := uint32(0); v < (1 << numBits); v++ {
		src := singleSource{v: v}
		u := uniformUint(&src, n, numBits)
		if src.callCount == 2 {
			// v was rejected, so continue.
			continue
		}
		require.Equal(t, uint32(1), src.callCount)
		require.GreaterOrEqual(t, u, 0)
		require.Less(t, u, n)
		buckets[u]++
	}
	expectedCount := int((1 << numBits) / n)
	for i := uint32(0); i < n; i++ {
		require.Equal(t, expectedCount, buckets[i], "i=%d", i)
	}
}

// TestUniformUint exhaustively tests small values for numBits, and all possible values of n for each
// value of numBits.
//
// We still have to test UniformUint32(), but this gives some confidence that the algorithm
// works in general.
func TestUniformUint(t *testing.T) {
	t.Parallel()
	for numBits := uint32(1); numBits < 10; numBits++ {
		t.Run(fmt.Sprintf("numBits=%d", numBits), func(t *testing.T) {
			t.Parallel()
			for n := uint32(1); n < 1<<numBits; n++ {
				testUniformUint(t, n, numBits)
			}
		})
	}
}

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
//
// TODO: Characterize range by three numbers, range size, threshold of X.

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

// testComputeRangeN tests that the values returned by computeRange partition the entire 32-bit range,
// and that the valid ranges for each i have the same number of values.
func testComputeRangeN(t *testing.T, n uint32) {
	// From https://graphics.stanford.edu/~seander/bithacks.html#DetermineIfPowerOf2
	nIsPowerOfTwo := n&(n-1) == 0
	var prevLastValidV uint32
	var validRange uint32
	for i := uint32(0); i < n; i++ {
		firstV, firstValidV, lastValidV := computeRange(i, n)
		if nIsPowerOfTwo {
			require.Equal(t, firstV, firstValidV)
		} else {
			// TODO: Figure out the threshold where the delta goes from 1 to 0.
			if firstV != firstValidV {
				require.LessOrEqual(t, firstV+1, firstValidV)
			}
		}
		require.LessOrEqual(t, firstValidV, lastValidV)

		if i == 0 {
			require.Equal(t, 0, firstV)

		} else {
			require.Equal(t, prevLastValidV+1, firstV)
			require.Equal(t, validRange, lastValidV-firstValidV+1)
		}

		if i == n-1 {
			require.Equal(t, 0xffffffff, lastValidV)
		}
		prevLastValidV = lastValidV
		validRange = lastValidV - firstValidV + 1
	}
}

func getTestNs(maxPower uint) []uint32 {
	var ns []uint32
	for i := uint(0); i < maxPower; i++ {
		n := uint32(1) << i
		if n >= 4 {
			ns = append(ns, n-1)
		}
		ns = append(ns, n)
		if n >= 4 {
			ns = append(ns, n+1)
		}
	}
	return ns
}

func TestComputeRange(t *testing.T) {
	t.Parallel()
	for _, n := range getTestNs(15) {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			t.Parallel()
			testComputeRangeN(t, n)
		})
	}
}

func expectUniformUint32Returns(t *testing.T, n, v, expected uint32) {
	t.Helper()
	src := singleSource{v: v}
	actual := UniformUint32(&src, n)
	if actual != expected {
		t.Errorf("(v=%d) expected %d, got %d", v, expected, actual)
	}
}

func expectUniformUint32Rejects(t *testing.T, n, v uint32) {
	t.Helper()
	src := singleSource{v: v}
	actual := UniformUint32(&src, n)
	if src.callCount != 2 {
		t.Errorf("(v=%d) expected %d, got %d (actual=%d)", v, 2, src.callCount, actual)
	}
}

func testUniformUint32Range(t *testing.T, n uint32) {
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

func TestUniformUint32Range(t *testing.T) {
	t.Parallel()
	for _, n := range getTestNs(8) {
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			t.Parallel()
			testUniformUint32Range(t, n)
		})
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
		src := singleSource{v: uint32(i)}
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

// TODO: Benchmarks.

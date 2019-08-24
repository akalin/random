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
	buckets := make([]uint32, n)
	for v := uint32(0); v < (1 << numBits); v++ {
		src := singleSource{v: v}
		u := uniformUint(&src, n, numBits)
		if src.callCount == 2 {
			// v was rejected, so continue.
			continue
		}
		require.Equal(t, uint32(1), src.callCount)
		require.Less(t, u, n)
		buckets[u]++
	}
	expectedCount := (1 << numBits) / n
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
		numBits := numBits // capture range variable.
		t.Run(fmt.Sprintf("numBits=%d", numBits), func(t *testing.T) {
			t.Parallel()
			for n := uint32(1); n < 1<<numBits; n++ {
				testUniformUint(t, n, numBits)
			}
		})
	}
}

func testUniformUint32(t *testing.T, n uint32) {
	two32 := uint64(1) << 32
	// count and vStart can be two32, so they both have to be uint64.
	count := two32 / uint64(n)
	vStart := uint64(0)
	for i := uint32(0); i < n; i++ {
		// Test vStart.
		src := singleSource{v: uint32(vStart)}
		u := UniformUint32(&src, n)
		if src.callCount == 2 {
			// vStart was rejected, so the actual vStart must be one higher.
			vStart++
			src = singleSource{v: uint32(vStart)}
			u = UniformUint32(&src, n)
		}
		require.Equal(t, uint32(1), src.callCount)
		require.Equal(t, i, u)

		vEnd := vStart + uint64(count)

		// Test last v.
		src = singleSource{v: uint32(vEnd - 1)}
		u = UniformUint32(&src, n)
		require.Equal(t, uint32(1), src.callCount)
		require.Equal(t, i, u)

		// Test a middle v.
		src = singleSource{v: uint32(vStart) + uint32(count/2)}
		u = UniformUint32(&src, n)
		require.Equal(t, uint32(1), src.callCount)
		require.Equal(t, i, u)

		vStart = vEnd
	}

	require.Equal(t, vStart, two32)
}

func TestUniformUint32(t *testing.T) {
	t.Parallel()
	var ns []uint32
	for i := uint32(0); i < 15; i++ {
		n := uint32(1) << i
		if i >= 2 {
			ns = append(ns, n-1)
		}
		ns = append(ns, n)
		if i >= 2 {
			ns = append(ns, n+1)
		}
	}
	for _, n := range ns {
		testUniformUint32(t, n)
	}
}

// TODO: Benchmarks.

package random

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

// uniformUint returns a uniformly-distributed number in the range 0 to n-1 (inclusive). n must be non-zero, and
// must fit in numBits bits. numBits must be at least 1 and less than 32.
//
// This is a more general and simplified version of UniformUint32 for testing.
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

	threshold := (1 << numBits) % n
	for {
		v := uint32(src.Int63()>>31) & mask
		prod := uint64(v) * uint64(n)
		low := uint32(prod) & mask
		if low >= threshold {
			return uint32(prod >> numBits)
		}
	}
}

// testSource is a source that returns a series of values for testing.
type testSource struct {
	vs        []uint32
	callCount uint32
}

// Uint32() returns the next value in vs, or panics if there aren't any left.
func (src *testSource) Int63() int64 {
	if src.callCount >= uint32(len(src.vs)) {
		panic("ran out of vs to return")
	}

	i := src.callCount
	src.callCount++
	return int64(src.vs[i])
}

func makeSingleSource(v uint32) testSource {
	return testSource{vs: []uint32{v, 0xffffffff}}
}

func makeDoubleSource(v uint32) testSource {
	return testSource{vs: []uint32{0x0, v, 0xffffffff}}
}

// testUniformUint loops through all numBits-bit values and checks to make sure that
// uniformUint() returns the values 0 to n-1 an equal number of times, filtering out
// the case where the first value is rejected.
func testUniformUint(t *testing.T, n, numBits uint32) {
	buckets := make([]uint32, n)
	for v := uint32(0); v < (1 << numBits); v++ {
		src := makeSingleSource(v)
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

func computeVStart(i, n uint32) uint64 {
	return (uint64(i)<<32 + uint64(n-1)) / uint64(n)
}

func testVStart(t *testing.T, i, n, vStart uint32) uint32 {
	// Test vStart.
	src := makeSingleSource(vStart)
	u := UniformUint32(&src, n)
	if src.callCount == 2 {
		// vStart was rejected, so the actual vStart must be one higher.
		vStart++
		src = makeSingleSource(vStart)
		u = UniformUint32(&src, n)
	}
	require.Equal(t, uint32(1), src.callCount)
	require.Equal(t, i, u)
	return vStart
}

func testV(t *testing.T, i, n, v uint32) {
	src := makeSingleSource(v)
	u := UniformUint32(&src, n)
	require.Equal(t, uint32(1), src.callCount)
	require.Equal(t, i, u)
}

func testUniformUint32(t *testing.T, n, delta uint32) {
	two32 := uint64(1) << 32
	// count and vEnd can be two32, so they both have to be uint64.
	count := two32 / uint64(n)
	var vEnd uint64
	for i := uint32(0); i < n; {
		vStart := uint32(computeVStart(i, n))
		vEnd = computeVStart(i+1, n)

		vStart = testVStart(t, i, n, vStart)

		// Test interval size.
		require.Less(t, uint64(vStart), vEnd)
		require.Equal(t, count, vEnd-uint64(vStart))

		// Test last v.
		testV(t, i, n, uint32(vEnd-1))

		// Test a middle v.
		testV(t, i, n, uint32(vStart)+uint32(count/2))

		if i == n-1 {
			break
		} else if (n - i) <= delta {
			i = n - 1
		} else {
			i += delta
		}
	}

	require.Equal(t, vEnd, two32)
}

func testVStartDouble(t *testing.T, i, n, vStart uint32) uint32 {
	// Test vStart.
	src := makeDoubleSource(vStart)
	u := UniformUint32(&src, n)
	if src.callCount == 3 {
		// vStart was rejected, so the actual vStart must be one higher.
		vStart++
		src = makeDoubleSource(vStart)
		u = UniformUint32(&src, n)
	}
	require.Equal(t, uint32(2), src.callCount)
	require.Equal(t, i, u)
	return vStart
}

func testVDouble(t *testing.T, i, n, v uint32) {
	src := makeDoubleSource(v)
	u := UniformUint32(&src, n)
	require.Equal(t, uint32(2), src.callCount)
	require.Equal(t, i, u)
}

func testUniformUint32Double(t *testing.T, n, delta uint32) {
	two32 := uint64(1) << 32
	// count and vEnd can be two32, so they both have to be uint64.
	count := two32 / uint64(n)
	var vEnd uint64
	for i := uint32(0); i < n; {
		vStart := uint32(computeVStart(i, n))
		vEnd = computeVStart(i+1, n)

		vStart = testVStartDouble(t, i, n, vStart)

		// Test interval size.
		require.Less(t, uint64(vStart), vEnd)
		require.Equal(t, count, vEnd-uint64(vStart))

		// Test last v.
		testVDouble(t, i, n, uint32(vEnd-1))

		// Test a middle v.
		testVDouble(t, i, n, uint32(vStart)+uint32(count/2))

		if i == n-1 {
			break
		} else if (n - i) <= delta {
			i = n - 1
		} else {
			i += delta
		}
	}

	require.Equal(t, vEnd, two32)
}

func TestUniformUint32SmallPowersOfTwo(t *testing.T) {
	t.Parallel()
	var ns []uint32
	for i := uint32(0); i < 15; i++ {
		ns = append(ns, uint32(1)<<i)
	}
	for _, n := range ns {
		testUniformUint32(t, n, 1)
	}
}

func TestUniformUint32LargePowersOfTwo(t *testing.T) {
	t.Parallel()
	var ns []uint32
	for i := uint32(15); i < 32; i++ {
		ns = append(ns, uint32(1)<<i)
	}
	for _, n := range ns {
		testUniformUint32(t, n, n/1000)
	}
}

func TestUniformUint32Small(t *testing.T) {
	t.Parallel()
	var ns []uint32
	for i := uint32(2); i < 15; i++ {
		n := uint32(1) << i
		ns = append(ns, n-1)
		ns = append(ns, n+1)
	}
	for _, n := range ns {
		testUniformUint32(t, n, 1)
		testUniformUint32Double(t, n, 1)
	}
}

func TestUniformUint32Medium(t *testing.T) {
	t.Parallel()
	var ns []uint32
	for i := uint32(15); i < 32; i++ {
		n := uint32(1) << i
		ns = append(ns, n-1)
		ns = append(ns, n+1)
	}
	for _, n := range ns {
		testUniformUint32(t, n, n/1000)
		testUniformUint32Double(t, n, n/1000)
	}
}

func TestUniformUint32Large(t *testing.T) {
	t.Parallel()
	var ns []uint32
	for i := uint32(0); i < 100; i++ {
		ns = append(ns, 0xffffffff-i)
	}
	for _, n := range ns {
		testUniformUint32(t, n, n/1000)
		testUniformUint32Double(t, n, n/1000)
	}
}

func shuffleUniformUint32(src Source, n int, swap func(i, j int)) {
	if n < 0 {
		panic("invalid argument to Shuffle")
	}

	i := n - 1
	if i > 1<<31-1-1 {
		panic("n too big")
	}

	for ; i > 0; i-- {
		j := int(UniformUint32(src, uint32(i+1)))
		swap(i, j)
	}
}

func shuffleRandInt31n(r *rand.Rand, n int, swap func(i, j int)) {
	if n < 0 {
		panic("invalid argument to Shuffle")
	}

	i := n - 1
	if i > 1<<31-1-1 {
		panic("n too big")
	}

	for ; i > 0; i-- {
		j := int(r.Int31n(int32(i + 1)))
		swap(i, j)
	}
}

var largeUniformUint32Result int

func BenchmarkLargeShuffleUniformUint32(b *testing.B) {
	src := rand.NewSource(4)
	swap := func(i, j int) {
		largeUniformUint32Result += i + j
	}
	for n := 0; n < b.N; n++ {
		shuffleUniformUint32(src, 0x0fffffff, swap)
	}
}

var largeInt31nResult int

func BenchmarkLargeShuffleRandInt31n(b *testing.B) {
	r := rand.New(rand.NewSource(4))
	swap := func(i, j int) {
		largeInt31nResult += i + j
	}
	for n := 0; n < b.N; n++ {
		shuffleRandInt31n(r, 0x0fffffff, swap)
	}
}

var largeRandShuffleResult int

func BenchmarkLargeRandShuffle(b *testing.B) {
	r := rand.New(rand.NewSource(4))
	swap := func(i, j int) {
		largeInt31nResult += i + j
	}
	for n := 0; n < b.N; n++ {
		r.Shuffle(0x0fffffff, swap)
	}
}

var smallUniformUint32Result int

func BenchmarkSmallShuffleUniformUint32(b *testing.B) {
	src := rand.NewSource(5)
	swap := func(i, j int) {
		smallUniformUint32Result += i + j
	}
	for n := 0; n < b.N; n++ {
		shuffleUniformUint32(src, 0xffff, swap)
	}
}

var smallInt31nResult int

func BenchmarkSmallShuffleRandInt31n(b *testing.B) {
	r := rand.New(rand.NewSource(5))
	swap := func(i, j int) {
		smallInt31nResult += i + j
	}
	for n := 0; n < b.N; n++ {
		shuffleRandInt31n(r, 0xffff, swap)
	}
}

var smallRandShuffleResult int

func BenchmarkSmallRandShuffle(b *testing.B) {
	r := rand.New(rand.NewSource(5))
	swap := func(i, j int) {
		smallRandShuffleResult += i + j
	}
	for n := 0; n < b.N; n++ {
		r.Shuffle(0xffff, swap)
	}
}

func fakeAllRangesShuffleUniformUint32(src Source) uint32 {
	var sum uint32
	for bit := uint32(1); int32(bit) > 0; bit <<= 1 {
		for i := uint32(0); i < 0x1000000; i++ {
			bound := bit | (i & (bit - 1))
			sum += UniformUint32(src, bound)
		}
	}
	return sum
}

func fakeAllRangesShuffleRand(r *rand.Rand) int32 {
	var sum int32
	for bit := int32(1); bit > 0; bit <<= 1 {
		for i := int32(0); i < 0x1000000; i++ {
			bound := bit | (i & (bit - 1))
			sum += r.Int31n(bound)
		}
	}
	return sum
}

var allUniformResult uint32

func BenchmarkAllRangesShuffleUniformUint32(b *testing.B) {
	src := rand.NewSource(6)
	for n := 0; n < b.N; n++ {
		allUniformResult += fakeAllRangesShuffleUniformUint32(src)
	}
}

var allRandResult int32

func BenchmarkAllRangesShuffleRand(b *testing.B) {
	r := rand.New(rand.NewSource(6))
	for n := 0; n < b.N; n++ {
		allRandResult += fakeAllRangesShuffleRand(r)
	}
}

package random

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

// uintn returns a uniformly-distributed number in the range 0 to n-1 (inclusive). n must be non-zero, and
// must fit in numBits bits. numBits must be at least 1 and less than 32.
//
// This is a more general and simplified version of Uint32n for testing.
func uintn(src Source, n, numBits uint32) uint32 {
	if n == 0 {
		panic("n must be non-zero in call to Uint32n")
	}

	if n >= 1<<numBits {
		panic("n must fit in numBits bits")
	}

	if numBits >= 32 {
		panic("numBits must be less than 32")
	}

	// Mask used to mask off all but the lower numBits bits of v and low.
	mask := uint32(1)<<numBits - 1

	threshold := (1 << numBits) % n
	for {
		v := randUint32(src) & mask
		prod := uint64(v) * uint64(n)
		low := uint32(prod) & mask
		if low >= threshold {
			return uint32(prod >> numBits)
		}
	}
}

// testSource is a source that returns a series of uint32 values for testing.
type testSource struct {
	vs        []uint32
	callCount int
}

// Int63() returns the next value in src.vs shifted up appropriately, or panics if there aren't any left.
func (src *testSource) Int63() int64 {
	if src.callCount >= len(src.vs) {
		panic("ran out of vs to return")
	}

	i := src.callCount
	src.callCount++
	// Uint32n() uses the top 32 bits.
	return int64(src.vs[i]) << 31
}

// makeTestSource returns a test source that returns a value that'll be rejected by uint32n or uintn
// rejectionCount times (assuming that the value of n isn't a power of two), then returns the given value,
// then returns a value that will always be accepted. Then src.callCount can be checked to see what
// actually happened.
func makeTestSource(rejectionCount int, v uint32) testSource {
	// The first group is always a big group, so 0 will always be rejected if
	// n isn't a power of two.
	vs := make([]uint32, rejectionCount)
	// The last group is always a small group, so 0xffffffff will always be accepted.
	return testSource{vs: append(vs, []uint32{v, 0xffffffff}...)}
}

// testUniformUint loops through all numBits-bit values and checks to make sure that
// uintn() returns the values 0 to n-1 an equal number of times, filtering out
// the case where the first value is rejected.
func testUniformUint(t *testing.T, n, numBits uint32) {
	buckets := make([]uint32, n)
	for v := uint32(0); v < (1 << numBits); v++ {
		src := makeTestSource(0, v)
		u := uintn(&src, n, numBits)
		if src.callCount == 2 {
			// v was rejected, so continue.
			continue
		}
		require.Equal(t, 1, src.callCount)
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
// We still have to test Uint32n(), but this gives some confidence that the algorithm
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

// It's infeasible to test Uint32n() exhaustively, so we need to think of something faster. Uint32n() is
// monotonic with respect to v, meaning that for each possible return value of Uint32n(), there is a range
// of v that would always return that value (except for maybe a single rejected value).
// Therefore, we can check the behavior of Uint32n() at the boundaries of these ranges,
// and also we can verify that the return value of Uint32n() doesn't change within a range.

// computeVStart computes the start of the range for v that would make Uint32n(src, n) return i,
// except that the first value in the range can possibly be rejected if n is not a power of two.
// The end of the range is simply computeVStart(i+1, n).
func computeVStart(i, n uint32) uint64 {
	// Compute ceil((i*2³²)/n).
	// Recall that ceil(a/b) == floor((a + (b - 1))/b).
	return (uint64(i)<<32 + uint64(n-1)) / uint64(n)
}

// Test the boundary behaviors of computeVStart for various values of n.
func TestComputeVStart(t *testing.T) {
	ns := []uint32{1, 2}
	for i := uint32(2); i < 32; i++ {
		n := uint32(1) << i
		ns = append(ns, []uint32{n - 1, n, n + 1, uint32(3 * uint64(n) / 2)}...)
	}
	for _, n := range ns {
		require.Equal(t, uint64(0), computeVStart(0, n))
		require.Equal(t, uint64(0x100000000), computeVStart(n, n))
	}
}

// testVStart checks that the given value of vStart (or the one after it, if n isn't a power of two)
// does indeed make Uint32n(src, n) return i. It then returns the actual value of vStart.
func testVStart(t *testing.T, rejectionCount int, i, n, vStart uint32) uint32 {
	src := makeTestSource(rejectionCount, vStart)
	u := Uint32n(&src, n)
	if n&(n-1) != 0 && src.callCount == rejectionCount+2 {
		// n is not a power of two and vStart was rejected, so the actual vStart must be one higher.
		vStart++
		src = makeTestSource(rejectionCount, vStart)
		u = Uint32n(&src, n)
	}
	require.Equal(t, rejectionCount+1, src.callCount)
	require.Equal(t, i, u)
	return vStart
}

// testV checks that the given value of v does indeed make Uint32n(src, n) return i.
func testV(t *testing.T, rejectionCount int, i, n, v uint32) {
	src := makeTestSource(rejectionCount, v)
	u := Uint32n(&src, n)
	require.Equal(t, rejectionCount+1, src.callCount)
	require.Equal(t, i, u)
}

// testUint32ni computes the v range for the given value of i and n and tests that
// the start and end of that range give i, and also vPoints number of points
// in the middle of the range.
func testUint32ni(t *testing.T, rejectionCount int, i, n, vPoints uint32) {
	vStart := computeVStart(uint32(i), n)
	vEnd := computeVStart(uint32(i+1), n)

	vStart = uint64(testVStart(t, rejectionCount, uint32(i), n, uint32(vStart)))

	expectedCount := 0x100000000 / uint64(n)
	count := vEnd - vStart
	require.Equal(t, expectedCount, count)

	vDelta := (count + uint64(vPoints) - 1) / uint64(vPoints)

	for v := vStart + uint64(vDelta); v < vEnd; {
		testV(t, rejectionCount, uint32(i), n, uint32(v))

		if v == vEnd-1 {
			break
		}

		v += uint64(vDelta)
		if v >= vEnd {
			v = vEnd - 1
		}
	}
}

// testUint32n calls testUint32ni for 0 up to n-1, going up by nDelta.
func testUint32n(t *testing.T, rejectionCount int, n, nDelta, vPoints uint32) {
	for i := uint64(0); i < uint64(n); {
		testUint32ni(t, rejectionCount, uint32(i), n, vPoints)

		if i == uint64(n-1) {
			break
		}

		i += uint64(nDelta)
		if i >= uint64(n) {
			i = uint64(n - 1)
		}
	}
}

// TestUint32n*PowersOfTwo calls testUint32n for n = all powers of two. Since no values of v will
// be rejected for such values of n, we can always pass in 0 for rejectionCount.

func TestUint32nSmallPowersOfTwo(t *testing.T) {
	t.Parallel()
	for i := uint32(0); i < 10; i++ {
		n := uint32(1) << i
		testUint32n(t, 0, n, 1, 100)
	}
}

func TestUint32nMediumPowersOfTwo(t *testing.T) {
	t.Parallel()
	for i := uint32(10); i < 17; i++ {
		n := uint32(1) << i
		testUint32n(t, 0, n, n>>9, 100)
	}
}

func TestUint32nLargePowersOfTwo(t *testing.T) {
	t.Parallel()
	for i := uint32(17); i < 32; i++ {
		n := uint32(1) << i
		testUint32n(t, 0, n, n>>9, 100)
	}
}

// TestUint32n*CloseToPowerOfTwo calls testUint32n for n = all powers of two ±1. Since some values of v will
// be rejected for such values of n, we test with various values of rejectionCount to cover the fast and slow
// path of Uint32n.

func TestUint32nSmallCloseToPowerOfTwo(t *testing.T) {
	t.Parallel()
	for i := uint32(2); i < 10; i++ {
		n := uint32(1) << i
		for r := 0; r < 3; r++ {
			testUint32n(t, r, n-1, 1, 100)
			testUint32n(t, r, n+1, 1, 100)
		}
	}
}

func TestUint32nMediumCloseToPowerOfTwo(t *testing.T) {
	t.Parallel()
	for i := uint32(10); i < 17; i++ {
		n := uint32(1) << i
		for r := 0; r < 2; r++ {
			testUint32n(t, r, n-1, n>>9, 50)
			testUint32n(t, r, n+1, n>>9, 50)
		}
	}
}

func TestUint32nLargeCloseToPowerOfTwo(t *testing.T) {
	t.Parallel()
	for i := uint32(17); i < 32; i++ {
		n := uint32(1) << i
		for r := 0; r < 2; r++ {
			testUint32n(t, r, n-1, n>>9, 30)
			testUint32n(t, r, n+1, n>>9, 30)
		}
	}
}

// TestUint32n*BetweenPowersOfTwo calls testUint32n for n ≈ (3/2) * all powers of two.

func TestUint32nSmallBetweenPowersOfTwo(t *testing.T) {
	t.Parallel()
	for i := uint32(2); i < 10; i++ {
		n := 3 * (uint32(1) << i) / 2
		for r := 0; r < 3; r++ {
			testUint32n(t, r, n, 1, 100)
		}
	}
}

func TestUint32nMediumBetweenPowersOfTwo(t *testing.T) {
	t.Parallel()
	for i := uint32(10); i < 17; i++ {
		n := 3 * (uint32(1) << i) / 2
		for r := 0; r < 2; r++ {
			testUint32n(t, r, n, n>>9, 90)
		}
	}
}

func TestUint32nLargeBetweenPowersOfTwo(t *testing.T) {
	t.Parallel()
	for i := uint32(17); i < 32; i++ {
		n := uint32(3 * (uint64(1) << i) / 2)
		for r := 0; r < 2; r++ {
			testUint32n(t, r, n, n>>9, 80)
		}
	}
}

// TestUint32nCloseToMax calls testUint32n for n close to 2³².
func TestUint32nCloseToMax(t *testing.T) {
	t.Parallel()
	for i := uint32(0); i < 150; i++ {
		n := 0xffffffff - i
		for r := 0; r < 2; r++ {
			testUint32n(t, r, n, n>>9, 80)
		}
	}
}

// Benchmarks
// ----------

// randInt63n is a copy of rand.Int63n() that is called by shuffleUint32n() and shuffleRandInt31n().
// It's actually never executed; we just have this here so that the shuffle functions are as close
// as possible to rand.Shuffle().
func randInt63n(src Source, n int64) int64 {
	if n <= 0 {
		panic("invalid argument to Int63n")
	}
	if n&(n-1) == 0 { // n is power of two, can mask
		return src.Int63() & (n - 1)
	}
	max := int64((1 << 63) - 1 - (1<<63)%uint64(n))
	v := src.Int63()
	for v > max {
		v = src.Int63()
	}
	return v % n
}

// shuffleUint32n is a copy of rand.Shuffle() that uses Uint32n() instead of rand.int31n().
func shuffleUint32n(src Source, n int, swap func(i, j int)) {
	if n < 0 {
		panic("invalid argument to shuffleUint32n")
	}

	i := n - 1
	for ; i > 1<<31-1-1; i-- {
		j := int(randInt63n(src, int64(i+1)))
		swap(i, j)
	}
	for ; i > 0; i-- {
		j := int(Uint32n(src, uint32(i+1)))
		swap(i, j)
	}
}

// randInt31 turns the output of src.Int63() into a uniformly-distributed pseudo-random int32 value in the range
// 0 to 2³¹-1 (inclusive).
func randInt31(src Source) int32 {
	// Take the top 31 bits, copying rand.Int31() from https://golang.org/src/math/rand/rand.go .
	return int32(src.Int63() >> 32)
}

// randInt31n is a copy of rand.Int31n() that is called by shuffleRandInt31n. This is because having
// shuffleRandInt31n call rand.Int31n() slows it down a bit (probably because it's not inlined),
// and we want to show that the fastest that shuffleRandInt31n can be is still slower than shuffleUint32n.
func randInt31n(src Source, n int32) int32 {
	if n <= 0 {
		panic("invalid argument to Int31n")
	}
	if n&(n-1) == 0 { // n is power of two, can mask
		return randInt31(src) & (n - 1)
	}
	max := int32((1 << 31) - 1 - (1<<31)%uint32(n))
	v := randInt31(src)
	for v > max {
		v = randInt31(src)
	}
	return v % n
}

// shuffleRandInt31n is a copy of rand.Shuffle() that uses randInt31n() instead of rand.int31n().
func shuffleRandInt31n(src Source, n int, swap func(i, j int)) {
	if n < 0 {
		panic("invalid argument to shuffleRandInt31n")
	}

	i := n - 1
	for ; i > 1<<31-1-1; i-- {
		j := int(randInt63n(src, int64(i+1)))
		swap(i, j)
	}
	for ; i > 0; i-- {
		j := int(randInt31n(src, int32(i+1)))
		swap(i, j)
	}
}

// The BenchmarkLargeShuffle* (Small) functions benchmark a shuffle using Uint32n or randInt31n against
// rand.Shuffle(), with a large (small) n and a no-op swap function.
//
// In my runs, shuffleUint32n() very slightly beats out rand.Shuffle(), probably because of better inlining,
// and both beat out shuffleRandInt31n().

const largeN = 0x0fffffff
const smallN = 0x0000ffff

// This variable (and the similar ones below) are to prevent the compiler from optimizing the benchmarks out.
var largeUint32nResult int

func BenchmarkLargeShuffleUint32n(b *testing.B) {
	src := rand.NewSource(4)
	swap := func(i, j int) {
		largeUint32nResult += i + j
	}
	for n := 0; n < b.N; n++ {
		shuffleUint32n(src, largeN, swap)
	}
}

var largeInt31nResult int

func BenchmarkLargeShuffleRandInt31n(b *testing.B) {
	src := rand.NewSource(4)
	swap := func(i, j int) {
		largeInt31nResult += i + j
	}
	for n := 0; n < b.N; n++ {
		shuffleRandInt31n(src, largeN, swap)
	}
}

var largeRandShuffleResult int

func BenchmarkLargeRandShuffle(b *testing.B) {
	r := rand.New(rand.NewSource(4))
	swap := func(i, j int) {
		largeInt31nResult += i + j
	}
	for n := 0; n < b.N; n++ {
		r.Shuffle(largeN, swap)
	}
}

var smallUint32nResult int

func BenchmarkSmallShuffleUint32n(b *testing.B) {
	src := rand.NewSource(5)
	swap := func(i, j int) {
		smallUint32nResult += i + j
	}
	for n := 0; n < b.N; n++ {
		shuffleUint32n(src, smallN, swap)
	}
}

var smallInt31nResult int

func BenchmarkSmallShuffleRandInt31n(b *testing.B) {
	src := rand.NewSource(5)
	swap := func(i, j int) {
		smallInt31nResult += i + j
	}
	for n := 0; n < b.N; n++ {
		shuffleRandInt31n(src, smallN, swap)
	}
}

var smallRandShuffleResult int

func BenchmarkSmallRandShuffle(b *testing.B) {
	r := rand.New(rand.NewSource(5))
	swap := func(i, j int) {
		smallRandShuffleResult += i + j
	}
	for n := 0; n < b.N; n++ {
		r.Shuffle(smallN, swap)
	}
}

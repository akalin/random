package random

// TODO: Add comment.
type Source interface {
	Uint32() uint32
}

// TODO: Add comment.
func UniformUint32(src Source, n uint32) uint32 {
	// To understand the algorithm below, let's pretend we're working with 3-bit and 6-bit integers
	// instead of 32-bit and 64-bit integers,
	// so src.Uint3() returns a number from 0 to 2³-1=7, and consider n=3. Then the possible values of:
	//
	//   v,
	//   prod = uint6(v)*uint6(n),
	//   high = prod >> 3,
	//   low  = uint3(prod), and
	//   2³-low
	//
	// are:
	//
	//      v:  0  1  2  | 3  4  5 |  6  7
	//   prod:  0  3  6  | 9 12 15 | 18 21
	//   high:  0  0  0  | 1  1  1 |  2  2
	//    low:  0  3  6  | 1  4  7 |  2  5
	// 2³-low:  8  5  2  | 7  4  1 |  6  3
	//
	// where we're grouping the table by values of high, which takes values from 0 to n-1.
	//
	// Note that each group has either 2 or 3 values, which means that if we simply returned high,
	// we wouldn't get a uniform distribution from 0 to n-1. This is because n doesn't evenly divide 2³,
	// so floor(2³ / n) = 2 < ceil(2³ / n) = 3. What we want to do is reject some values of v so that
	// each group has exactly floor(2³ / n) values. But how?
	//
	// Note that the first group has 3 entries and the last group has 2 entries. This is a general phenomenon:
	// the first group will *always* have ceil(2³ / n) entries, and the last group will *always* have
	// floor(2³ / n) entries.
	//
	// Why? Because the first group contains prod=0,
	// so low starts at 0 and counts up by 3 as you go left to right in the first group.
	// On the other hand, if we extended the table to contain v=8, it would have high=8...
	//
	// low=8 and 2³-low=0,
	// so 2³-low starts at 0 and counts up by 3 as you go from right to left in the last group.
	// But v=8 isn't a possible value, so the last group must contain one fewer entry than the first group.
	//
	// How does that help us? Because the first value for low in last group is the smallest value of low such that
	// a group can have floor(2³ / n) entries. Why?
	//
	// Compute first value of low for the last group, and show that it's b % n.
	//
	// So if we look at each group and reject each column such that low < b % n, all groups would have
	// floor(2³ / n) entries, and so we'd have a uniform distribution for high.
	//
	// Now the same logic holds if we work with 32-bit integers; there was nothing special about 2³. Therefore,
	// we have the algorithm:
	//
	// while True {
	//   v := src.Uint32()
	//   prod := v*n
	//   high := prod >> 32
	//   low  := uint32(prod)
	//   if low >= 2³² % n {
	//     return high
	//   }
	// }

	// Blurb and links about Lemire's algorithm. Mention that the
	// complexity is to avoid divisions, which include modulo operations.

	// Explain how this is an unrolled first iteration of the loop
	// below, with one optimization.
	v := src.Uint32()
	prod := uint64(v) * uint64(n)
	low := uint32(prod)
	// This is to avoid calculating thresh when not necessary.
	if low >= n {
		return uint32(prod >> 32)
	}

	// Explain how this is really to calculate 2³² % n.
	thresh := uint32(-n) % uint32(n)
	if low >= thresh {
		return uint32(prod >> 32)
	}

	// Slow path. If we remove all the code above, should have
	// exactly the same behavior.
	for {
		v = src.Uint32()
		prod = uint64(v) * uint64(n)
		low = uint32(prod)
		if low >= thresh {
			return uint32(prod >> 32)
		}
	}
}

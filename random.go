package random

// A Source represents a source of uniformly-distributed pseudo-random uint32 values in the range 0 to 2³²-1 (inclusive).
type Source interface {
	Uint32() uint32
}

// UniformUint32 returns a uniformly-distributed number in the range 0 to n-1 (inclusive). n must be non-zero.
func UniformUint32(src Source, n uint32) uint32 {
	if n == 0 {
		panic("n must be non-zero in call to UniformUint32")
	}

	// The algorithm below is taken from Lemire's "Fast Random Integer Generation in an Interval", available at
	// https://arxiv.org/abs/1805.10941 . See also
	// https://lemire.me/blog/2019/06/06/nearly-divisionless-random-integer-generation-on-various-systems/ and
	// http://www.pcg-random.org/posts/bounded-rands.html for more details beyond the following explanation.
	//
	// To understand the algorithm below, let's pretend we're working with 3-bit and 6-bit integers
	// instead of 32-bit and 64-bit integers, so instead of src.Uint32() we have src.Uint3(), which returns
	// a number from 0 to 2³-1=7, and the restriction n ≤ 7 is imposed. With n=3, the possible values of:
	//
	//   v,
	//   prod = v*n,
	//   high = floor(prod / 2³),
	//   low  = prod % 2³, and
	//
	// are:
	//
	//      v:  0  1  2  | 3  4  5 |  6  7
	//   prod:  0  3  6  | 9 12 15 | 18 21
	//   high:  0  0  0  | 1  1  1 |  2  2
	//    low:  0  3  6  | 1  4  7 |  2  5
	//
	// where we're grouping the table by values of high, which takes values from 0 to n-1.
	//
	// Note that because n doesn't evenly divide 2³, each group has either floor(2³/n) = 2 or ceil(2³/n) = 3 values,
	// which means that if we simply returned high, we wouldn't get a uniform distribution from 0 to n-1.
	// What we want to do is reject some values of v so that each group has exactly floor(2³/n) values. But how?
	//
	// We want to use the value of low to decide which values of v to reject. In this case,
	// if we reject all values of v where low < 2, then we'd throw out v=0 and v=3, so each group would have
	// exactly 2 values.
	//
	// Now the question is how we decide what threshold to set for low. The first fact we use is that the group
	// with v=0 contains ceil(2³/n) entries. This is because that group contains exactly the values of v such that
	// 0 ≤ v*n < 2³.
	//
	// The second fact we use is that the leftmost entry in a group has 0 ≤ low < n. This is because if an entry in a group
	// has low > n, then moving left decreases the value of prod and low by n but leaves high the same.
	//
	// In general, the values of low for a single group go up by n as you go right. So consider the first group in
	// our example above:
	//
	//  0 3 6
	//
	// If we increment the leftmost value by 1, we must increment all the other values by 1:
	//
	//  1 4 7
	//
	// If we do it again, we get:
	//
	//  2 5 (8)
	//
	// where the last value is ≥ 2³, so it actually belongs to the next group. So the smallest value of low
	// where we get a "small" group is 2³ minus 6, the rightmost value of low of the first group.
	//
	// Now we can derive a general formula for the threshold value. The rightmost value of low of the first group
	// is the largest multiple of n < 2³, which is
	//
	//   2³ - (2³ % n).
	//
	// Therefore, the threshold value is
	//
	//   2³ - (2³ - (2³ % n)) = 2³ % n.
	//
	// That is, if we filter out values of v where low < 2³ % n, then we remove a single entry from each "big" group,
	// turning into a "small" group. Then all groups would have the same size, and we'd have a uniform distribution.
	//
	// Now the same logic holds if we work with 32-bit integers; we just replace 2³ with 2³² everywhere. Now recall that
	//
	//    floor(prod / 2³²) = prod >> 32
	//
	// and
	//
	//    prod % 2³² = uint32(prod).
	//
	// Then we have the algorithm (in pseudocode):
	//
	//   thresh := 2³² % n
	//   while True {
	//     v := src.Uint32()
	//     prod := uint64(v)*uint64(n)
	//     high := prod >> 32
	//     low  := uint32(prod)
	//     if low >= thresh {
	//       return high
	//     }
	//   }
	//
	// Now the question is: why go all through this trouble in the first place? Because we want to avoid
	// division or remainder operations, which are the most expensive. The above algorithm does exactly
	// one remainder operation, whereas the straightforward algorithm:
	//
	//   thresh := 2³² - (2³² % n)
	//   while True {
	//     v := src.Uint32()
	//     if v < thresh {
	//       return v % n
	//     }
	//   }
	//
	// does at least two.

	// ...But we have one more trick up our sleeve! First we pull out the first iteration of the loop:
	v := src.Uint32()
	prod := uint64(v) * uint64(n)
	low := uint32(prod)
	// Then we know that thresh < n, so if low >= n, then we already know that low >= thresh without having
	// to explicitly calculate thresh, possibly removing the last expensive operation.
	if low >= n {
		return uint32(prod >> 32)
	}

	// Here we want to calculate 2³² % n, but 2³² doesn't fit in a 32-bit integer. Adding or subtracting n
	// doesn't change the result of the remainder operation, so:
	//
	//   2³² % n = (2³² - n) % n.
	//
	// But uint32(-n) == 2³² - n, so
	//
	//   2³² % n = uint32(-n) % n.
	thresh := uint32(-n) % uint32(n)
	if low >= thresh {
		return uint32(prod >> 32)
	}

	// Since we've already calculate thresh, we can just fall back to the loop we describe above.
	for {
		v = src.Uint32()
		prod = uint64(v) * uint64(n)
		low = uint32(prod)
		if low >= thresh {
			return uint32(prod >> 32)
		}
	}
}

package random

// A Source represents a source of uniformly-distributed pseudo-random int64 values in the range 0 to 2⁶³-1 (inclusive).
//
// We only need pseudo-random values in the range 0 to 2³²-1 (inclusive), but we also want rand.Source objects to
// be usable directly.
type Source interface {
	Int63() int64
}

// randUint32 turns the output of src.Int63() into a uniformly-distributed pseudo-random uint32 value in the range
// 0 to 2³²-1 (inclusive).
func randUint32(src Source) uint32 {
	// Take the top 32 bits, copying rand.Uint32() from https://golang.org/src/math/rand/rand.go .
	return uint32(src.Int63() >> 31)
}

/*
The algorithm used by Uint32n() below is taken from Lemire's "Fast Random Integer Generation in an Interval",
available at https://arxiv.org/abs/1805.10941 . See also
https://lemire.me/blog/2019/06/06/nearly-divisionless-random-integer-generation-on-various-systems/ and
http://www.pcg-random.org/posts/bounded-rands.html for more details; the following is a shorter and (hopefully)
more intuitive explanation.

Lemire's algorithm gets its speed by avoiding expensive divisions and remainder operations as much as possible.
How does it do that? The intuition is to start with this:

  RandomFraction(src Source, n uint32) {
    return randUint32(src) * (n/2³²)
  }

which, if all calculations were done exactly, would return a number at least 0 and less than n. The problem is that
if n doesn't divide 2³² (i.e., n is not a power of two), the returned number would have a fractional part. We can
solve this by doing the multiplication first (with 64-bit integers) and doing integer division. We can also replace
division by 2³² with right-shifting by 32:

  BiasedUint32n(src Source, n uint32) {
    return (uint64(randUint32(src)) * uint64(n)) >> 32
  }

so we've avoided all divisions entirely! But the new problem is that if n doesn't divide 2³², rounding down means
that some numbers will be returned more often than others, i.e. this random number generator is biased.

We solve this by rejecting some values of randUint32(src) (i.e., trying again with a new call to randUint32(src)).
To decide which ones to reject, we look at the low 32 bits of randUint32(src)*n. The logic is the same if
we work with 3-bit integers instead of 32-bit integers, so assume that instead of randUint32(src) we have randUint3(src),
which returns a number from 0 to 2³-1=7, and that n is restricted to be ≤ 7.

As an example, take n=3. Then the possible values of:

  v,
  prod = v*n,
  high = floor(prod / 2³),
  low  = prod % 2³, and

are:

     v:  0  1  2  |  3  4  5  |   6  7
  prod:  0  3  6  |  9 12 15  |  18 21
  high:  0  0  0  |  1  1  1  |   2  2
   low:  0  3  6  |  1  4  7  |   2  5

where we're grouping the table by values of high, which takes values from 0 to n-1.

As mentioned above, because n doesn't evenly divide 2³, each group has either floor(2³/n) = 2 (a small group) or
ceil(2³/n) = 3 values (a big group), which means that if we simply returned high,
we wouldn't get a uniform distribution from 0 to n-1.

We want to use the value of low to decide which values of v to reject. In this case, if we reject all values of v
where low < 2, then we'd throw out v=0 and v=3, so each group would have exactly 2 values.

How do we decide what threshold to set for low? In general, the values of low for a single group look like:

  low: k k+n k+2*n ... k+m*n

where 0 ≤ k < n, and m is the largest integer such that k+m*n < 2³. For the first group, k=0, and if we let k
increase, at some point the rightmost entry will be too big to stay in the group. This correct threshold value
is k == 2³%n, which is 2 for our example above. Therefore, if we filter out values of v where low < 2³%n,
then we remove a single entry from each big group, turning into a small group. Then all groups would have the same size,
and we'd have a uniform distribution. (This is Lemma 4.1, the main result from Lemire's paper.)

What happens if n divides 2³ exactly? Then the threshold would be 2³ % n == 0, and we wouldn't filter out
any values of v, as we would expect.

Now the same logic holds if we work with 32-bit integers; we just replace 2³ with 2³² everywhere.
Now recall that

   floor(prod / 2³²) = prod >> 32

and

   prod % 2³² = uint32(prod).

Then we have the algorithm (in pseudocode): (***)

  Uint32n(src Source, n uint32) {
    threshold := 2³² % n
    while True {
      v := randUint32(src)
      prod := uint64(v) * uint64(n)
      high := prod >> 32
      low  := uint32(prod)
      if low >= threshold {
        return high
      }
    }
  }

Now we have an unbiased algorithm that does exactly one remainder operation! Compare this to the straightforward algorithm:

  SlowUint32n(src Source, n uint32) {
    threshold := 2³² - (2³² % n)
    while True {
      v := randUint32(src)
      if v < threshold {
        return v % n
      }
    }
  }

which does at least two remainder operations.

In fact, the final implementation of Uint32n() avoids even the single remainder operation some of the time:
see the comments in the function for details!
*/

// Uint32n returns a uniformly-distributed number in the range 0 to n-1 (inclusive). n must be non-zero.
//
// This function is basically rand.int31n from https://golang.org/src/math/rand/rand.go , edited for clarity.
func Uint32n(src Source, n uint32) uint32 {
	if n == 0 {
		panic("n must be non-zero in call to Uint32n")
	}

	// As mentioned above, we have one more trick to avoid doing the remainder operation most of the time.
	// First we pull out the first iteration of the loop:
	v := randUint32(src)
	prod := uint64(v) * uint64(n)
	low := uint32(prod)
	// Then we know that threshold < n, so if low ≥ n, then we already know that low ≥ threshold without having
	// to explicitly calculate threshold.
	if low >= n {
		return uint32(prod >> 32)
	}

	// Here we want to calculate 2³² % n, but 2³² doesn't fit in a 32-bit integer. Adding or subtracting n
	// doesn't change the result of the remainder operation, so:
	//
	//   2³² % n == (2³² - n) % n.
	//
	// But for uint32s, -n == 2³² - n, so
	//
	//   2³² % n == -n % n.
	threshold := -n % n
	if low >= threshold {
		return uint32(prod >> 32)
	}

	// Since we've already calculated threshold, we can just fall back to the loop described above (***).
	for {
		v = randUint32(src)
		prod = uint64(v) * uint64(n)
		low = uint32(prod)
		if low >= threshold {
			return uint32(prod >> 32)
		}
	}
}

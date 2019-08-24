package random

// A Source represents a source of uniformly-distributed pseudo-random uint32 values in the range 0 to 2³²-1 (inclusive).
type Source interface {
	Uint32() uint32
}

/*
The algorithm used by UniformUint32() below is taken from Lemire's "Fast Random Integer Generation in an Interval",
available at https://arxiv.org/abs/1805.10941 . See also
https://lemire.me/blog/2019/06/06/nearly-divisionless-random-integer-generation-on-various-systems/ and
http://www.pcg-random.org/posts/bounded-rands.html for more details; the following is a shorter and (hopefully)
more intuitive explanation.

Lemire's algorithm avoids expensive divisions and remainder operations as much as possible. How does it do that?
The intuition is to start with this:

  RandomFraction(src Source, n uint32) {
    return src.Uint32() * (n/2³²)
  }

which, if all calculations were done exactly, would return a number at least 0 and less than n. The problem is that
if n doesn't divide 2³² (i.e., n is not a power of two), the returned number wouldn't have a fractional part. We can
solve this by doing the multiplication first (with 64-bit integers) and doing integer division. We can also replace
division by 2³² with right-shifting by 32:

  BiasedUint32(src Source, n uint32) {
    return (uint64(src.Uint32()) * uint64(n)) >> 32
  }

so it looks like we've avoided all divisions entirely! But the new problem is that if n doesn't divide 2³²,
rounding down means that some numbers will be returned more often than others, i.e. this random number generator
is biased.

How do we solve this? We do so by rejecting some values of src.Uint32() (i.e., trying again with a new call to src.Uint32()),
and to decide which ones to reject, we look at the low 32 bits of src.Uint32()*n. The logic is the same if
we work with 3-bit integers instead of 32-bit integers, so assume that instead of src.Uint32() we have src.Uint3(),
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

Now the question is: how we decide what threshold to set for low? The first fact we use is that the group with v=0
is a big group.contains. This is because that group contains exactly the values of v such that 0 ≤ v*n < 2³,
and there are exactly ceil(2³/n) such values.

The second fact we use is that the leftmost entry in each group has 0 ≤ low < n. This is because if an entry in a group
has low > n, then moving left decreases the value of prod and low by n but leaves high the same.

In general, the values of low for a single group increase by n as you go right. Consider the first group in
our example above:

 0 3 6

If we increment the leftmost value by 1, we must increment all the other values by 1:

 1 4 7

If we do it again, we get:

 2 5 (8)

where the last value is ≥ 2³, so it actually belongs to the following group. So the smallest value of low
where we get a small group is 2³ minus 6, the rightmost value of low of the first group.

Now we can derive a general formula for the threshold value. The rightmost value of low of the first group
is the largest multiple of n < 2³, which is

  2³ - (2³ % n).

Therefore, the threshold value is

  2³ - (2³ - (2³ % n)) = 2³ % n.

That is, if we filter out values of v where low < 2³ % n, then we remove a single entry from each big group,
turning into a small group. Then all groups would have the same size, and we'd have a uniform distribution.

What happens if n divides 2³ exactly? Then the threshold would be 2³ % n == 0, and we wouldn't filter out
any values of v, as we would expect.

Now the same logic holds if we work with 32-bit integers; we just replace 2³ with 2³² everywhere.
Now recall that

   floor(prod / 2³²) = prod >> 32

and

   prod % 2³² = uint32(prod).

Then we have the algorithm (in pseudocode):

  UniformUint32(src Source, n uint32) {
    threshold := 2³² % n
    while True {
      v := src.Uint32()
      prod := uint64(v) * uint64(n)
      high := prod >> 32
      low  := uint32(prod)
      if low >= threshold {
        return high
      }
    }
  }

Now we have an unbiased algorithm that does exactly one remainder operation! Compare this to the straightforward algorithm:

  SlowUniformUint32(src Source, n uint32) {
    threshold := 2³² - (2³² % n)
    while True {
      v := src.Uint32()
      if v < threshold {
        return v % n
      }
    }
  }

which does at least two remainder operations.

In fact, the final implementation of UniformUint32() avoids even the single remainder operation some of the time:
see the comments in the function for details!
*/

// UniformUint32 returns a uniformly-distributed number in the range 0 to n-1 (inclusive). n must be non-zero.
func UniformUint32(src Source, n uint32) uint32 {
	if n == 0 {
		panic("n must be non-zero in call to UniformUint32")
	}

	// As mentioned above, we have one more trick to avoid doing the remainder operation most of the time.
	// First we pull out the first iteration of the loop:
	v := src.Uint32()
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

	// Since we've already calculated threshold, we can just fall back to the loop described above.
	for {
		v = src.Uint32()
		prod = uint64(v) * uint64(n)
		low = uint32(prod)
		if low >= threshold {
			return uint32(prod >> 32)
		}
	}
}

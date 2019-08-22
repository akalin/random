package random

type Source interface {
	Uint32() uint32
}

// Blurb and links about Lemire's algorithm. Mention that the
// complexity is to avoid divisions, which include modulo operations.

func UniformUint32(src Source, n uint32) uint32 {
	// Explain how this is an unrolled first iteration of the loop
	// below, with one optimization.
	v := src.Uint32()
	prod := uint64(v) * uint64(n)
	low := uint32(prod)
	// This is to avoid calculating thresh when not necessary.
	if low >= n {
		return uint32(prod >> 32)
	}

	// Explain how this is really to calculate 2**32 % n.
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

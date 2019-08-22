package random

type Source interface {
	Uint32() uint32
}

func UniformUint32(src Source, n uint32) uint32 {
	v := src.Uint32()
	prod := uint64(v) * uint64(n)
	low := uint32(prod)
	if low < uint32(n) {
		thresh := uint32(-n) % uint32(n)
		for low < thresh {
			v = src.Uint32()
			prod = uint64(v) * uint64(n)
			low = uint32(prod)
		}
	}
	return uint32(prod >> 32)
}

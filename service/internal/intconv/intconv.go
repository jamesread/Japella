// Package intconv converts between integer types with range checks.
//
// Most conversions in Japella are database IDs and counts, which are never
// negative and fit in 32 bits. A value outside the target range can only come
// from corrupt input, so it converts to 0, which matches no row.
package intconv

import "math"

type signed interface {
	~int | ~int32 | ~int64
}

func ToUint32[T signed](v T) uint32 {
	n := int64(v)
	if n < 0 || n > math.MaxUint32 {
		return 0
	}
	return uint32(n)
}

func ToInt32[T signed | ~uint32](v T) int32 {
	n := int64(v)
	if n < math.MinInt32 || n > math.MaxInt32 {
		return 0
	}
	return int32(n)
}

func ToUint64(v int64) uint64 {
	if v < 0 {
		return 0
	}
	return uint64(v)
}

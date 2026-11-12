// Package intconv converts between integer types without silent wraparound.
//
// Values outside the target range are clamped to its nearest bound. Database
// ids and counts never legitimately exceed these ranges, so clamping keeps
// callers simple while turning an overflow into a saturated value instead of
// an unrelated wrapped one.
package intconv

import "math"

type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

func Uint8[T Integer](v T) uint8 {
	if v < 0 {
		return 0
	}
	if uint64(v) > math.MaxUint8 {
		return math.MaxUint8
	}
	return uint8(v)
}

func Uint16[T Integer](v T) uint16 {
	if v < 0 {
		return 0
	}
	if uint64(v) > math.MaxUint16 {
		return math.MaxUint16
	}
	return uint16(v)
}

func Uint32[T Integer](v T) uint32 {
	if v < 0 {
		return 0
	}
	if uint64(v) > math.MaxUint32 {
		return math.MaxUint32
	}
	return uint32(v)
}

func Uint64[T Integer](v T) uint64 {
	if v < 0 {
		return 0
	}
	return uint64(v)
}

func Int32[T Integer](v T) int32 {
	if v < 0 {
		if int64(v) < math.MinInt32 {
			return math.MinInt32
		}
		return int32(v)
	}
	if uint64(v) > math.MaxInt32 {
		return math.MaxInt32
	}
	return int32(v)
}

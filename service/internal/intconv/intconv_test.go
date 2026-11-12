package intconv

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUint32(t *testing.T) {
	assert.Equal(t, uint32(0), Uint32(int64(-1)))
	assert.Equal(t, uint32(42), Uint32(int64(42)))
	assert.Equal(t, uint32(math.MaxUint32), Uint32(int64(math.MaxUint32)))
	assert.Equal(t, uint32(math.MaxUint32), Uint32(int64(math.MaxUint32)+1))
	assert.Equal(t, uint32(math.MaxUint32), Uint32(uint64(math.MaxUint64)))
	assert.Equal(t, uint32(7), Uint32(int32(7)))
	assert.Equal(t, uint32(0), Uint32(int32(math.MinInt32)))
}

func TestUint64(t *testing.T) {
	assert.Equal(t, uint64(0), Uint64(int64(-5)))
	assert.Equal(t, uint64(math.MaxInt64), Uint64(int64(math.MaxInt64)))
	assert.Equal(t, uint64(9), Uint64(9))
}

func TestUint16(t *testing.T) {
	assert.Equal(t, uint16(0), Uint16(-1))
	assert.Equal(t, uint16(500), Uint16(500))
	assert.Equal(t, uint16(math.MaxUint16), Uint16(math.MaxUint16+1))
}

func TestUint8(t *testing.T) {
	assert.Equal(t, uint8(0), Uint8(-1))
	assert.Equal(t, uint8(200), Uint8(200))
	assert.Equal(t, uint8(math.MaxUint8), Uint8(256))
}

func TestInt32(t *testing.T) {
	assert.Equal(t, int32(-3), Int32(int64(-3)))
	assert.Equal(t, int32(math.MinInt32), Int32(int64(math.MinInt32)-1))
	assert.Equal(t, int32(math.MaxInt32), Int32(int64(math.MaxInt32)+1))
	assert.Equal(t, int32(math.MaxInt32), Int32(uint32(math.MaxUint32)))
	assert.Equal(t, int32(12), Int32(uint32(12)))
}

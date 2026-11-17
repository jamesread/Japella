package intconv

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToUint32(t *testing.T) {
	assert.Equal(t, uint32(42), ToUint32(42))
	assert.Equal(t, uint32(42), ToUint32(int32(42)))
	assert.Equal(t, uint32(math.MaxUint32), ToUint32(int64(math.MaxUint32)))
	assert.Equal(t, uint32(0), ToUint32(-1))
	assert.Equal(t, uint32(0), ToUint32(int32(math.MinInt32)))
	assert.Equal(t, uint32(0), ToUint32(int64(math.MaxUint32)+1))
}

func TestToInt32(t *testing.T) {
	assert.Equal(t, int32(42), ToInt32(uint32(42)))
	assert.Equal(t, int32(-7), ToInt32(-7))
	assert.Equal(t, int32(math.MaxInt32), ToInt32(uint32(math.MaxInt32)))
	assert.Equal(t, int32(0), ToInt32(uint32(math.MaxInt32)+1))
	assert.Equal(t, int32(0), ToInt32(int64(math.MinInt32)-1))
}

func TestToUint64(t *testing.T) {
	assert.Equal(t, uint64(42), ToUint64(int64(42)))
	assert.Equal(t, uint64(0), ToUint64(int64(-1)))
}

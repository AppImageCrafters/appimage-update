package zsync

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestComputeBlockRSum(t *testing.T) {
	res := ComputeRollingChecksum([]byte("abcde"))
	assert.Equal(t, uint16(51440), res.toUInt16())

	res = ComputeRollingChecksum([]byte("abcdef"))
	assert.Equal(t, uint16(8279), res.toUInt16())

	res = ComputeRollingChecksum([]byte("abcdefgh"))
	assert.Equal(t, uint16(1575), res.toUInt16())
}

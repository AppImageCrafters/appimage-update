package zsync

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLoadControl(t *testing.T) {
	data := []byte(`zsync: 0.6.2
Filename: Hello World-latest-x86_64.AppImage
MTime: Fri, 08 May 2020 17:36:00 +0000
Blocksize: 2048
Length: 60403752
Hash-Lengths: 2,2,5
URL: Hello World-latest-x86_64.AppImage
SHA-1: da7a3ee0ebb42db73f96c67438ff38c21204f676

--DATA--`)

	control, _ := LoadControl(data)
	assert.Equal(t, "0.6.2", control.Version)
	assert.Equal(t, "Hello World-latest-x86_64.AppImage", control.FileName)
	assert.Equal(t, "Fri, 08 May 2020 17:36:00 +0000", control.MTime)
	assert.Equal(t, uint64(2048), control.BlockSize)
	assert.Equal(t, uint64(60403752), control.Length)
	assert.Equal(t, uint64(2), control.HashLengths.ConsecutiveMatchNeeded)
	assert.Equal(t, uint64(2), control.HashLengths.WeakCheckSumBytes)
	assert.Equal(t, uint64(5), control.HashLengths.StrongCheckSumBytes)
	assert.Equal(t, "Hello World-latest-x86_64.AppImage", control.URL)
	assert.Equal(t, "da7a3ee0ebb42db73f96c67438ff38c21204f676", control.SHA1)
	assert.Equal(t, []byte("--DATA--"), control.data)
}
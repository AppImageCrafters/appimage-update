package zsync

import (
	"fmt"
	"testing"
)

func TestLoad(t *testing.T) {
	data := []byte(`zsync: 0.6.2
Filename: Hello World-latest-x86_64.AppImage
MTime: Fri, 08 May 2020 17:36:00 +0000
Blocksize: 2048
Length: 60403752
Hash-Lengths: 2,2,5
URL: Hello World-latest-x86_64.AppImage
SHA-1: da7a3ee0ebb42db73f96c67438ff38c21204f676

---DATA--`)

	control, _ := Load(data)
	fmt.Print(control)
}

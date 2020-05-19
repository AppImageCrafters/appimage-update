package zsync

import (
	"io/ioutil"
	"testing"
)

func TestSync(t *testing.T) {
	controlData, _ := ioutil.ReadFile("/tmp/OpenRA-Red-Alert-x86_64.AppImage.zsync")
	ParseControl(controlData)
}

package update

import (
	"appimage-update/src/appimage/update/methods"
	"fmt"
	"strings"
)

type Method interface {
	Name() string
	Execute() error
}

// factory methods for creating Update Method instances from the AppImage update info
func NewMethod(updateInfo *string) (Method, error) {
	parts := strings.Split(*updateInfo, "|")

	switch parts[0] {
	case "zsync":
		return methods.NewZsyncUpdate(parts)
	case "gh-releases-zsync":
		return methods.NewGitHubUpdate(parts)
	case "bintray-zsync":
		return methods.NewBintrayZsync(parts)
	default:
		return nil, fmt.Errorf("Unknown update method: " + parts[0])
	}
}

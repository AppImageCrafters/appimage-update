package update

import (
	"appimage-update/src/appimage"
	"appimage-update/src/appimage/update/updaters"
	"fmt"
	"strings"
)

type Updater interface {
	Method() string

	Lookup() (updateAvailable bool, err error)
	Download() (output string, err error)
}

// factory updaters for creating Updater instances from an AppImage file
func NewUpdaterFor(target *string) (Updater, error) {
	appImage := appimage.AppImage{*target}
	updateInfoString, err := appImage.GetUpdateInfo()
	if err != nil {
		return nil, err
	}

	if strings.HasPrefix(updateInfoString, "zsync") {
		return updaters.NewZSyncUpdater(&updateInfoString, target)
	}

	if strings.HasPrefix(updateInfoString, "gh-releases-zsync") {
		return updaters.NewGitHubUpdater(&updateInfoString, target)
	}

	return nil, fmt.Errorf("Invalid updated information: ", updateInfoString)
}

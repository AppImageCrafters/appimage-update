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

	return NewUpdateForUpdateString(updateInfoString, appImage)
}

func NewUpdateForUpdateString(updateInfoString string, appImage appimage.AppImage) (Updater, error) {
	if strings.HasPrefix(updateInfoString, "zsync") {
		return updaters.NewZSyncUpdater(&updateInfoString, &appImage)
	}

	if strings.HasPrefix(updateInfoString, "gh-releases-zsync") {
		return updaters.NewGitHubZsyncUpdater(&updateInfoString, &appImage)
	}

	if strings.HasPrefix(updateInfoString, "gh-releases-direct") {
		return updaters.NewGitHubDirectUpdater(&updateInfoString, &appImage)
	}

	if strings.HasPrefix(updateInfoString, "ocs-v1-appimagehub-direct") {
		return updaters.NewOCSAppImageHubDirect(&updateInfoString, &appImage)
	}

	if strings.HasPrefix(updateInfoString, "ocs-v1-appimagehub-zsync") {
		return updaters.NewOCSAppImageHubZSync(&updateInfoString, &appImage)
	}

	return nil, fmt.Errorf("Invalid updated information: ", updateInfoString)
}

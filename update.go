package update

import (
	"fmt"
	"strings"

	"github.com/AppImageCrafters/appimage-update/updaters"
	"github.com/AppImageCrafters/appimage-update/util"
)

type Updater interface {
	Method() string

	Lookup() (updateAvailable bool, err error)
	Download() (output string, err error)
}

// factory updaters for creating Updater instances from an AppImage file
func NewUpdaterFor(target string) (Updater, error) {
	updateInfoString, err := util.ReadUpdateInfo(target)
	if err != nil {
		return nil, err
	}

	return NewUpdateForUpdateString(updateInfoString, target)
}

func NewUpdateForUpdateString(updateInfoString string, appImagePath string) (Updater, error) {
	if strings.HasPrefix(updateInfoString, "zsync") {
		return updaters.NewZSyncUpdater(&updateInfoString, appImagePath)
	}

	if strings.HasPrefix(updateInfoString, "gh-releases-zsync") {
		return updaters.NewGitHubZsyncUpdater(&updateInfoString, appImagePath)
	}

	if strings.HasPrefix(updateInfoString, "gh-releases-direct") {
		return updaters.NewGitHubDirectUpdater(&updateInfoString, appImagePath)
	}

	if strings.HasPrefix(updateInfoString, "ocs-v1-appimagehub-direct") {
		return updaters.NewOCSAppImageHubDirect(&updateInfoString, appImagePath)
	}

	if strings.HasPrefix(updateInfoString, "ocs-v1-appimagehub-zsync") {
		return updaters.NewOCSAppImageHubZSync(&updateInfoString, appImagePath)
	}

	return nil, fmt.Errorf("Invalid updated information: ", updateInfoString)
}

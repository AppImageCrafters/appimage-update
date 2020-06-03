package updaters

import (
	"appimage-update/src/appimage"
	"appimage-update/src/zsync"
	"appimage-update/src/zsync/control"
	"bytes"
	"fmt"
	"github.com/schollz/progressbar/v3"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type ZSync struct {
	url string

	seed          appimage.AppImage
	seedSHA1      string
	seedRenamed   bool
	updateControl *control.Control
}

func NewZSyncUpdater(updateInfoString *string, target *appimage.AppImage) (*ZSync, error) {
	parts := strings.Split(*updateInfoString, "|")

	if len(parts) != 2 {
		return nil, fmt.Errorf("Invalid zsync update info. Expected: zsync|<url>")
	}

	info := ZSync{
		url:         parts[1],
		seed:        *target,
		seedRenamed: false,
	}

	return &info, nil
}

func (inst *ZSync) Method() string {
	return "zsync"
}

func (inst *ZSync) Lookup() (updateAvailable bool, err error) {
	zsyncRawData, err := getZsyncRawData(inst.url)
	if err != nil {
		return false, err
	}

	inst.updateControl, err = control.ParseControl(zsyncRawData)
	if err != nil {
		return false, err
	}

	inst.seedSHA1 = inst.seed.GetSHA1()

	if inst.seedSHA1 == inst.updateControl.SHA1 {
		return false, nil
	}

	return true, nil
}

func (inst *ZSync) Download() (output string, err error) {
	output = inst.GetOutputPath()
	inst.updateControl.URL = inst.resolveUrl()
	err = inst.RenameSeedIfRequired(output)
	if err != nil {
		return
	}

	err = inst.DownloadTo(output)
	if err != nil {
		fmt.Println("Old AppImage restored to: ", output)
		inst.restoreFileAppImage(output)
		return
	}

	err = inst.validateDownload(output)
	return
}

func (inst *ZSync) RenameSeedIfRequired(output string) (err error) {
	if output == inst.seed.Path {
		fileExtension := filepath.Ext(inst.seed.Path)
		newSeedPath := inst.seed.Path[:len(inst.seed.Path)-len(fileExtension)]
		newSeedPath = newSeedPath + "-old" + fileExtension

		err = os.Rename(inst.seed.Path, newSeedPath)
		if err != nil {
			return
		}

		inst.seed.Path = newSeedPath
		inst.seedRenamed = true
		fmt.Println("Old AppImage renamed to: ", newSeedPath)
	}
	return nil
}

func (inst *ZSync) GetOutputPath() (output string) {
	return filepath.Dir(inst.seed.Path) + "/" + inst.updateControl.FileName
}

func (inst *ZSync) resolveUrl() string {
	if strings.HasPrefix(inst.updateControl.URL, "http") ||
		strings.HasPrefix(inst.updateControl.URL, "ftp") {
		return inst.updateControl.URL
	}

	urlPrefixEnd := strings.LastIndex(inst.url, "/")
	return inst.url[:urlPrefixEnd] + "/" + inst.updateControl.URL
}

func (inst *ZSync) validateDownload(output string) error {
	appImage := appimage.AppImage{output}
	if appImage.GetSHA1() != inst.updateControl.SHA1 {
		inst.restoreFileAppImage(output)
		fmt.Println("Old AppImage restored to: ", output)
		return fmt.Errorf("downloaded file checksums don't match")
	} else {
		fmt.Println("File checksum verified.")
	}

	return nil
}

func (inst *ZSync) restoreFileAppImage(output string) {
	if inst.seedRenamed {
		_ = os.Rename(inst.seed.Path, output)
	} else {
		_ = os.Remove(output)
	}
}

func (inst *ZSync) DownloadTo(targetPath string) (err error) {
	local, err := os.Open(inst.seed.Path)
	if err != nil {
		return
	}
	defer local.Close()

	output, err := os.OpenFile(targetPath, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return
	}
	defer output.Close()

	err = zsync.Sync(local, output, *inst.updateControl)
	return
}

func getZsyncRawData(url string) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "appimage-update-go/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("Zsync file download failed: %d", resp.StatusCode)
	}

	bar := progressbar.DefaultBytes(
		resp.ContentLength,
		"Downloading zsync file: ",
	)

	var buf bytes.Buffer
	_, err = io.Copy(io.MultiWriter(&buf, bar), resp.Body)

	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

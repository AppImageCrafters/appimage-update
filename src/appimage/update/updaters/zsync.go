package updaters

import (
	"appimage-update/src/appimage"
	"appimage-update/src/zsync"
	"appimage-update/src/zsync/blocksources"
	"appimage-update/src/zsync/control"
	"appimage-update/src/zsync/filechecksum"
	"bytes"
	"fmt"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/crypto/md4"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type ZSync struct {
	url string

	seed     appimage.AppImage
	seedSHA1 string

	updateControl *control.Control
}

func NewZSyncUpdater(updateInfoString *string, target *appimage.AppImage) (*ZSync, error) {
	parts := strings.Split(*updateInfoString, "|")

	if len(parts) != 2 {
		return nil, fmt.Errorf("Invalid zsync update info. Expected: zsync|<url>")
	}

	info := ZSync{
		url:  parts[1],
		seed: *target,
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
	output = filepath.Dir(inst.seed.Path) + "/" + inst.updateControl.FileName

	fs := &zsync.BasicSummary{
		ChecksumIndex:  inst.updateControl.ChecksumIndex,
		ChecksumLookup: inst.updateControl.ChecksumLookup,
		BlockCount:     inst.updateControl.Blocks,
		BlockSize:      inst.updateControl.BlockSize,
		FileSize:       inst.updateControl.FileLength,
	}

	inputFile, err := os.Open(inst.seed.Path)
	if err != nil {
		return
	}

	patchedFile, err := os.OpenFile(output, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return
	}

	resolver := blocksources.MakeFileSizedBlockResolver(
		uint64(fs.GetBlockSize()),
		fs.GetFileSize(),
	)

	rsync := &zsync.RSync{
		Input:  inputFile,
		Output: patchedFile,
		Source: blocksources.NewHttpBlockSource(
			inst.resolveUrl(),
			1,
			resolver,
			&filechecksum.HashVerifier{
				Hash:                md4.New(),
				BlockSize:           fs.GetBlockSize(),
				BlockChecksumGetter: fs,
				FinalChunkLen:       inst.updateControl.HashLengths.StrongCheckSumBytes,
			},
		),
		Summary: fs,
		OnClose: nil,
	}

	err = rsync.Patch()

	if err != nil {
		return
	}

	rsync.Close()

	err = inst.validateDownload(output)
	if err != nil {
		os.Remove(output)

		return
	}
	return output, nil
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
		return fmt.Errorf("downloaded file checksums don't match")
	} else {
		fmt.Println("File checksum verified.")
	}

	return nil
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

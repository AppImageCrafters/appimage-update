package updaters

import (
	"appimage-update/src/appimage"
	"appimage-update/src/zsync"
	"bufio"
	"bytes"
	"fmt"
	"github.com/Redundancy/go-sync"
	"github.com/Redundancy/go-sync/filechecksum"
	"github.com/Redundancy/go-sync/indexbuilder"
	"github.com/schollz/progressbar/v3"
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

	updateControl *zsync.Control
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

	inst.updateControl, err = zsync.ParseControl(zsyncRawData)
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
	generator := filechecksum.NewFileChecksumGenerator(uint(inst.updateControl.BlockSize))
	sourceFile, _ := os.Open(inst.seed.Path)

	_, referenceFileIndex, checksumLookup, err := indexbuilder.BuildChecksumIndex(generator, sourceFile)
	if err != nil {
		return output, err
	}
	sourceFileInfo, err := sourceFile.Stat()
	if err != nil {
		return output, err
	}
	output = filepath.Dir(inst.seed.Path) + "/" + inst.updateControl.FileName

	blockCount := (uint64(sourceFileInfo.Size()) + inst.updateControl.BlockSize - 1) / inst.updateControl.BlockSize

	fs := &gosync.BasicSummary{
		ChecksumIndex:  referenceFileIndex,
		ChecksumLookup: checksumLookup,
		BlockCount:     uint(blockCount),
		BlockSize:      uint(inst.updateControl.BlockSize),
		FileSize:       sourceFileInfo.Size(),
	}

	rsync, err := gosync.MakeRSync(inst.seed.Path, inst.url, output, fs)
	if err != nil {
		return output, err
	}

	defer rsync.Close()

	err = rsync.Patch()

	if err != nil {
		return output, err
	}

	outputAppImage := appimage.AppImage{Path: output}
	outputAppImage.SetExecutionPermissions()

	return output, nil
}

func getZsyncRawData(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bar := progressbar.DefaultBytes(
		resp.ContentLength,
		"Downloading zsync file: "+url,
	)

	var buf bytes.Buffer
	bufWriter := bufio.NewWriter(&buf)

	_, err = io.Copy(io.MultiWriter(bufWriter, bar), resp.Body)

	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

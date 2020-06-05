package appimage

import (
	"bytes"
	"crypto/sha1"
	"debug/elf"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
)

type AppImage struct {
	Path string
}

func (target *AppImage) GetUpdateInfo() (string, error) {
	elfFile, err := elf.Open(target.Path)
	if err != nil {
		panic("Unable to open target: \"" + target.Path + "\"." + err.Error())
	}

	updInfo := elfFile.Section(".upd_info")
	if updInfo == nil {
		panic("Missing update section on target elf ")
	}
	sectionData, err := updInfo.Data()

	if err != nil {
		panic("Unable to parse update section: " + err.Error())
	}

	str_end := bytes.Index(sectionData, []byte("\000"))
	if str_end == -1 || str_end == 0 {
		return "", fmt.Errorf("No update information found in: " + target.Path)
	}

	update_info := string(sectionData[:str_end])
	return update_info, nil
}

func (target *AppImage) GetSHA1() string {
	f, err := os.Open(target.Path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}

	return hex.EncodeToString(h.Sum(nil))
}

func (target *AppImage) SetExecutionPermissions() error {
	return os.Chmod(target.Path, 7550)
}

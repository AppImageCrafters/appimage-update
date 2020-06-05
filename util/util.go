package util

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

func GetSHA1(appImagePath string) string {
	f, err := os.Open(appImagePath)
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

func setExecutionPermissions(appImagePath string) error {
	return os.Chmod(appImagePath, 7550)
}

func ReadUpdateInfo(appImagePath string) (string, error) {
	elfFile, err := elf.Open(appImagePath)
	if err != nil {
		panic("Unable to open target: \"" + appImagePath + "\"." + err.Error())
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
		return "", fmt.Errorf("No update information found in: " + appImagePath)
	}

	update_info := string(sectionData[:str_end])
	return update_info, nil
}

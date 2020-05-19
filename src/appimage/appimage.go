package appimage

import (
	"bytes"
	"debug/elf"
	"fmt"
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
		return "", fmt.Errorf("No update information found")
	}

	update_info := string(sectionData[:str_end])
	return update_info, nil
}

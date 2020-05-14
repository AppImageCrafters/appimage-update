package main

import (
	"appimage-update/src/appimage"
	"appimage-update/src/appimage/update"
	"flag"
	"fmt"
)

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		panic("Target File paths expected")
	}

	for _, target := range args {
		fmt.Println("Reading update information: ", target)

		appImage := appimage.AppImage{target}
		updateInfoString, err := appImage.GetUpdateInfo()
		if err != nil {
			fmt.Println(err)
			continue
		}

		updateMethod, err := update.NewMethod(updateInfoString)
		if err != nil {
			fmt.Println(err)
			continue
		} else {
			updateMethod.Execute()
		}
	}
}

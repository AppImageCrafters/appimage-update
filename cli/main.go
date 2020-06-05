package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/AppImageCrafters/appimage-update"
)

func main() {
	var updateInfoString string
	flag.StringVar(&updateInfoString, "u", "", "Custom update info string")

	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		panic("Target File paths expected")
	}

	if updateInfoString != "" {
		appImage := args[0]
		updater, err := update.NewUpdateForUpdateString(updateInfoString, appImage)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}

		tryUpdate(appImage, updater)
		return
	}

	for _, target := range args {
		updateMethod, err := update.NewUpdaterFor(target)
		if err != nil {
			fmt.Println(err.Error())
		} else {
			tryUpdate(target, updateMethod)
		}
	}
}

func tryUpdate(target string, updateMethod update.Updater) {

	fmt.Println("Looking for updates of: ", target)
	updateAvailable, err := updateMethod.Lookup()
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	if !updateAvailable {
		fmt.Println("No updates were found for: ", target)
		return
	}

	result, err := updateMethod.Download()
	if err != nil {
		fmt.Println("Error: ", err.Error())
		return
	}

	fmt.Println("Update downloaded to: " + result)
}

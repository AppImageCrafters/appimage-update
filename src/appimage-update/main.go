package main

import (
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
		updateMethod, err := update.NewUpdaterFor(&target)
		if err != nil {
			fmt.Println(err.Error())
			continue
		}

		fmt.Println("Looking for updates of: ", target)
		updateAvailable, err := updateMethod.Lookup()
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
		if !updateAvailable {
			fmt.Println("No updates were found for: ", target)
			continue
		}

		result, err := updateMethod.Download()
		if err != nil {
			fmt.Println(err.Error())
		}
		fmt.Println("Update downloaded to: " + result)
	}
}

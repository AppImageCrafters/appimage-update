package updaters

import (
	"fmt"

	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/schollz/progressbar/v3"
)

type Direct struct {
	url  string
	seed string
}

func NewDirectUpdater(url string, seed string) (*Direct, error) {
	return &Direct{
		url:  url,
		seed: seed,
	}, nil
}

func (d *Direct) Method() string {
	return "direct"
}

func (d *Direct) Lookup() (updateAvailable bool, err error) {
	outputFile := d.getOutputFileName()
	if d.seed == outputFile {
		return false, nil
	} else {
		return true, nil
	}
}

func (d *Direct) Download() (output string, err error) {
	output = d.getOutputFileName()
	err = downloadFile(output, d.url)

	return
}

func (d *Direct) getOutputFileName() string {
	urlLastPartStart := strings.LastIndex(d.url, "/")
	if urlLastPartStart == -1 {
		urlLastPartStart = 0
	}

	urlArgumentsStart := strings.LastIndex(d.url, "?")
	if urlArgumentsStart == -1 {
		urlArgumentsStart = len(d.url)
	}

	fileName := d.url[urlLastPartStart:urlArgumentsStart]

	return filepath.Dir(d.seed) + "/" + fileName
}

func downloadFile(filepath string, url string) (err error) {

	// Create the file
	out, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	progress := progressbar.DefaultBytes(
		resp.ContentLength,
		"Downloading: "+url,
	)

	// Writer the body to file
	_, err = io.Copy(io.MultiWriter(out, progress), resp.Body)
	if err != nil {
		return err
	}

	return nil
}

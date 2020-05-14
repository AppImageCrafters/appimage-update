package methods

import (
	"appimage-update/src/zsync"
	"bufio"
	"bytes"
	"fmt"
	"github.com/schollz/progressbar/v3"
	"io"
	"net/http"
)

type Zsync struct {
	url string
}

func (*Zsync) Name() string {
	return "zsync"
}

func (instance *Zsync) Execute() error {
	fmt.Println("Running ", instance.Name())

	zsyncRawData, err := getZsyncRawData(instance.url)
	if err != nil {
		return err
	}
	zsync.Load(zsyncRawData)

	return nil
}

func NewZsyncUpdate(parts []string) (*Zsync, error) {
	if len(parts) != 2 {
		return nil, fmt.Errorf("Invalid zsync update info. Expected: zsync|<url>")
	}

	info := Zsync{parts[1]}
	return &info, nil
}

func getZsyncRawData(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bar := progressbar.DefaultBytes(
		resp.ContentLength,
		"Downloading zsync file: ",
	)

	var buf bytes.Buffer
	bufWriter := bufio.NewWriter(&buf)

	_, err = io.Copy(io.MultiWriter(bufWriter, bar), resp.Body)

	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

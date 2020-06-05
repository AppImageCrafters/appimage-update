package updaters

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/beevik/etree"
	"github.com/danwakefield/fnmatch"
	"github.com/schollz/progressbar/v3"
)

type OCSAppImageHubZSync struct {
	zsync ZSync

	apiV1Url  string
	productId string
	pattern   string
}

func NewOCSAppImageHubZSync(updateInfoString *string, appImagePath string) (*OCSAppImageHubZSync, error) {
	parts := strings.Split(*updateInfoString, "|")

	if len(parts) != 4 {
		return nil, fmt.Errorf("Invalid OCSAppImageHubZSync update instance. Expected: ocs-v1-appimagehub-zsync|<api url>|<product id>|<pattern>")
	}

	instance := OCSAppImageHubZSync{
		zsync: ZSync{
			seed: appImagePath,
		},

		apiV1Url:  parts[1],
		productId: parts[2],
		pattern:   parts[3],
	}

	return &instance, nil
}

func (O *OCSAppImageHubZSync) Method() string {
	return "ocs-v1-appimagehub-zsync"
}

func (O *OCSAppImageHubZSync) Lookup() (updateAvailable bool, err error) {
	url := fmt.Sprint("https://", O.apiV1Url, "/content/data/", O.productId)
	data, err := getOcsContentData(url)
	if err != nil {
		return false, err
	}

	doc := etree.NewDocument()
	err = doc.ReadFromBytes(data)
	if err != nil {
		return false, err
	}

	downloadUrl, err := O.getDownload(doc, O.pattern)
	if err != nil {
		return false, err
	}

	O.zsync.url = downloadUrl + ".zsync"
	return O.zsync.Lookup()
}

func (O *OCSAppImageHubZSync) getDownload(doc *etree.Document, pattern string) (url string, err error) {
	downloadIdx := 1

	for true {
		downloadNameTag := fmt.Sprintf("//downloadname%d", downloadIdx)
		downloadNameItem := doc.FindElement(downloadNameTag)
		if downloadNameItem == nil {
			break
		}
		downloadName := downloadNameItem.Text()

		if fnmatch.Match(pattern, downloadName, fnmatch.FNM_IGNORECASE) {
			downloadLinkTag := fmt.Sprintf("//downloadlink%d", downloadIdx)
			return doc.FindElement(downloadLinkTag).Text(), nil
		}

		downloadIdx++
	}

	return "", fmt.Errorf("no downloads found for: %s", pattern)
}

func (O *OCSAppImageHubZSync) Download() (output string, err error) {
	return O.zsync.Download()
}

func getOcsContentData(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bar := progressbar.DefaultBytes(
		resp.ContentLength,
		"Downloading content data: "+url,
	)

	var buf bytes.Buffer
	_, err = io.Copy(io.MultiWriter(&buf, bar), resp.Body)

	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

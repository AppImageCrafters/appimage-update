package updaters

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/beevik/etree"
	"github.com/danwakefield/fnmatch"
	"github.com/schollz/progressbar/v3"
)

type OCSAppImageHubDirect struct {
	direct Direct

	apiV1Url  string
	productId string
	fileName  string
}

func NewOCSAppImageHubDirect(updateInfoString *string, target string) (*OCSAppImageHubDirect, error) {
	parts := strings.Split(*updateInfoString, "|")

	if len(parts) != 4 {
		return nil, fmt.Errorf("Invalid OCSAppImageHubDirect update instance. Expected: ocs-v1-appimagehub-direct|<api url>|<product id>|<file name>")
	}

	instance := OCSAppImageHubDirect{
		direct: Direct{
			seed: target,
		},

		apiV1Url:  parts[1],
		productId: parts[2],
		fileName:  parts[3],
	}

	return &instance, nil
}

func (O *OCSAppImageHubDirect) Method() string {
	return "ocs-v1-appimagehub-direct"
}

func (O *OCSAppImageHubDirect) Lookup() (updateAvailable bool, err error) {
	url := fmt.Sprint("https://", O.apiV1Url, "/content/data/", O.productId)
	data, err := getContentData(url)
	if err != nil {
		return false, err
	}

	doc := etree.NewDocument()
	err = doc.ReadFromBytes(data)
	if err != nil {
		return false, err
	}

	O.direct.url = O.resolveDownloadUrl(doc)
	return O.direct.Lookup()
}

func (O *OCSAppImageHubDirect) resolveDownloadUrl(doc *etree.Document) string {
	downloadIdx := 1
	for true {
		downloadNameTag := fmt.Sprintf("//downloadname%d", downloadIdx)
		downloadNameItem := doc.FindElement(downloadNameTag)
		if downloadNameItem == nil {
			break
		}
		downloadName := downloadNameItem.Text()

		if fnmatch.Match(O.fileName, downloadName, fnmatch.FNM_IGNORECASE) {
			downloadLinkTag := fmt.Sprintf("//downloadlink%d", downloadIdx)
			return doc.FindElement(downloadLinkTag).Text()
		}

		downloadIdx++
	}

	return ""
}

func (O *OCSAppImageHubDirect) Download() (output string, err error) {
	return O.direct.Download()
}

func getContentData(url string) ([]byte, error) {
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
	bufWriter := bufio.NewWriter(&buf)

	_, err = io.Copy(io.MultiWriter(bufWriter, bar), resp.Body)

	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

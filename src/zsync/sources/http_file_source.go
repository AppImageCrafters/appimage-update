package chunks

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

type HttpFileSource struct {
	URL    string
	Offset int64
	Size   int64
}

func (h HttpFileSource) Read(b []byte) (n int, err error) {
	rangedRequest, err := http.NewRequest("GET", h.URL, nil)

	if err != nil {
		return 0, fmt.Errorf("Error creating request for \"%v\": %v", h.URL, err)
	}

	range_start := h.Offset
	range_end := h.Offset + int64(len(b)) - 1

	rangeSpecifier := fmt.Sprintf("bytes=%v-%v", range_start, range_end)
	rangedRequest.ProtoAtLeast(1, 1)
	rangedRequest.Header.Add("Range", rangeSpecifier)
	rangedRequest.Header.Add("Accept-Encoding", "identity")

	client := &http.Client{}
	rangedResponse, err := client.Do(rangedRequest)

	if err != nil {
		return 0, fmt.Errorf("Error executing request for \"%v\": %v", h.URL, err)
	}

	defer rangedResponse.Body.Close()

	if rangedResponse.StatusCode == 404 {
		return 0, fmt.Errorf("URL not found")
	} else if rangedResponse.StatusCode != 206 {
		return 0, fmt.Errorf("ranged request not supported")
	} else if strings.Contains(
		rangedResponse.Header.Get("Content-Encoding"),
		"gzip",
	) {
		return 0, fmt.Errorf("response from server was GZiped")
	} else {
		bytesRead, err := io.ReadFull(rangedResponse.Body, b)

		if err != nil {
			err = fmt.Errorf(
				"Failed to read response body for %v (%v-%v): %v",
				h.URL, range_start, range_end,
				err,
			)
		}

		if bytesRead != len(b) {
			err = fmt.Errorf(
				"Unexpected response length %v (%v): %v",
				h.URL,
				bytesRead,
				len(b),
			)
		}

		return bytesRead, nil
	}
}

func (h *HttpFileSource) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case 0:
		h.Offset = offset
	case 1:
		h.Offset += offset
	case 2:
		h.Offset = h.Size + offset
	default:
		return -1, fmt.Errorf("Unknown whence value: %d", whence)
	}

	return offset, nil
}

package zsync

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

type ControlHeader struct {
	Version     string
	MTime       string
	FileName    string
	BlockSize   uint64
	Length      uint64
	HashLengths []uint64
	URL         string
	SHA1        string
}

type Control struct {
	ControlHeader

	data []byte
}

func LoadControl(data []byte) (*Control, error) {
	header, dataStart := loadControlHeader(data)

	return &Control{header, data[dataStart:]}, nil
}

func loadControlHeader(data []byte) (header ControlHeader, dataStart int) {
	slice := data[:]
	line_end := bytes.Index(slice, []byte("\n"))
	dataStart += line_end

	for line_end != 0 && line_end != -1 {
		line := string(slice[:line_end])

		k, v := parseHeaderLine(line)
		setHeaderValue(&header, k, v)

		slice = slice[line_end+1:]
		line_end = bytes.Index(slice, []byte("\n"))
		dataStart += line_end
	}

	return header, bytes.Index(data, slice[1:])
}

func setHeaderValue(header *ControlHeader, k string, v string) {
	switch k {
	case "zsync":
		header.Version = v
	case "filename":
		header.FileName = v
	case "mtime":
		header.MTime = v
	case "blocksize":
		vi, err := strconv.ParseUint(v, 10, 0)
		if err == nil {
			header.BlockSize = vi
		}

	case "length":
		vi, err := strconv.ParseUint(v, 10, 0)
		if err == nil {
			header.Length = vi
		}
	case "hash-lengths":
		hashLenghts, err := parseHaseLengths(v)
		if err == nil {
			header.HashLengths = hashLenghts
		}
	case "url":
		header.URL = v
	case "sha-1":
		header.SHA1 = v
	default:
		fmt.Println("Unknown zsync control key: " + k)
	}
}

func parseHaseLengths(s string) (hashLengths []uint64, err error) {
	parts := strings.Split(s, ",")
	hashLengths = make([]uint64, len(parts))

	for i, v := range parts {
		vi, err := strconv.ParseUint(v, 10, 0)
		if err == nil {
			hashLengths[i] = vi
		} else {
			return nil, err
		}
	}

	return hashLengths, nil
}

func parseHeaderLine(line string) (key string, value string) {
	parts := strings.SplitN(line, ":", 2)
	key = strings.ToLower(parts[0])

	if len(parts) == 2 {
		value = strings.TrimSpace(parts[1])
	}

	return key, value
}

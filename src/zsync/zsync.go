package zsync

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

type ZsyncHeader struct {
	Version     string
	MTime       string
	FileName    string
	BlockSize   uint64
	Length      uint64
	HashLengths []uint64
	URL         string
	SHA1        string
}

type ZsyncControl struct {
	ZsyncHeader

	data []byte
}

func Load(data []byte) (*ZsyncHeader, error) {
	slice := data[:]
	line_end := bytes.Index(slice, []byte("\n"))

	header := &ZsyncHeader{}
	for line_end != 0 && line_end != -1 {
		line := string(slice[:line_end])

		k, v := parseHeaderLine(line)
		switch k {
		case "zsync":
			header.Version = v
		case "filename":
			header.FileName = v
		case "mtime":
			header.MTime = v
		case "blocksize":
			vi, err := strconv.ParseUint(v, 10, 64)
			if err != nil {
				header.BlockSize = vi
			}

		case "length":
			vi, err := strconv.ParseUint(v, 10, 64)
			if err != nil {
				header.Length = vi
			}
		case "hash-lengths":
			hashLenghts, err := parseHaseLengths(v)
			if err != nil {
				header.HashLengths = hashLenghts
			}
		case "url":
			header.URL = v
		case "sha-1":
			header.SHA1 = v
		default:
			fmt.Println("Unknown zsync control key: " + k)
		}

		slice = slice[line_end+1:]
		line_end = bytes.Index(slice, []byte("\n"))
	}

	return header, nil
}

func parseHaseLengths(s string) (hashLengths []uint64, err error) {
	parts := strings.Split(s, ",")
	hashLengths = make([]uint64, len(parts))

	for i, v := range parts {
		vi, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			hashLengths[i] = vi
		} else {
			return nil, err
		}
	}

	return hashLengths, nil
}

func parseHeaderLine(line string) (key string, value string) {
	fmt.Println(line)

	parts := strings.SplitN(line, ":", 2)
	key = strings.ToLower(parts[0])

	if len(parts) == 2 {
		value = parts[1]
	}

	return key, value
}

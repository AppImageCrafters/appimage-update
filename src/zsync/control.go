package zsync

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

type ControlHeaderHashLenghts struct {
	ConsecutiveMatchNeeded uint64
	WeakCheckSumBytes      uint64
	StrongCheckSumBytes    uint64
}

type ControlHeader struct {
	Version     string
	MTime       string
	FileName    string
	Blocks      uint64
	BlockSize   uint64
	FileLength  uint64
	HashLengths ControlHeaderHashLenghts
	URL         string
	SHA1        string
}

type Control struct {
	ControlHeader

	checksums [][]byte
}

func ParseControl(data []byte) (*Control, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("Missing zsync control data")
	}
	header, _, err := LoadControlHeader(data)
	if err != nil {
		return nil, err
	}

	return &Control{header, nil}, nil
}

func LoadControlHeader(data []byte) (header ControlHeader, dataStart int, err error) {
	slice := data[:]
	line_end := bytes.Index(slice, []byte("\n"))

	// the header end is marked by an empty line "\n"
	for line_end != 0 && line_end != -1 {
		dataStart += line_end + 1
		line := string(slice[:line_end])

		k, v := parseHeaderLine(line)
		setHeaderValue(&header, k, v)

		slice = slice[line_end+1:]
		line_end = bytes.Index(slice, []byte("\n"))
	}

	if line_end >= 0 {
		dataStart += line_end + 1
	}

	if header.BlockSize == 0 {
		return header, dataStart, fmt.Errorf("Malformed zsync control: missing BlockSize ")
	}

	header.Blocks = (header.FileLength + header.BlockSize - 1) / header.BlockSize

	return header, dataStart, nil
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
			header.FileLength = vi
		}
	case "hash-lengths":
		hashLenghts, err := parseHaseLengths(v)
		if err == nil {
			header.HashLengths = *hashLenghts
		}
	case "url":
		header.URL = v
	case "sha-1":
		header.SHA1 = v
	default:
		fmt.Println("Unknown zsync control key: " + k)
	}
}

func parseHaseLengths(s string) (hashLengths *ControlHeaderHashLenghts, err error) {
	const errorPrefix = "Invalid Hash-Lengths entry"
	parts := strings.Split(s, ",")
	hashLengthsArray := make([]uint64, len(parts))

	for i, v := range parts {
		vi, err := strconv.ParseUint(v, 10, 0)
		if err == nil {
			hashLengthsArray[i] = vi
		} else {
			return nil, err
		}
	}

	if len(hashLengthsArray) != 3 {
		return nil,
			fmt.Errorf(errorPrefix + ", expected: " + " ConsecutiveMatchNeeded, WeakCheckSumBytes, StrongCheckSumBytes")
	}

	hashLengths = &ControlHeaderHashLenghts{
		ConsecutiveMatchNeeded: hashLengthsArray[0],
		WeakCheckSumBytes:      hashLengthsArray[1],
		StrongCheckSumBytes:    hashLengthsArray[2],
	}

	if hashLengths.ConsecutiveMatchNeeded < 1 || hashLengths.ConsecutiveMatchNeeded > 2 {
		return nil, fmt.Errorf(errorPrefix + ": ConsecutiveMatchNeeded must be in rage [1, 2] ")
	}

	if hashLengths.WeakCheckSumBytes < 1 || hashLengths.WeakCheckSumBytes > 4 {
		return nil, fmt.Errorf(errorPrefix + ": WeakCheckSumBytes must be in rage [1, 4] ")
	}

	if hashLengths.StrongCheckSumBytes < 3 || hashLengths.StrongCheckSumBytes > 16 {
		return nil, fmt.Errorf(errorPrefix + ": StrongCheckSumBytes must be in rage [4, 16] ")
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

package sources

import (
	"io"
)

type Source interface {
	Read(b []byte) (n int, err error)
	Seek(offset int64, whence int) (int64, error)
}

func ReadChunk(source Source, offset int64, requiredBytes int64) (blockData []byte, err error) {
	_, err = source.Seek(offset, 0)
	if err != nil {
		return nil, err
	}

	reader := io.LimitedReader{source, requiredBytes}
	blockData = make([]byte, requiredBytes)
	_, err = reader.Read(blockData)

	if err != nil {
		return nil, err
	}

	return
}

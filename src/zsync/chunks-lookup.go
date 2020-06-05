package zsync

import (
	"appimage-update/src/zsync/chunks"
	"appimage-update/src/zsync/circularbuffer"
	"github.com/schollz/progressbar/v3"
	"io"
	"math"
)

type ChunkLookupSlice struct {
	chunkOffset         int64
	chunkSize           int64
	chunksSize          int64
	chunksCount         int64
	lastFullChunkOffset int64
	fileSize            int64
	file                io.ReadSeeker
	buffer              *circularbuffer.C2
}

func NewChunkLookupSlice(file io.ReadSeeker, chunkSize int64) (*ChunkLookupSlice, error) {
	fileSize, err := file.Seek(0, 2)
	if err != nil {
		return nil, err
	}
	_, err = file.Seek(0, 0)

	chunkCount := int64(math.Ceil(float64(fileSize) / float64(chunkSize)))

	lookupSlice := &ChunkLookupSlice{
		chunksSize:          chunkSize,
		chunksCount:         chunkCount,
		lastFullChunkOffset: (fileSize / chunkSize) * chunkSize,
		fileSize:            fileSize,
		file:                file,
		buffer:              circularbuffer.MakeC2Buffer(int(chunkSize)),
	}
	err = lookupSlice.readChunk()
	if err != nil {
		return nil, err
	}

	return lookupSlice, nil
}

func (s *ChunkLookupSlice) isEOF() bool {
	return s.chunkOffset > s.lastFullChunkOffset
}

func (s ChunkLookupSlice) getNextChunkSize() int64 {
	if s.chunkOffset+s.chunksSize > s.fileSize {
		return s.fileSize - s.chunkOffset
	} else {
		return s.chunksSize
	}
}

func (s *ChunkLookupSlice) consumeChunk() error {
	s.chunkOffset += s.chunksSize

	err := s.readChunk()
	return err
}

func (s *ChunkLookupSlice) consumeByte() error {
	s.chunkOffset += 1

	err := s.readByte()
	return err
}

func (s *ChunkLookupSlice) readByte() error {
	if s.chunkOffset+s.chunkSize > s.fileSize {
		s.chunkSize = s.fileSize - s.chunkOffset
	}

	_, err := io.CopyN(s.buffer, s.file, 1)
	if err == io.EOF {
		_, err = s.buffer.Write([]byte{1})
	}

	return nil
}

func (s *ChunkLookupSlice) readChunk() (err error) {
	n, err := io.CopyN(s.buffer, s.file, s.chunksSize)
	if err == io.EOF {
		zeroChunk := make([]byte, s.chunksSize-n)
		_, err = s.buffer.Write(zeroChunk)
	}

	s.chunkSize = n

	return
}

func (s *ChunkLookupSlice) getBlock() []byte {
	return s.buffer.GetBlock()
}

func (syncData *SyncData) identifyAllLocalMatchingChunks(matchingChunks []chunks.ChunkInfo) ([]chunks.ChunkInfo, error) {
	lookupSlice, err := NewChunkLookupSlice(syncData.Local, int64(syncData.BlockSize))
	if err != nil {
		return nil, err
	}

	progress := progressbar.DefaultBytes(
		lookupSlice.fileSize,
		"Searching reusable chunks: ",
	)

	for !lookupSlice.isEOF() {
		_ = progress.Set(int(lookupSlice.chunkOffset))

		data := lookupSlice.getBlock()

		matches := syncData.searchMatchingChunks(data)
		if matches != nil {
			matchingChunks = syncData.appendMatchingChunks(matchingChunks, matches, lookupSlice.chunkSize, lookupSlice.chunkOffset)
			err = lookupSlice.consumeChunk()
		} else {
			err = lookupSlice.consumeByte()
		}

		if err != nil {
			return nil, err
		}
	}
	_ = progress.Set(int(lookupSlice.fileSize))
	return matchingChunks, nil
}

func (syncData *SyncData) appendMatchingChunks(matchingChunks []chunks.ChunkInfo, matches []chunks.ChunkChecksum, chunkSize int64, offset int64) []chunks.ChunkInfo {
	for _, match := range matches {
		newChunk := chunks.ChunkInfo{
			Size:         chunkSize,
			Source:       syncData.Local,
			SourceOffset: offset,
			TargetOffset: int64(match.ChunkOffset * syncData.BlockSize),
		}

		// chop zero filled chunks at the end
		if newChunk.TargetOffset+newChunk.Size > syncData.FileLength {
			newChunk.Size = syncData.FileLength - newChunk.TargetOffset
		}
		matchingChunks = append(matchingChunks, newChunk)
	}
	return matchingChunks
}

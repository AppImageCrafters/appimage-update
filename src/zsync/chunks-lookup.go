package zsync

import (
	"appimage-update/src/zsync/chunks"
	"appimage-update/src/zsync/sources"
	"github.com/schollz/progressbar/v3"
	"io"
)

type ChunkLookupSlice struct {
	currentOffset int64
	ChunkSize     int64
	FileSize      int64
	file          io.ReadSeeker
}

func (syncData *SyncData) identifyAllLocalMatchingChunks(matchingChunks []chunks.ChunkInfo) ([]chunks.ChunkInfo, error) {
	lookup := int64(syncData.BlockSize)
	sourceFileSize, err := syncData.Local.Seek(0, 2)
	if err != nil {
		return nil, err
	}
	_, err = syncData.Local.Seek(0, 0)

	progress := progressbar.DefaultBytes(
		sourceFileSize,
		"Searching reusable chunks: ",
	)

	for offset := int64(0); offset < sourceFileSize; offset += lookup {
		_ = progress.Set(int(offset))
		chunkSize := int64(syncData.BlockSize)
		if offset+chunkSize > sourceFileSize {
			chunkSize = sourceFileSize - offset
		}

		data, err := sources.ReadChunk(syncData.Local, offset, chunkSize)
		if err != nil {
			return nil, err
		}

		if chunkSize < int64(syncData.BlockSize) {
			zeroChunk := make([]byte, int64(syncData.BlockSize)-chunkSize)
			data = append(data, zeroChunk...)
		}

		matches := syncData.searchMatchingChunks(data)
		if matches != nil {
			matchingChunks = syncData.appendMatchingChunks(matchingChunks, matches, chunkSize, offset)
			lookup = int64(syncData.BlockSize)
		} else {
			lookup = 1
		}
	}
	_ = progress.Set(int(sourceFileSize))
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

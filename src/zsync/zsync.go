package zsync

import (
	"appimage-update/src/zsync/chunks"
	"appimage-update/src/zsync/control"
	"appimage-update/src/zsync/rollsum"
	"appimage-update/src/zsync/sources"
	"fmt"
	"github.com/jinzhu/copier"
	"golang.org/x/crypto/md4"
	"hash"
	"io"
	"os"
	"sort"
)

type ReadSeeker interface {
	Read(b []byte) (n int, err error)
	Seek(offset int64, whence int) (int64, error)
}

type ChunkInfo struct {
	size         int64
	source       ReadSeeker
	sourceOffset int64
	targetOffset int64
}

type SyncData struct {
	control.Control

	weakChecksumBuilder   hash.Hash
	strongChecksumBuilder hash.Hash
	local                 *os.File
	output                io.Writer
}

func Sync(local *os.File, output io.Writer, control control.Control) (err error) {
	syncData := SyncData{
		Control:               control,
		weakChecksumBuilder:   rollsum.NewRollsum32(control.BlockSize),
		strongChecksumBuilder: md4.New(),
		local:                 local,
		output:                output,
	}

	matchingChunks, err := syncData.SearchLocalMatchingChunks()
	if err != nil {
		return err
	}
	missingChunks := syncData.IdentifyMissingChunks(matchingChunks)

	allChunks := append(matchingChunks, missingChunks...)
	sortChunksByTargetOffset(allChunks)

	for _, chunk := range allChunks {
		chunkData, err := readChunk(chunk.source, chunk.sourceOffset, chunk.size)
		if err != nil {
			return err
		}
		_, err = output.Write(chunkData)
		if err != nil {
			return err
		}
	}

	return nil
}

func (syncData *SyncData) SearchLocalMatchingChunks() (matchingChunks []ChunkInfo, err error) {
	fmt.Println("Looking for reusable chunks")

	matchingChunks, err = syncData.identifyAllLocalMatchingChunks(matchingChunks)
	if err != nil {
		return nil, err
	}

	matchingChunks = removeDuplicatedChunks(matchingChunks)
	matchingChunks = removeSmallChunks(matchingChunks, syncData.FileLength)
	matchingChunks = squashMatchingChunks(matchingChunks)

	syncData.printMatchingChunksStatistics(matchingChunks)

	return
}

func (syncData *SyncData) printMatchingChunksStatistics(matchingChunks []ChunkInfo) {
	reusableChunksSize := int64(0)
	for _, chunk := range matchingChunks {
		reusableChunksSize += chunk.size
	}
	fmt.Printf("Reusable chunks found: %d %dKb (%d%%)\n",
		len(matchingChunks), reusableChunksSize/1024, reusableChunksSize*100/syncData.FileLength)
}

func removeSmallChunks(matchingChunks []ChunkInfo, length int64) (filteredChunks []ChunkInfo) {
	for _, chunk := range matchingChunks {
		if chunk.size > 1024 || chunk.targetOffset+chunk.size == length {
			filteredChunks = append(filteredChunks, chunk)
		}
	}

	return
}

func (syncData *SyncData) identifyAllLocalMatchingChunks(matchingChunks []ChunkInfo) ([]ChunkInfo, error) {
	lookup := int64(syncData.BlockSize)
	sourceFileSize, err := syncData.local.Seek(0, 2)
	if err != nil {
		return nil, err
	}
	_, err = syncData.local.Seek(0, 0)

	for offset := int64(0); offset < sourceFileSize; offset += lookup {
		chunkSize := int64(syncData.BlockSize)
		if offset+chunkSize > sourceFileSize {
			chunkSize = sourceFileSize - offset
		}

		data, err := readChunk(syncData.local, offset, chunkSize)
		if err != nil {
			return nil, err
		}

		if chunkSize < int64(syncData.BlockSize) {
			zeroChunk := make([]byte, int64(syncData.BlockSize)-chunkSize)
			data = append(data, zeroChunk...)
		}

		matches := syncData.searchMatchingChunks(data)
		if matches != nil {
			for _, match := range matches {
				newChunk := ChunkInfo{
					size:         chunkSize,
					source:       syncData.local,
					sourceOffset: offset,
					targetOffset: int64(match.ChunkOffset * syncData.BlockSize),
				}
				matchingChunks = append(matchingChunks, newChunk)
			}

			lookup = int64(syncData.BlockSize)
		} else {
			lookup = 1
		}
	}
	return matchingChunks, nil
}

func squashMatchingChunks(matchingChunks []ChunkInfo) (squashedChunks []ChunkInfo) {
	sortChunksBySourceOffset(matchingChunks)

	var currentChunk *ChunkInfo
	for _, chunk := range matchingChunks {
		if currentChunk == nil {
			_ = copier.Copy(&chunk, &currentChunk)
		} else {
			if areContiguousChunks(currentChunk, chunk) {
				currentChunk.size += chunk.size
			} else {
				squashedChunks = append(squashedChunks, *currentChunk)
				_ = copier.Copy(&chunk, &currentChunk)
			}
		}
	}

	if currentChunk != nil {
		squashedChunks = append(squashedChunks, *currentChunk)
		currentChunk = nil
	}
	return
}

func removeDuplicatedChunks(matchingChunks []ChunkInfo) []ChunkInfo {
	m := make(map[int64]ChunkInfo)
	for _, item := range matchingChunks {
		if _, ok := m[item.targetOffset]; ok {
			// prefer chunks with the same offset in both files
			if item.sourceOffset == item.targetOffset {
				m[item.targetOffset] = item
			}
		} else {
			m[item.targetOffset] = item
		}
	}

	var result []ChunkInfo
	for _, item := range m {
		result = append(result, item)
	}

	return result
}

func areContiguousChunks(currentChunk *ChunkInfo, chunk ChunkInfo) bool {
	return currentChunk.sourceOffset+currentChunk.size == chunk.sourceOffset &&
		currentChunk.targetOffset+currentChunk.size == chunk.targetOffset &&
		currentChunk.source == chunk.source
}

func sortChunksBySourceOffset(matchingChunks []ChunkInfo) {
	sort.Slice(matchingChunks, func(i, j int) bool {
		return matchingChunks[i].sourceOffset < matchingChunks[j].sourceOffset
	})
}

func sortChunksByTargetOffset(matchingChunks []ChunkInfo) {
	sort.Slice(matchingChunks, func(i, j int) bool {
		return matchingChunks[i].targetOffset < matchingChunks[j].targetOffset
	})
}

func (syncData *SyncData) searchMatchingChunks(blockData []byte) []chunks.ChunkChecksum {
	syncData.weakChecksumBuilder.Write(blockData)
	weakSum := syncData.weakChecksumBuilder.Sum(nil)
	weakMatches := syncData.ChecksumIndex.FindWeakChecksum2(weakSum)
	if weakMatches != nil {
		syncData.strongChecksumBuilder.Reset()
		syncData.strongChecksumBuilder.Write(blockData)
		strongSum := syncData.strongChecksumBuilder.Sum(nil)

		return syncData.ChecksumIndex.FindStrongChecksum2(strongSum, weakMatches)
	}

	return nil
}

func (syncData *SyncData) IdentifyMissingChunks(matchingChunks []ChunkInfo) (missing []ChunkInfo) {
	missingChunksSource := sources.HttpFileSource{syncData.URL, 0, syncData.FileLength}

	offset := int64(0)
	for _, chunk := range matchingChunks {
		if chunk.targetOffset != offset {
			missingChunk := ChunkInfo{
				size:         chunk.targetOffset - offset,
				source:       missingChunksSource,
				sourceOffset: offset,
				targetOffset: offset,
			}

			missing = append(missing, missingChunk)
		}
		offset = chunk.targetOffset + chunk.size
	}

	if offset < syncData.FileLength {
		missingChunk := ChunkInfo{
			size:         syncData.FileLength - offset,
			source:       missingChunksSource,
			sourceOffset: offset,
			targetOffset: offset,
		}

		missing = append(missing, missingChunk)
	}

	return
}

func readChunk(local ReadSeeker, offset int64, requiredBytes int64) (blockData []byte, err error) {
	_, err = local.Seek(offset, 0)
	if err != nil {
		return nil, err
	}

	reader := io.LimitedReader{local, requiredBytes}
	blockData = make([]byte, requiredBytes)
	_, err = reader.Read(blockData)

	if err != nil {
		return nil, err
	}

	return
}

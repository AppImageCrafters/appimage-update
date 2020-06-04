package zsync

import (
	"appimage-update/src/zsync/chunks"
	"appimage-update/src/zsync/control"
	"appimage-update/src/zsync/rollsum"
	"appimage-update/src/zsync/sources"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"github.com/schollz/progressbar/v3"
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

	syncData.printChunksSummary(matchingChunks)
	allChunks := syncData.IdentifyMissingChunks(matchingChunks)

	err = mergeChunks(allChunks, output, syncData)
	return nil
}

func mergeChunks(allChunks []ChunkInfo, output io.Writer, syncData SyncData) error {
	outputSHA1 := sha1.New()

	bar := progressbar.DefaultBytes(
		syncData.FileLength,
		"Merging chunks: ",
	)

	for _, chunk := range allChunks {
		chunkData, err := readChunk(chunk.source, chunk.sourceOffset, chunk.size)
		if err != nil {
			return err
		}
		_, err = output.Write(chunkData)
		if err != nil {
			return err
		}

		outputSHA1.Write(chunkData)
		_, _ = bar.Write(chunkData)
	}

	outputSHA1Sum := hex.EncodeToString(outputSHA1.Sum(nil))
	if outputSHA1Sum != syncData.SHA1 {
		return fmt.Errorf("output checksum don't match with the expected")
	}
	return nil
}

func (syncData *SyncData) SearchLocalMatchingChunks() (matchingChunks []ChunkInfo, err error) {
	matchingChunks, err = syncData.identifyAllLocalMatchingChunks(matchingChunks)
	if err != nil {
		return nil, err
	}

	matchingChunks = removeDuplicatedChunks(matchingChunks)
	matchingChunks = removeSmallChunks(matchingChunks, syncData.FileLength)

	return
}

func (syncData *SyncData) printChunksSummary(matchingChunks []ChunkInfo) {
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

				// chop zero filled chunks at the end
				if newChunk.targetOffset+newChunk.size > syncData.FileLength {
					newChunk.size = syncData.FileLength - newChunk.targetOffset
				}
				matchingChunks = append(matchingChunks, newChunk)
			}

			lookup = int64(syncData.BlockSize)
		} else {
			lookup = 1
		}
	}
	_ = progress.Set(int(sourceFileSize))
	return matchingChunks, nil
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
	sortChunksByTargetOffset(matchingChunks)
	missingChunksSource := sources.HttpFileSource{syncData.URL, 0, syncData.FileLength}

	offset := int64(0)
	for _, chunk := range matchingChunks {
		gapSize := chunk.targetOffset - offset
		if gapSize > 0 {
			if chunk.targetOffset != offset {
				missingChunk := ChunkInfo{
					size:         gapSize,
					source:       &missingChunksSource,
					sourceOffset: offset,
					targetOffset: offset,
				}

				missing = append(missing, missingChunk)
				offset += gapSize
			}
		}

		missing = append(missing, chunk)
		offset += chunk.size
	}

	if offset < syncData.FileLength {
		missingChunk := ChunkInfo{
			size:         syncData.FileLength - offset,
			source:       &missingChunksSource,
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

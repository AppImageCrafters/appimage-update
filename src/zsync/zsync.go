package zsync

import (
	"appimage-update/src/zsync/chunks"
	"appimage-update/src/zsync/control"
	chunks2 "appimage-update/src/zsync/reader"
	"appimage-update/src/zsync/rollsum"
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

type SyncData struct {
	control.Control

	WeakChecksumBuilder   hash.Hash
	StrongChecksumBuilder hash.Hash
	Local                 *os.File
	Output                io.Writer
}

func Sync(local *os.File, output io.Writer, control control.Control) (err error) {
	syncData := SyncData{
		Control:               control,
		WeakChecksumBuilder:   rollsum.NewRollsum32(control.BlockSize),
		StrongChecksumBuilder: md4.New(),
		Local:                 local,
		Output:                output,
	}

	matchingChunks, err := syncData.SearchLocalMatchingChunks()
	if err != nil {
		return err
	}

	syncData.printChunksSummary(matchingChunks)
	allChunks := syncData.IdentifyMissingChunks(matchingChunks)

	err = syncData.mergeChunks(allChunks, output)
	return nil
}

func (syncData *SyncData) mergeChunks(allChunks []chunks.ChunkInfo, output io.Writer) error {
	outputSHA1 := sha1.New()

	bar := progressbar.DefaultBytes(
		syncData.FileLength,
		"Merging chunks: ",
	)

	for _, chunk := range allChunks {
		chunkData, err := readChunk(chunk.Source, chunk.SourceOffset, chunk.Size)
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
		return fmt.Errorf("Output checksum don't match with the expected")
	}
	return nil
}

func (syncData *SyncData) SearchLocalMatchingChunks() (matchingChunks []chunks.ChunkInfo, err error) {
	matchingChunks, err = syncData.identifyAllLocalMatchingChunks(matchingChunks)
	if err != nil {
		return nil, err
	}

	matchingChunks = removeDuplicatedChunks(matchingChunks)
	matchingChunks = removeSmallChunks(matchingChunks, syncData.FileLength)

	return
}

func (syncData *SyncData) printChunksSummary(matchingChunks []chunks.ChunkInfo) {
	reusableChunksSize := int64(0)
	for _, chunk := range matchingChunks {
		reusableChunksSize += chunk.Size
	}
	fmt.Printf("Reusable chunks found: %d %dKb (%d%%)\n",
		len(matchingChunks), reusableChunksSize/1024, reusableChunksSize*100/syncData.FileLength)
}

func removeSmallChunks(matchingChunks []chunks.ChunkInfo, length int64) (filteredChunks []chunks.ChunkInfo) {
	for _, chunk := range matchingChunks {
		if chunk.Size > 1024 || chunk.TargetOffset+chunk.Size == length {
			filteredChunks = append(filteredChunks, chunk)
		}
	}

	return
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

		data, err := readChunk(syncData.Local, offset, chunkSize)
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

			lookup = int64(syncData.BlockSize)
		} else {
			lookup = 1
		}
	}
	_ = progress.Set(int(sourceFileSize))
	return matchingChunks, nil
}

func removeDuplicatedChunks(matchingChunks []chunks.ChunkInfo) []chunks.ChunkInfo {
	m := make(map[int64]chunks.ChunkInfo)
	for _, item := range matchingChunks {
		if _, ok := m[item.TargetOffset]; ok {
			// prefer chunks with the same offset in both files
			if item.SourceOffset == item.TargetOffset {
				m[item.TargetOffset] = item
			}
		} else {
			m[item.TargetOffset] = item
		}
	}

	var result []chunks.ChunkInfo
	for _, item := range m {
		result = append(result, item)
	}

	return result
}

func sortChunksByTargetOffset(matchingChunks []chunks.ChunkInfo) {
	sort.Slice(matchingChunks, func(i, j int) bool {
		return matchingChunks[i].TargetOffset < matchingChunks[j].TargetOffset
	})
}

func (syncData *SyncData) searchMatchingChunks(blockData []byte) []chunks.ChunkChecksum {
	syncData.WeakChecksumBuilder.Write(blockData)
	weakSum := syncData.WeakChecksumBuilder.Sum(nil)
	weakMatches := syncData.ChecksumIndex.FindWeakChecksum2(weakSum)
	if weakMatches != nil {
		syncData.StrongChecksumBuilder.Reset()
		syncData.StrongChecksumBuilder.Write(blockData)
		strongSum := syncData.StrongChecksumBuilder.Sum(nil)

		return syncData.ChecksumIndex.FindStrongChecksum2(strongSum, weakMatches)
	}

	return nil
}

func (syncData *SyncData) IdentifyMissingChunks(matchingChunks []chunks.ChunkInfo) (missing []chunks.ChunkInfo) {
	sortChunksByTargetOffset(matchingChunks)
	missingChunksSource := chunks2.HttpFileReader{syncData.URL, 0, syncData.FileLength}

	offset := int64(0)
	for _, chunk := range matchingChunks {
		gapSize := chunk.TargetOffset - offset
		if gapSize > 0 {
			if chunk.TargetOffset != offset {
				missingChunk := chunks.ChunkInfo{
					Size:         gapSize,
					Source:       &missingChunksSource,
					SourceOffset: offset,
					TargetOffset: offset,
				}

				missing = append(missing, missingChunk)
				offset += gapSize
			}
		}

		missing = append(missing, chunk)
		offset += chunk.Size
	}

	if offset < syncData.FileLength {
		missingChunk := chunks.ChunkInfo{
			Size:         syncData.FileLength - offset,
			Source:       &missingChunksSource,
			SourceOffset: offset,
			TargetOffset: offset,
		}

		missing = append(missing, missingChunk)
	}

	return
}

func readChunk(local chunks2.ReadSeeker, offset int64, requiredBytes int64) (blockData []byte, err error) {
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

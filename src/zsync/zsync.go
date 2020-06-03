package zsync

import (
	"appimage-update/src/zsync/chunks"
	"appimage-update/src/zsync/control"
	"appimage-update/src/zsync/rollsum"
	"fmt"
	"github.com/jinzhu/copier"
	"golang.org/x/crypto/md4"
	"hash"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
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

type HttpFileSource struct {
	url    string
	offset int64
	size   int64
}

func (h HttpFileSource) Read(b []byte) (n int, err error) {
	rangedRequest, err := http.NewRequest("GET", h.url, nil)

	if err != nil {
		return 0, fmt.Errorf("Error creating request for \"%v\": %v", h.url, err)
	}

	range_start := h.offset
	range_end := h.offset + int64(len(b)) - 1

	rangeSpecifier := fmt.Sprintf("bytes=%v-%v", range_start, range_end)
	rangedRequest.ProtoAtLeast(1, 1)
	rangedRequest.Header.Add("Range", rangeSpecifier)
	rangedRequest.Header.Add("Accept-Encoding", "identity")

	client := &http.Client{}
	rangedResponse, err := client.Do(rangedRequest)

	if err != nil {
		return 0, fmt.Errorf("Error executing request for \"%v\": %v", h.url, err)
	}

	defer rangedResponse.Body.Close()

	if rangedResponse.StatusCode == 404 {
		return 0, fmt.Errorf("url not found")
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
				h.url, range_start, range_end,
				err,
			)
		}

		if bytesRead != len(b) {
			err = fmt.Errorf(
				"Unexpected response length %v (%v): %v",
				h.url,
				bytesRead,
				len(b),
			)
		}

		return bytesRead, nil
	}
}

func (h HttpFileSource) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case 0:
		h.offset = offset
	case 1:
		h.offset += offset
	case 2:
		h.offset = h.size + offset
	default:
		return -1, fmt.Errorf("Unknown whence value: %d", whence)
	}

	return offset, nil
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
	fmt.Println("Reusable chunks found: ", len(matchingChunks))
	for _, chunk := range matchingChunks {
		fmt.Printf("Source offset: %d\n", chunk.sourceOffset)
		fmt.Printf("Target offset: %d\n", chunk.targetOffset)
		fmt.Printf("Size: %d\n", chunk.size)
	}

	matchingChunks = removeSmallChunks(matchingChunks, syncData.FileLength)
	matchingChunks = squashMatchingChunks(matchingChunks)

	return
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
	missingChunksSource := HttpFileSource{syncData.URL, 0, syncData.FileLength}

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

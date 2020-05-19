/*
Package indexbuilder provides a few shortbuts to building a checksum index by generating and then loading
the checksums, and building an index from that. It's potentially a sign that the responsibilities here need refactoring.
*/
package indexbuilder

import (
	"appimage-update/src/zsync/chunks"
	"appimage-update/src/zsync/filechecksum"
	"appimage-update/src/zsync/index"
	"bytes"
	"io"
)

// Generates an index from a reader
// This is mostly a utility function to avoid being overly verbose in tests that need
// an index to work, but don't want to construct one by hand in order to avoid the dependencies
// obviously this means that those tests are likely to fail if there are issues with any of the other
// modules, which is not ideal.
// TODO: move to util?
func BuildChecksumIndex(check *filechecksum.FileChecksumGenerator, r io.Reader) (
	fcheck []byte,
	i *index.ChecksumIndex,
	lookup filechecksum.ChecksumLookup,
	err error,
) {
	b := bytes.NewBuffer(nil)
	fcheck, err = check.GenerateChecksums(r, b)

	if err != nil {
		return
	}

	weakSize := check.WeakRollingHash.Size()
	strongSize := check.GetStrongHash().Size()
	readChunks, err := chunks.LoadChecksumsFromReader(b, weakSize, strongSize)

	if err != nil {
		return
	}

	i = index.MakeChecksumIndex(readChunks)
	lookup = chunks.StrongChecksumGetter(readChunks)

	return
}

func BuildIndexFromString(generator *filechecksum.FileChecksumGenerator, reference string) (
	fileCheckSum []byte,
	referenceIndex *index.ChecksumIndex,
	lookup filechecksum.ChecksumLookup,
	err error,
) {
	return BuildChecksumIndex(generator, bytes.NewBufferString(reference))
}

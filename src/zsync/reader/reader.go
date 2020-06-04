package chunks

type ReadSeeker interface {
	Read(b []byte) (n int, err error)
	Seek(offset int64, whence int) (int64, error)
}

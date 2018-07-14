package cas

import (
	"crypto/sha256"
	"io"
)

type WriteCloser interface {
	io.WriteCloser
	CloseWithError(err error)
}

type BlobAccess interface {
	Get(checksum [sha256.Size]byte, size uint64) (error, io.Reader)
	Put(checksum [sha256.Size]byte, size uint64) (error, WriteCloser)
}

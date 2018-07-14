package cas

import (
	"io"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
)

type WriteCloser interface {
	io.WriteCloser
	CloseWithError(err error)
}

type BlobAccess interface {
	Get(digest *remoteexecution.Digest) (io.Reader, error)
	Put(digest *remoteexecution.Digest) (WriteCloser, error)
	FindMissing(digests []*remoteexecution.Digest) ([]*remoteexecution.Digest, error)
}

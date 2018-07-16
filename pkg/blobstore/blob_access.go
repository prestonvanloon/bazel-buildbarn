package blobstore

import (
	"io"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
)

type WriteCloser interface {
	io.WriteCloser
	Abandon()
}

type BlobAccess interface {
	Get(instance string, digest *remoteexecution.Digest) (io.Reader, error)
	Put(instance string, digest *remoteexecution.Digest) (WriteCloser, error)
	FindMissing(instance string, digests []*remoteexecution.Digest) ([]*remoteexecution.Digest, error)
}

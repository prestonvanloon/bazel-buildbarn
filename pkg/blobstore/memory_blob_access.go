package blobstore

import (
	"bytes"
	"io"
	"sync"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type memoryBlobAccess struct {
	blobKeyer BlobKeyer
	lock      sync.RWMutex
	blobs     map[string][]byte
}

func NewMemoryBlobAccess(blobKeyer BlobKeyer) BlobAccess {
	return &memoryBlobAccess{
		blobKeyer: blobKeyer,
		blobs:     map[string][]byte{},
	}
}

func (ba *memoryBlobAccess) Get(instance string, digest *remoteexecution.Digest) (io.Reader, error) {
	key, err := ba.blobKeyer(instance, digest)
	if err != nil {
		return nil, err
	}
	ba.lock.RLock()
	blob, ok := ba.blobs[key]
	ba.lock.RUnlock()
	if !ok {
		return nil, status.Errorf(codes.NotFound, "Blob %s not found", key)
	}
	return bytes.NewReader(blob), nil
}

func (ba *memoryBlobAccess) Put(instance string, digest *remoteexecution.Digest) (WriteCloser, error) {
	key, err := ba.blobKeyer(instance, digest)
	if err != nil {
		return nil, err
	}
	return &memoryBlobWriter{
		key:        key,
		blobAccess: ba,
	}, nil
}

func (ba *memoryBlobAccess) FindMissing(instance string, digests []*remoteexecution.Digest) ([]*remoteexecution.Digest, error) {
	var missing []*remoteexecution.Digest
	ba.lock.RLock()
	defer ba.lock.RUnlock()
	for _, digest := range digests {
		key, err := ba.blobKeyer(instance, digest)
		if err != nil {
			return nil, err
		}
		if _, ok := ba.blobs[key]; !ok {
			missing = append(missing, digest)
		}
	}
	return missing, nil
}

type memoryBlobWriter struct {
	key        string
	data       []byte
	blobAccess *memoryBlobAccess
}

func (bw *memoryBlobWriter) Write(p []byte) (n int, err error) {
	bw.data = append(bw.data, p...)
	return len(p), nil
}

func (bw *memoryBlobWriter) Close() error {
	bw.blobAccess.lock.Lock()
	bw.blobAccess.blobs[bw.key] = bw.data
	bw.blobAccess.lock.Unlock()
	return nil
}

func (bw *memoryBlobWriter) Abandon() {
}

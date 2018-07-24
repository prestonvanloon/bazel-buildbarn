package blobstore

import (
	"bytes"
	"io"
	"io/ioutil"
	"sync"

	"github.com/EdSchouten/bazel-buildbarn/pkg/util"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type memoryBlobAccess struct {
	blobKeyer util.DigestKeyer
	lock      sync.RWMutex
	blobs     map[string][]byte
}

func NewMemoryBlobAccess(blobKeyer util.DigestKeyer) BlobAccess {
	return &memoryBlobAccess{
		blobKeyer: blobKeyer,
		blobs:     map[string][]byte{},
	}
}

func (ba *memoryBlobAccess) Get(instance string, digest *remoteexecution.Digest) io.ReadCloser {
	key, err := ba.blobKeyer(instance, digest)
	if err != nil {
		return &errorReader{err: err}
	}
	ba.lock.RLock()
	blob, ok := ba.blobs[key]
	ba.lock.RUnlock()
	if !ok {
		return &errorReader{err: status.Errorf(codes.NotFound, "Blob %s not found", key)}
	}
	return ioutil.NopCloser(bytes.NewReader(blob))
}

func (ba *memoryBlobAccess) Put(instance string, digest *remoteexecution.Digest, r io.Reader) error {
	key, err := ba.blobKeyer(instance, digest)
	if err != nil {
		return err
	}
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	ba.lock.Lock()
	ba.blobs[key] = data
	ba.lock.Unlock()
	return nil
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

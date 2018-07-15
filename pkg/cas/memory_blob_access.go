package cas

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
)

func digestToKey(digest *remoteexecution.Digest) string {
	return fmt.Sprintf("%s/%d", digest.Hash, digest.SizeBytes)
}

type memoryBlobAccess struct {
	lock  sync.RWMutex
	blobs map[string][]byte
}

var _ BlobAccess = (*memoryBlobAccess)(nil)

func NewMemoryBlobAccess() BlobAccess {
	return &memoryBlobAccess{}
}

func (ba *memoryBlobAccess) Get(digest *remoteexecution.Digest) (io.Reader, error) {
	key := digestToKey(digest)
	ba.lock.RLock()
	blob, ok := ba.blobs[key]
	ba.lock.RUnlock()
	if !ok {
		return nil, fmt.Errorf("Blob %s not found", key)
	}
	return bytes.NewReader(blob), nil
}

func (ba *memoryBlobAccess) Put(digest *remoteexecution.Digest) (WriteCloser, error) {
	// TODO(edsch): Implement!
	return nil, fmt.Errorf("Not implemented!")
}

func (ba *memoryBlobAccess) FindMissing(digests []*remoteexecution.Digest) ([]*remoteexecution.Digest, error) {
	var missing []*remoteexecution.Digest
	ba.lock.RLock()
	for _, digest := range digests {
		if _, ok := ba.blobs[digestToKey(digest)]; !ok {
			missing = append(missing, digest)
		}
	}
	ba.lock.RUnlock()
	return missing, nil
}

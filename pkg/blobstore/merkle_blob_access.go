package blobstore

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
)

// extractDigest validates the format of fields in a Digest object and returns them.
func extractDigest(digest *remoteexecution.Digest) ([sha256.Size]byte, uint64, error) {
	var checksumBytes [sha256.Size]byte
	checksum, err := hex.DecodeString(digest.Hash)
	if err != nil {
		return checksumBytes, 0, err
	}
	if len(checksum) != sha256.Size {
		return checksumBytes, 0, fmt.Errorf("Expected checksum to be %d bytes; not %d", sha256.Size, len(checksum))
	}
	if digest.SizeBytes < 0 {
		return checksumBytes, 0, fmt.Errorf("Invalid negative size: %d", digest.SizeBytes)
	}
	copy(checksumBytes[:], checksum)
	return checksumBytes, uint64(digest.SizeBytes), nil
}

type merkleBlobAccess struct {
	blobAccess BlobAccess
}

func NewMerkleBlobAccess(blobAccess BlobAccess) BlobAccess {
	return &merkleBlobAccess{
		blobAccess: blobAccess,
	}
}

func (ba *merkleBlobAccess) Get(instance string, digest *remoteexecution.Digest) io.ReadCloser {
	checksum, size, err := extractDigest(digest)
	if err != nil {
		return &errorReader{err: err}
	}
	return &checksumValidatingReader{
		reader:   ba.blobAccess.Get(instance, digest),
		checksum: checksum,
		sizeLeft: size,
	}
}

func (ba *merkleBlobAccess) Put(instance string, digest *remoteexecution.Digest, r io.Reader) error {
	checksum, size, err := extractDigest(digest)
	if err != nil {
		return err
	}
	return ba.blobAccess.Put(instance, digest, &checksumValidatingReader{
		reader:   r,
		checksum: checksum,
		sizeLeft: size,
	})
}

func (ba *merkleBlobAccess) FindMissing(instance string, digests []*remoteexecution.Digest) ([]*remoteexecution.Digest, error) {
	for _, digest := range digests {
		_, _, err := extractDigest(digest)
		if err != nil {
			return nil, err
		}
	}
	return ba.blobAccess.FindMissing(instance, digests)
}

type checksumValidatingReader struct {
	reader   io.Reader
	checksum [sha256.Size]byte
	sizeLeft uint64
}

func (r *checksumValidatingReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	nLen := uint64(n)
	if nLen > r.sizeLeft {
		return 0, fmt.Errorf("Blob is %d bytes longer than expected", nLen-r.sizeLeft)
	}
	r.sizeLeft -= nLen

	if err == io.EOF {
		if r.sizeLeft != 0 {
			err := fmt.Errorf("Blob is %d bytes shorter than expected", r.sizeLeft)
			return 0, err
		}
		// TODO(edsch): Validate checksum.
	}
	return n, err
}

func (r *checksumValidatingReader) Close() error {
	return r.reader.(io.ReadCloser).Close()
}

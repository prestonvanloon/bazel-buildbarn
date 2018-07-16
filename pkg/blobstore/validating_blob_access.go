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

type validatingBlobAccess struct {
	blobAccess BlobAccess
}

var _ BlobAccess = (*validatingBlobAccess)(nil)

func NewValidatingBlobAccess(blobAccess BlobAccess) BlobAccess {
	return &validatingBlobAccess{
		blobAccess: blobAccess,
	}
}

func (ba *validatingBlobAccess) Get(instance string, digest *remoteexecution.Digest) (io.Reader, error) {
	checksum, size, err := extractDigest(digest)
	if err != nil {
		return nil, err
	}
	r, err := ba.blobAccess.Get(instance, digest)
	if err != nil {
		return nil, err
	}
	vr := validatingReader{
		reader:   r,
		checksum: checksum,
		sizeLeft: size,
	}
	return &vr, nil
}

func (ba *validatingBlobAccess) Put(instance string, digest *remoteexecution.Digest) (WriteCloser, error) {
	checksum, size, err := extractDigest(digest)
	if err != nil {
		return nil, err
	}
	w, err := ba.blobAccess.Put(instance, digest)
	if err != nil {
		return nil, err
	}
	return &validatingWriter{
		writer:   w,
		checksum: checksum,
		sizeLeft: size,
	}, nil
}

func (ba *validatingBlobAccess) FindMissing(instance string, digests []*remoteexecution.Digest) ([]*remoteexecution.Digest, error) {
	for _, digest := range digests {
		_, _, err := extractDigest(digest)
		if err != nil {
			return nil, err
		}
	}
	return ba.blobAccess.FindMissing(instance, digests)
}

type validatingReader struct {
	reader   io.Reader
	checksum [sha256.Size]byte
	sizeLeft uint64
}

func (r *validatingReader) Read(p []byte) (int, error) {
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

type validatingWriter struct {
	writer   WriteCloser
	checksum [sha256.Size]byte
	sizeLeft uint64
}

func (w *validatingWriter) Write(p []byte) (int, error) {
	// TODO(edsch): Update checksum.
	if pLen := uint64(len(p)); pLen > w.sizeLeft {
		return 0, fmt.Errorf("Attempted to write %d bytes too many", pLen-w.sizeLeft)
	}
	n, err := w.writer.Write(p)
	w.sizeLeft -= uint64(n)
	return n, err
}

func (w *validatingWriter) Close() error {
	// TODO(edsch): Validate checksum.
	if w.sizeLeft != 0 {
		w.writer.Abandon()
		return fmt.Errorf("Blob is %d bytes shorter than expected", w.sizeLeft)
	}
	return w.writer.Close()
}

func (w *validatingWriter) Abandon() {
	w.writer.Abandon()
}

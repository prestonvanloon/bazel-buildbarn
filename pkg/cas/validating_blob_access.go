package cas

import (
	"crypto/sha256"
	"fmt"
	"io"
)

type ValidatingBlobAccess struct {
	blobAccess BlobAccess
}

func (ba *ValidatingBlobAccess) Get(checksum [sha256.Size]byte, size uint64) (error, io.Reader) {
	err, r := ba.blobAccess.Get(checksum, size)
	if err != nil {
		return err, nil
	}
	return nil, &validatingReader{
		reader:   r,
		checksum: checksum,
		sizeLeft: size,
	}
}

func (ba *ValidatingBlobAccess) Put(checksum [sha256.Size]byte, size uint64) (error, WriteCloser) {
	err, w := ba.blobAccess.Put(checksum, size)
	if err != nil {
		return err, nil
	}
	return nil, &validatingWriter{
		writer:   w,
		checksum: checksum,
		sizeLeft: size,
	}
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
		return 0, fmt.Errorf("Blob is %u bytes longer than expected", nLen - r.sizeLeft)
	}
	r.sizeLeft -= nLen

	if err == io.EOF {
		if r.sizeLeft != 0 {
			err := fmt.Errorf("Blob is %u bytes shorter than expected", r.sizeLeft)
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
		return 0, fmt.Errorf("Attempted to write %u bytes too many", pLen - w.sizeLeft)
	}
	n, err := w.writer.Write(p)
	w.sizeLeft -= uint64(n)
	return n, err
}

func (w *validatingWriter) Close() error {
	// TODO(edsch): Validate checksum.
	if w.sizeLeft != 0 {
		err := fmt.Errorf("Blob is %u bytes shorter than expected", w.sizeLeft)
		w.writer.CloseWithError(err)
		return err
	}
	return w.writer.Close()
}

func (w *validatingWriter) CloseWithError(err error) {
	w.writer.CloseWithError(err)
}

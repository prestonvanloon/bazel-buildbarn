package blobstore

import (
	"errors"
	"io"

	"github.com/EdSchouten/bazel-buildbarn/pkg/util"

	"golang.org/x/net/context"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
)

type pendingFile struct {
	digest *remoteexecution.Digest
	r      io.ReadCloser
}

type BatchWritingBlobAccess struct {
	BlobAccess

	instance     *string
	pendingFiles map[string]*pendingFile
}

func NewBatchWritingBlobAccess(base BlobAccess) *BatchWritingBlobAccess {
	return &BatchWritingBlobAccess{
		BlobAccess:   base,
		pendingFiles: map[string]*pendingFile{},
	}
}

func (ba *BatchWritingBlobAccess) Put(ctx context.Context, instance string, digest *remoteexecution.Digest, r io.ReadCloser) error {
	// TODO(edsch): Make the threshold configurable?
	if len(ba.pendingFiles) >= 250 {
		if err := ba.Flush(ctx); err != nil {
			r.Close()
			return err
		}
	}

	if ba.instance != nil && *ba.instance != instance {
		r.Close()
		return errors.New("Attempted to mix blobs between namespaces, which disallows the use of FindMissing")
	}
	key, err := util.KeyDigestWithoutInstance(instance, digest)
	if err != nil {
		r.Close()
		return err
	}
	if _, ok := ba.pendingFiles[key]; ok {
		r.Close()
		return nil
	}

	ba.instance = &instance
	ba.pendingFiles[key] = &pendingFile{
		digest: digest,
		r:      r,
	}
	return nil
}

func (ba *BatchWritingBlobAccess) Flush(ctx context.Context) error {
	if len(ba.pendingFiles) == 0 {
		return nil
	}

	// Reset to the initial state upon completion.
	defer func() {
		for _, pendingFile := range ba.pendingFiles {
			pendingFile.r.Close()
		}
		ba.instance = nil
		ba.pendingFiles = map[string]*pendingFile{}
	}()

	// Figure out which files aren't present yet.
	var digests []*remoteexecution.Digest
	for _, pendingFile := range ba.pendingFiles {
		digests = append(digests, pendingFile.digest)
	}
	missing, err := ba.BlobAccess.FindMissing(ctx, *ba.instance, digests)
	if err != nil {
		return err
	}

	// Upload all files that were missing.
	for _, digest := range missing {
		key, err := util.KeyDigestWithoutInstance(*ba.instance, digest)
		if err != nil {
			return err
		}
		pendingFile, ok := ba.pendingFiles[key]
		if !ok {
			return errors.New("FindMissing returned a digest that was not requested")
		}
		err = ba.Put(ctx, *ba.instance, pendingFile.digest, pendingFile.r)
		delete(ba.pendingFiles, key)
		if err != nil {
			return err
		}
	}
	return nil
}

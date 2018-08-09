package blobstore

import (
	"context"
	"io"

	"cloud.google.com/go/storage"
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
)

type gsBlobAccess struct {
	blobKeyer  util.DigestKeyer
	bucketName string
}

func NewGsBlobAccess(bucketName string, blobKeyer util.DigestKeyer) BlobAccess {
	return &gsBlobAccess{
		blobKeyer:  blobKeyer,
		bucketName: bucketName,
	}
}

func configureStorage(ctx context.Context, bucketName string) (*storage.BucketHandle, error) {
	return nil, nil
}

func (ba *gsBlobAccess) object(ctx context.Context, instance string, digest *remoteexecution.Digest) (*storage.ObjectHandle, error) {
	bkt, err := configureStorage(ctx, ba.bucketName)
	if err != nil {
		return nil, err
	}
	key, err := ba.blobKeyer(instance, digest)
	if err != nil {
		return nil, err
	}
	return bkt.Object(key), nil
}

func (ba *gsBlobAccess) Get(ctx context.Context, instance string, digest *remoteexecution.Digest) io.ReadCloser {
	obj, err := ba.object(ctx, instance, digest)
	if err != nil {
		return &errorReader{err: err}
	}
	r, err := obj.NewReader(ctx)
	if err != nil {
		return &errorReader{err: err}		
	}
	return r
}

func (ba *gsBlobAccess) Put(ctx context.Context, instance string, digest *remoteexecution.Digest, r io.ReadCloser) error {
	obj, err := ba.object(ctx, instance, digest)
	if err != nil {
		return err
	}
	w := obj.NewWriter(ctx)
	_, err = io.Copy(w, r)
	return err
}

func (ba *gsBlobAccess) FindMissing(ctx context.Context, instance string, digests []*remoteexecution.Digest) ([]*remoteexecution.Digest, error) {
	var missing []*remoteexecution.Digest
	for _, digest := range digests {
		obj, err := ba.object(ctx, instance, digest)
		if err != nil {
			return nil, err
		}
		_, err = obj.Attrs(ctx)
		if err == storage.ErrObjectNotExist {
			missing = append(missing, digest)
			continue
		}
		if err != nil {
			return nil, err
		}
	}
	return missing, nil
}

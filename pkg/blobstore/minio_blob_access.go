package blobstore

import (
	"io"

	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	"github.com/minio/minio-go"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
)

type minioBlobAccess struct {
	client     *minio.Client
	bucketName string
	blobKeyer  util.DigestKeyer
}

func NewMinioBlobAccess(client *minio.Client, bucketName string, blobKeyer util.DigestKeyer) BlobAccess {
	return &minioBlobAccess{
		client:     client,
		bucketName: bucketName,
		blobKeyer:  blobKeyer,
	}
}

func (ba *minioBlobAccess) Get(instance string, digest *remoteexecution.Digest) (io.ReadCloser, error) {
	key, err := ba.blobKeyer(instance, digest)
	if err != nil {
		return nil, err
	}
	return ba.client.GetObject(ba.bucketName, key, minio.GetObjectOptions{})
}

func (ba *minioBlobAccess) Put(instance string, digest *remoteexecution.Digest, r io.Reader) error {
	key, err := ba.blobKeyer(instance, digest)
	if err != nil {
		return err
	}
	_, err = ba.client.PutObject(ba.bucketName, key, r, -1, minio.PutObjectOptions{})
	return err
}

func (ba *minioBlobAccess) FindMissing(instance string, digests []*remoteexecution.Digest) ([]*remoteexecution.Digest, error) {
	var missing []*remoteexecution.Digest
	for _, digest := range digests {
		key, err := ba.blobKeyer(instance, digest)
		if err != nil {
			return nil, err
		}
		// TODO(edsch): Bail out upon failures.
		_, err = ba.client.StatObject(ba.bucketName, key, minio.StatObjectOptions{})
		if err == nil {
			missing = append(missing, digest)
		}
	}
	return missing, nil
}

package blobstore

import (
	"io"

	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	"github.com/minio/minio-go"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func convertMinioError(err error) error {
	if err != nil {
		if errorResponse := minio.ToErrorResponse(err); errorResponse.StatusCode == 404 {
			err = status.Errorf(codes.NotFound, errorResponse.Message)
		}
	}
	return err
}

type minioReader struct {
	io.ReadCloser
}

func (r *minioReader) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	return n, convertMinioError(err)
}

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

func (ba *minioBlobAccess) Get(instance string, digest *remoteexecution.Digest) io.ReadCloser {
	key, err := ba.blobKeyer(instance, digest)
	if err != nil {
		return &errorReader{err: err}
	}
	r, err := ba.client.GetObject(ba.bucketName, key, minio.GetObjectOptions{})
	if err != nil {
		return &errorReader{err: convertMinioError(err)}
	}
	return &minioReader{ReadCloser: r}
}

func (ba *minioBlobAccess) Put(instance string, digest *remoteexecution.Digest, r io.Reader) error {
	key, err := ba.blobKeyer(instance, digest)
	if err != nil {
		return err
	}
	_, err = ba.client.PutObject(ba.bucketName, key, r, -1, minio.PutObjectOptions{})
	return convertMinioError(err)
}

func (ba *minioBlobAccess) FindMissing(instance string, digests []*remoteexecution.Digest) ([]*remoteexecution.Digest, error) {
	var missing []*remoteexecution.Digest
	for _, digest := range digests {
		key, err := ba.blobKeyer(instance, digest)
		if err != nil {
			return nil, err
		}
		_, err = ba.client.StatObject(ba.bucketName, key, minio.StatObjectOptions{})
		if err != nil {
			err = convertMinioError(err)
			if status.Code(err) == codes.NotFound {
				missing = append(missing, digest)
			} else {
				return nil, err
			}
		}
	}
	return missing, nil
}

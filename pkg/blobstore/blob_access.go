package blobstore

import (
	"bytes"
	"io"
	"io/ioutil"

	"github.com/golang/protobuf/proto"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
)

type BlobAccess interface {
	Get(instance string, digest *remoteexecution.Digest) (io.ReadCloser, error)
	Put(instance string, digest *remoteexecution.Digest, r io.Reader) error
	FindMissing(instance string, digests []*remoteexecution.Digest) ([]*remoteexecution.Digest, error)
}

func GetMessageFromBlobAccess(blobAccess BlobAccess, instance string, digest *remoteexecution.Digest, out proto.Message) error {
	r, err := blobAccess.Get(instance, digest)
	if err != nil {
		return err
	}
	defer r.Close()

	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	return proto.Unmarshal(data, out)
}

func PutMessageToBlobAccess(blobAccess BlobAccess, instance string, digest *remoteexecution.Digest, in proto.Message) error {
	data, err := proto.Marshal(in)
	if err != nil {
		return err
	}
	return blobAccess.Put(instance, digest, bytes.NewBuffer(data))
}

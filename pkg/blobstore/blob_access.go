package blobstore

import (
	"io"
	"io/ioutil"

	"github.com/golang/protobuf/proto"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
)

type WriteCloser interface {
	io.WriteCloser
	Abandon()
}

type BlobAccess interface {
	// TODO(edsch): Should this be ReadCloser?
	Get(instance string, digest *remoteexecution.Digest) (io.Reader, error)
	Put(instance string, digest *remoteexecution.Digest) (WriteCloser, error)
	FindMissing(instance string, digests []*remoteexecution.Digest) ([]*remoteexecution.Digest, error)
}

func GetMessageFromBlobAccess(blobAccess BlobAccess, instance string, digest *remoteexecution.Digest, out proto.Message) error {
	r, err := blobAccess.Get(instance, digest)
	if err != nil {
		return err
	}
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
	w, err := blobAccess.Put(instance, digest)
	if err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		w.Abandon()
		return err
	}
	return w.Close()
}

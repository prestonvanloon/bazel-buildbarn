package main

import (
	"errors"
	"log"

	"github.com/EdSchouten/bazel-buildbarn/pkg/cas"

	"golang.org/x/net/context"
	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
)

type contentAddressableStorageServer struct {
	blobAccess cas.BlobAccess
}

func NewContentAddressableStorageServer(blobAccess cas.BlobAccess) remoteexecution.ContentAddressableStorageServer {
	return &contentAddressableStorageServer{
		blobAccess: blobAccess,
	}
}

func (s *contentAddressableStorageServer) FindMissingBlobs(ctx context.Context, in *remoteexecution.FindMissingBlobsRequest) (*remoteexecution.FindMissingBlobsResponse, error) {
	log.Print("Attempted to call ContentAddressableStorage.FindMissingBlobs")
	digests, err := s.blobAccess.FindMissing(in.BlobDigests)
	if err != nil {
		return nil, err
	}
	return &remoteexecution.FindMissingBlobsResponse{
		MissingBlobDigests: digests,
	}, nil
}

func (s *contentAddressableStorageServer) BatchUpdateBlobs(ctx context.Context, in *remoteexecution.BatchUpdateBlobsRequest) (*remoteexecution.BatchUpdateBlobsResponse, error) {
	log.Print("Attempted to call ContentAddressableStorage.BatchUpdateBlobs")
	return nil, errors.New("Fail!")
}

func (s *contentAddressableStorageServer) GetTree(ctx context.Context, in *remoteexecution.GetTreeRequest) (*remoteexecution.GetTreeResponse, error) {
	log.Print("Attempted to call ContentAddressableStorage.GetTree")
	return nil, errors.New("Fail!")
}

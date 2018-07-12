package main

import (
	"errors"
	"log"

	"golang.org/x/net/context"
	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
)

type ContentAddressableStorageServer struct {
}

func (s *ContentAddressableStorageServer) FindMissingBlobs(ctx context.Context, in *remoteexecution.FindMissingBlobsRequest) (*remoteexecution.FindMissingBlobsResponse, error) {
	log.Print("Attempted to call ContentAddressableStorage.FindMissingBlobs")
	return &remoteexecution.FindMissingBlobsResponse{
		MissingBlobDigests: in.BlobDigests,
	}, nil
}

func (s *ContentAddressableStorageServer) BatchUpdateBlobs(ctx context.Context, in *remoteexecution.BatchUpdateBlobsRequest) (*remoteexecution.BatchUpdateBlobsResponse, error) {
	log.Print("Attempted to call ContentAddressableStorage.BatchUpdateBlobs")
	return nil, errors.New("Fail!")
}

func (s *ContentAddressableStorageServer) GetTree(ctx context.Context, in *remoteexecution.GetTreeRequest) (*remoteexecution.GetTreeResponse, error) {
	log.Print("Attempted to call ContentAddressableStorage.GetTree")
	return nil, errors.New("Fail!")
}

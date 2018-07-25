package blobstore

import (
	"log"

	"golang.org/x/net/context"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type contentAddressableStorageServer struct {
	contentAddressableStorage BlobAccess
}

func NewContentAddressableStorageServer(contentAddressableStorage BlobAccess) remoteexecution.ContentAddressableStorageServer {
	return &contentAddressableStorageServer{
		contentAddressableStorage: contentAddressableStorage,
	}
}

func (s *contentAddressableStorageServer) FindMissingBlobs(ctx context.Context, in *remoteexecution.FindMissingBlobsRequest) (*remoteexecution.FindMissingBlobsResponse, error) {
	log.Print("Attempted to call ContentAddressableStorage.FindMissingBlobs")
	digests, err := s.contentAddressableStorage.FindMissing(in.InstanceName, in.BlobDigests)
	if err != nil {
		return nil, err
	}
	return &remoteexecution.FindMissingBlobsResponse{
		MissingBlobDigests: digests,
	}, nil
}

func (s *contentAddressableStorageServer) BatchUpdateBlobs(ctx context.Context, in *remoteexecution.BatchUpdateBlobsRequest) (*remoteexecution.BatchUpdateBlobsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "This service does not support batched uploading of blobs")
}

func (s *contentAddressableStorageServer) GetTree(ctx context.Context, in *remoteexecution.GetTreeRequest) (*remoteexecution.GetTreeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "This service does not support downloading directory trees")
}

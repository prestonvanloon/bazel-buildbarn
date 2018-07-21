package main

import (
	"log"
	"net"

	"github.com/EdSchouten/bazel-buildbarn/pkg/blobstore"
	"github.com/EdSchouten/bazel-buildbarn/pkg/builder"
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"

	"google.golang.org/genproto/googleapis/bytestream"
	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	watcher "google.golang.org/genproto/googleapis/watcher/v1"
	"google.golang.org/grpc"
)

func main() {
	sock, err := net.Listen("tcp", ":8980")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	contentAddressableStorage := blobstore.NewMerkleBlobAccess(blobstore.NewMemoryBlobAccess(util.KeyDigestWithoutInstance))
	buildExecutor := builder.NewLocalBuildExecutor(contentAddressableStorage)
	actionCache := blobstore.NewMemoryBlobAccess(util.KeyDigestWithInstance)
	buildQueue := builder.NewCachedBuildQueue(actionCache, builder.NewSynchronousBuildQueue(buildExecutor, util.KeyDigestWithInstance))

	s := grpc.NewServer()
	remoteexecution.RegisterActionCacheServer(s, NewActionCacheServer(actionCache))
	remoteexecution.RegisterContentAddressableStorageServer(s, NewContentAddressableStorageServer(contentAddressableStorage))
	bytestream.RegisterByteStreamServer(s, NewByteStreamServer(contentAddressableStorage))
	remoteexecution.RegisterExecutionServer(s, buildQueue)
	watcher.RegisterWatcherServer(s, buildQueue)
	if err := s.Serve(sock); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

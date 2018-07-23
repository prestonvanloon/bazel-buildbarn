package main

import (
	"log"
	"net"
	"os"
	"syscall"

	"github.com/EdSchouten/bazel-buildbarn/pkg/blobstore"
	"github.com/EdSchouten/bazel-buildbarn/pkg/builder"
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"

	"google.golang.org/genproto/googleapis/bytestream"
	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	watcher "google.golang.org/genproto/googleapis/watcher/v1"
	"google.golang.org/grpc"
)

func main() {
	// Respect file permissions that we pass to os.OpenFile(), os.Mkdir(), etc.
	syscall.Umask(0)

	sock, err := net.Listen("tcp", ":8980")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	contentAddressableStorage := blobstore.NewMerkleBlobAccess(blobstore.NewMemoryBlobAccess(util.KeyDigestWithoutInstance))
	if err := os.Mkdir("/cache", 0); err != nil {
		log.Fatal("Failed to create cache directory: ", err)
	}
	inputFileExposer := builder.NewCachedInputFileExposer(builder.NewUncachedInputFileExposer(contentAddressableStorage), util.KeyDigestWithoutInstance, "/cache")
	buildExecutor := builder.NewLocalBuildExecutor(contentAddressableStorage, inputFileExposer)
	actionCache := blobstore.NewMemoryBlobAccess(util.KeyDigestWithInstance)
	synchronousBuildQueue := builder.NewSynchronousBuildQueue(buildExecutor, util.KeyDigestWithInstance, 10)
	go synchronousBuildQueue.Run()
	buildQueue := builder.NewCachedBuildQueue(actionCache, synchronousBuildQueue)

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

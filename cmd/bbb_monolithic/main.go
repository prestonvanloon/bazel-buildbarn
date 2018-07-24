package main

import (
	"flag"
	"log"
	"net"
	"os"
	"syscall"

	"github.com/EdSchouten/bazel-buildbarn/pkg/blobstore"
	"github.com/EdSchouten/bazel-buildbarn/pkg/builder"
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	"github.com/minio/minio-go"

	"google.golang.org/genproto/googleapis/bytestream"
	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	watcher "google.golang.org/genproto/googleapis/watcher/v1"
	"google.golang.org/grpc"
)

func main() {
	var (
		minioEndpoint        = flag.String("minio-endpoint", "", "S3 compatible object storage endpoint for the Content Addressable Storage and the Action Cache")
		minioAccessKeyId     = flag.String("minio-access-key-id", "", "Access key for the object storage")
		minioSecretAccessKey = flag.String("minio-secret-access-key", "", "Secret key for the object storage")
		minioSsl             = flag.Bool("minio-ssl", false, "Whether to use HTTPS for the object storage")
	)
	flag.Parse()

	// Respect file permissions that we pass to os.OpenFile(), os.Mkdir(), etc.
	syscall.Umask(0)

	// Storage of content and actions.
	var contentAddressableStorage blobstore.BlobAccess
	var actionCache blobstore.BlobAccess
	if *minioEndpoint == "" {
		contentAddressableStorage = blobstore.NewMemoryBlobAccess(util.KeyDigestWithoutInstance)
		actionCache = blobstore.NewMemoryBlobAccess(util.KeyDigestWithInstance)
	} else {
		minioClient, err := minio.New(*minioEndpoint, *minioAccessKeyId, *minioSecretAccessKey, *minioSsl)
		if err != nil {
			log.Fatal("Failed to create Minio client: ", err)
		}
		contentAddressableStorage = blobstore.NewMinioBlobAccess(minioClient, "content-addressable-storage", util.KeyDigestWithoutInstance)
		actionCache = blobstore.NewMinioBlobAccess(minioClient, "action-cache", util.KeyDigestWithInstance)
	}
	contentAddressableStorage = blobstore.NewMerkleBlobAccess(contentAddressableStorage)

	// On-disk caching of content for efficient linking into build environments.
	if err := os.Mkdir("/cache", 0); err != nil {
		log.Fatal("Failed to create cache directory: ", err)
	}
	inputFileExposer := builder.NewHardlinkingInputFileExposer(builder.NewBlobAccessInputFileExposer(contentAddressableStorage), util.KeyDigestWithoutInstance, "/cache", 10000, 1<<30)

	buildExecutor := builder.NewCachingBuildExecutor(builder.NewLocalBuildExecutor(contentAddressableStorage, inputFileExposer), actionCache)
	synchronousBuildQueue := builder.NewSynchronousBuildQueue(buildExecutor, util.KeyDigestWithInstance, 10)
	go synchronousBuildQueue.Run()
	buildQueue := builder.NewCachedBuildQueue(actionCache, synchronousBuildQueue)

	sock, err := net.Listen("tcp", ":8980")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
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

package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"syscall"

	"github.com/EdSchouten/bazel-buildbarn/pkg/blobstore"
	"github.com/EdSchouten/bazel-buildbarn/pkg/builder"
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"google.golang.org/genproto/googleapis/bytestream"
	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	watcher "google.golang.org/genproto/googleapis/watcher/v1"
	"google.golang.org/grpc"
)

func main() {
	var (
		s3Endpoint        = flag.String("s3-endpoint", "", "S3 compatible object storage endpoint for the Content Addressable Storage and the Action Cache")
		s3AccessKeyId     = flag.String("s3-access-key-id", "", "Access key for the object storage")
		s3SecretAccessKey = flag.String("s3-secret-access-key", "", "Secret key for the object storage")
		s3Region          = flag.String("s3-region", "", "Region of the object storage")
		s3DisableSsl      = flag.Bool("s3-disable-ssl", false, "Whether to use HTTP for the object storage instead of HTTPS")
	)
	flag.Parse()

	// Respect file permissions that we pass to os.OpenFile(), os.Mkdir(), etc.
	syscall.Umask(0)

	// Web server for metrics and profiling.
	go func() {
		log.Println(http.ListenAndServe(":80", nil))
	}()

	// Storage of content and actions.
	var contentAddressableStorage blobstore.BlobAccess
	var actionCache blobstore.BlobAccess
	if *s3Endpoint == "" {
		contentAddressableStorage = blobstore.NewMemoryBlobAccess(util.KeyDigestWithoutInstance)
		actionCache = blobstore.NewMemoryBlobAccess(util.KeyDigestWithInstance)
	} else {
		// Create an S3 client. Set the uploader concurrency to 1 to drastically reduce memory usage.
		session := session.New(&aws.Config{
			Credentials:      credentials.NewStaticCredentials(*s3AccessKeyId, *s3SecretAccessKey, ""),
			Endpoint:         s3Endpoint,
			Region:           s3Region,
			DisableSSL:       s3DisableSsl,
			S3ForcePathStyle: aws.Bool(true),
		})
		s3 := s3.New(session)
		uploader := s3manager.NewUploader(session)
		uploader.Concurrency = 1

		contentAddressableStorage = blobstore.NewS3BlobAccess(s3, uploader, aws.String("content-addressable-storage"), util.KeyDigestWithoutInstance)
		actionCache = blobstore.NewS3BlobAccess(s3, uploader, aws.String("action-cache"), util.KeyDigestWithInstance)
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

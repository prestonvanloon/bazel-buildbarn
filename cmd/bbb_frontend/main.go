package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"

	"github.com/EdSchouten/bazel-buildbarn/pkg/ac"
	"github.com/EdSchouten/bazel-buildbarn/pkg/blobstore"
	"github.com/EdSchouten/bazel-buildbarn/pkg/builder"
	"github.com/EdSchouten/bazel-buildbarn/pkg/cas"
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/prometheus/client_golang/prometheus/promhttp"

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

		schedulerAddress = flag.String("scheduler-address", "", "Address at which the scheduler process is running")
	)
	flag.Parse()

	// Web server for metrics and profiling.
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		log.Fatal(http.ListenAndServe(":80", nil))
	}()

	// Create an S3 client. Set the uploader concurrency to 1 to drastically reduce memory usage.
	// TODO(edsch): Maybe the concurrency can be left alone for this process?
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

	// Storage of content and actions.
	contentAddressableStorageBlobAccess := blobstore.NewMetricsBlobAccess(
		blobstore.NewMerkleBlobAccess(
			blobstore.NewMetricsBlobAccess(
				blobstore.NewS3BlobAccess(s3, uploader, aws.String("content-addressable-storage"), util.KeyDigestWithoutInstance),
				"cas_s3")),
		"cas_merkle")
	actionCacheBlobAccess := blobstore.NewMetricsBlobAccess(
		blobstore.NewS3BlobAccess(s3, uploader, aws.String("action-cache"), util.KeyDigestWithInstance),
		"ac_s3")

	// Backend capable of compiling.
	// TODO(edsch): Pass in a list and demultiplex based on instance name.
	scheduler, err := grpc.Dial(*schedulerAddress, grpc.WithInsecure())
	if err != nil {
		log.Fatal("Failed to create scheduler RPC client: ", err)
	}
	buildQueue := builder.NewGrpcClientConnBuildQueue(scheduler)

	// RPC server.
	s := grpc.NewServer()
	remoteexecution.RegisterActionCacheServer(s, ac.NewActionCacheServer(ac.NewBlobAccessActionCache(actionCacheBlobAccess)))
	remoteexecution.RegisterContentAddressableStorageServer(s, cas.NewContentAddressableStorageServer(contentAddressableStorageBlobAccess))
	bytestream.RegisterByteStreamServer(s, blobstore.NewByteStreamServer(contentAddressableStorageBlobAccess))
	remoteexecution.RegisterExecutionServer(s, buildQueue)
	watcher.RegisterWatcherServer(s, buildQueue)

	sock, err := net.Listen("tcp", ":8980")
	if err != nil {
		log.Fatal("Failed to create listening socket: ", err)
	}
	if err := s.Serve(sock); err != nil {
		log.Fatal("Failed to serve RPC server: ", err)
	}
}

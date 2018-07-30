package main

import (
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"syscall"
	"time"

	"github.com/EdSchouten/bazel-buildbarn/pkg/ac"
	"github.com/EdSchouten/bazel-buildbarn/pkg/blobstore"
	"github.com/EdSchouten/bazel-buildbarn/pkg/builder"
	"github.com/EdSchouten/bazel-buildbarn/pkg/cas"
	"github.com/EdSchouten/bazel-buildbarn/pkg/proto/scheduler"
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func main() {
	var (
		s3Endpoint        = flag.String("s3-endpoint", "", "S3 compatible object storage endpoint for the Content Addressable Storage and the Action Cache")
		s3AccessKeyId     = flag.String("s3-access-key-id", "", "Access key for the object storage")
		s3SecretAccessKey = flag.String("s3-secret-access-key", "", "Secret key for the object storage")
		s3Region          = flag.String("s3-region", "", "Region of the object storage")
		s3DisableSsl      = flag.Bool("s3-disable-ssl", false, "Whether to use HTTP for the object storage instead of HTTPS")

		schedulerAddress = flag.String("scheduler", "", "Address of the scheduler to which to connect")
	)
	flag.Parse()

	// Respect file permissions that we pass to os.OpenFile(), os.Mkdir(), etc.
	syscall.Umask(0)

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
	batchWritingBlobAccess := blobstore.NewBatchWritingBlobAccess(
		blobstore.NewMetricsBlobAccess(
			blobstore.NewMerkleBlobAccess(
				blobstore.NewMetricsBlobAccess(
					blobstore.NewS3BlobAccess(s3, uploader, aws.String("content-addressable-storage"), util.KeyDigestWithoutInstance),
					"cas_s3")),
			"cas_merkle"))
	contentAddressableStorageBlobAccess := blobstore.NewMetricsBlobAccess(batchWritingBlobAccess, "cas_batch_writer")
	actionCacheBlobAccess := blobstore.NewMetricsBlobAccess(
		blobstore.NewS3BlobAccess(s3, uploader, aws.String("action-cache"), util.KeyDigestWithInstance),
		"ac_s3")

	// On-disk caching of content for efficient linking into build environments.
	if err := os.Mkdir("/cache", 0); err != nil {
		log.Fatal("Failed to create cache directory: ", err)
	}

	buildExecutor := builder.NewCachingBuildExecutor(
		builder.NewLocalBuildExecutor(
			cas.NewDirectoryCachingContentAddressableStorage(
				cas.NewHardlinkingContentAddressableStorage(
					cas.NewBlobAccessContentAddressableStorage(
						contentAddressableStorageBlobAccess),
					util.KeyDigestWithoutInstance, "/cache", 10000, 1<<30),
				util.KeyDigestWithoutInstance, 1000)),
		ac.NewBlobAccessActionCache(
			blobstore.NewMetricsBlobAccess(actionCacheBlobAccess, "ac_build_executor")))

	// Create connection with scheduler.
	schedulerConnection, err := grpc.Dial(
		*schedulerAddress,
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(grpc_prometheus.UnaryClientInterceptor),
		grpc.WithStreamInterceptor(grpc_prometheus.StreamClientInterceptor))
	if err != nil {
		log.Fatal("Failed to create scheduler RPC client: ", err)
	}
	schedulerClient := scheduler.NewSchedulerClient(schedulerConnection)

	// Repeatedly ask the scheduler for work.
	for {
		err := subscribeAndExecute(schedulerClient, buildExecutor, batchWritingBlobAccess)
		log.Print("Failed to subscribe and execute: ", err)
		time.Sleep(time.Second * 3)
	}
}

func subscribeAndExecute(schedulerClient scheduler.SchedulerClient, buildExecutor builder.BuildExecutor, batchWritingBlobAccess *blobstore.BatchWritingBlobAccess) error {
	stream, err := schedulerClient.GetWork(context.Background())
	if err != nil {
		return err
	}
	defer stream.CloseSend()
	ctx := stream.Context()

	for {
		request, err := stream.Recv()
		if err != nil {
			return err
		}
		log.Print("Request: ", request)
		response := buildExecutor.Execute(ctx, request)
		if err := batchWritingBlobAccess.Flush(ctx); err != nil {
			response = builder.ConvertErrorToExecuteResponse(err)
		}
		log.Print("Response: ", response)
		if err := stream.Send(response); err != nil {
			return err
		}
	}
}

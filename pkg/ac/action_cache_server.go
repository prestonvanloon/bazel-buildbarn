package ac

import (
	"log"

	"github.com/EdSchouten/bazel-buildbarn/pkg/blobstore"

	"golang.org/x/net/context"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type actionCacheServer struct {
	actionCache blobstore.BlobAccess
}

func NewActionCacheServer(actionCache blobstore.BlobAccess) remoteexecution.ActionCacheServer {
	return &actionCacheServer{
		actionCache: actionCache,
	}
}

func (s *actionCacheServer) GetActionResult(ctx context.Context, in *remoteexecution.GetActionResultRequest) (*remoteexecution.ActionResult, error) {
	var actionResult remoteexecution.ActionResult
	if err := blobstore.GetMessageFromBlobAccess(s.actionCache, ctx, in.InstanceName, in.ActionDigest, &actionResult); err != nil {
		log.Print("actionCacheServer.GetActionResult: ", err)
		return nil, err
	}
	return &actionResult, nil
}

func (s *actionCacheServer) UpdateActionResult(ctx context.Context, in *remoteexecution.UpdateActionResultRequest) (*remoteexecution.ActionResult, error) {
	return nil, status.Error(codes.PermissionDenied, "This service can only be used to get action results")
}

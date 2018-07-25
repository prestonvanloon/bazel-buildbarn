package ac

import (
	"golang.org/x/net/context"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type actionCacheServer struct {
	actionCache ActionCache
}

func NewActionCacheServer(actionCache ActionCache) remoteexecution.ActionCacheServer {
	return &actionCacheServer{
		actionCache: actionCache,
	}
}

func (s *actionCacheServer) GetActionResult(ctx context.Context, in *remoteexecution.GetActionResultRequest) (*remoteexecution.ActionResult, error) {
	return s.actionCache.GetActionResult(ctx, in.InstanceName, in.ActionDigest)
}

func (s *actionCacheServer) UpdateActionResult(ctx context.Context, in *remoteexecution.UpdateActionResultRequest) (*remoteexecution.ActionResult, error) {
	return nil, status.Error(codes.PermissionDenied, "This service can only be used to get action results")
}

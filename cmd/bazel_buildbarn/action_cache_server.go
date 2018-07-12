package main

import (
	"errors"
	"log"

	"golang.org/x/net/context"
	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ActionCacheServer struct {
}

func (s *ActionCacheServer) GetActionResult(ctx context.Context, in *remoteexecution.GetActionResultRequest) (*remoteexecution.ActionResult, error) {
	log.Print("Attempted to call ActionCache.GetActionResult")
	return nil, status.Error(codes.NotFound, "Fail!")
}

func (s *ActionCacheServer) UpdateActionResult(ctx context.Context, in *remoteexecution.UpdateActionResultRequest) (*remoteexecution.ActionResult, error) {
	log.Print("Attempted to call ActionCache.UpdateActionResult")
	return nil, errors.New("Fail!")
}

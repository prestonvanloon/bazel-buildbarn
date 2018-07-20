package main

import (
	"errors"
	"io/ioutil"
	"log"

	"github.com/EdSchouten/bazel-buildbarn/pkg/blobstore"
	"github.com/golang/protobuf/proto"

	"golang.org/x/net/context"
	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
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
	r, err := s.actionCache.Get(in.InstanceName, in.ActionDigest)
	if err != nil {
		log.Print("actionCacheServer.GetActionResult: ", err)
		return nil, err
	}
	actionResultData, err := ioutil.ReadAll(r)
	if err != nil {
		log.Print("actionCacheServer.GetActionResult: ", err)
		return nil, err
	}
	var actionResult remoteexecution.ActionResult
	if err := proto.Unmarshal(actionResultData, &actionResult); err != nil {
		log.Print("actionCacheServer.GetActionResult: ", err)
		return nil, err
	}
	return &actionResult, nil
}

func (s *actionCacheServer) UpdateActionResult(ctx context.Context, in *remoteexecution.UpdateActionResultRequest) (*remoteexecution.ActionResult, error) {
	log.Print("Attempted to call ActionCache.UpdateActionResult")
	return nil, errors.New("Fail!")
}

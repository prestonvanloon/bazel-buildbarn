package main

import (
	"errors"
	"io/ioutil"
	"log"

	"github.com/EdSchouten/bazel-buildbarn/pkg/cas"
	"github.com/golang/protobuf/proto"

	"golang.org/x/net/context"
	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	"google.golang.org/genproto/googleapis/longrunning"
)

type executionServer struct {
	blobAccess cas.BlobAccess
}

func NewExecutionServer(blobAccess cas.BlobAccess) remoteexecution.ExecutionServer {
	return &executionServer{
		blobAccess: blobAccess,
	}
}

func (s *executionServer) Execute(ctx context.Context, in *remoteexecution.ExecuteRequest) (*longrunning.Operation, error) {
	r, err := s.blobAccess.Get(in.InstanceName, in.Action.CommandDigest)
	if err != nil {
		log.Print("Execution.Execute: ", err)
		return nil, err
	}
	commandData, err := ioutil.ReadAll(r)
	if err != nil {
		log.Print("Execution.Execute: ", err)
		return nil, err
	}
	var command remoteexecution.Command
	if err := proto.Unmarshal(commandData, &command); err != nil {
		log.Print("Execution.Execute: ", err)
		return nil, err
	}
	log.Print("Got command: ", command)
	return nil, errors.New("Fail!")
}

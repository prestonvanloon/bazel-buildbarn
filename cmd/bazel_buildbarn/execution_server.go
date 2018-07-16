package main

import (
	"errors"
	"io/ioutil"
	"log"

	"github.com/EdSchouten/bazel-buildbarn/pkg/blobstore"
	"github.com/golang/protobuf/proto"

	"golang.org/x/net/context"
	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	"google.golang.org/genproto/googleapis/longrunning"
)

type executionServer struct {
	blobAccess blobstore.BlobAccess
}

func NewExecutionServer(blobAccess blobstore.BlobAccess) remoteexecution.ExecutionServer {
	return &executionServer{
		blobAccess: blobAccess,
	}
}

func (s *executionServer) Execute(ctx context.Context, in *remoteexecution.ExecuteRequest) (*longrunning.Operation, error) {
	log.Print("Got ExecuteRequest:" , in)

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

	r, err = s.blobAccess.Get(in.InstanceName, in.Action.InputRootDigest)
	if err != nil {
		log.Print("Execution.Execute: ", err)
		return nil, err
	}
	inputRootData, err := ioutil.ReadAll(r)
	if err != nil {
		log.Print("Execution.Execute: ", err)
		return nil, err
	}
	var inputRoot remoteexecution.Directory
	if err := proto.Unmarshal(inputRootData, &inputRoot); err != nil {
		log.Print("Execution.Execute: ", err)
		return nil, err
	}
	log.Print("Got input root: ", inputRoot)

	return nil, errors.New("Fail!")
}

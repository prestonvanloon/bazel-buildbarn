package main

import (
	"errors"
	"log"

	"golang.org/x/net/context"
	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	"google.golang.org/genproto/googleapis/longrunning"
)

type ExecutionServer struct {
}

func (s *ExecutionServer) Execute(ctx context.Context, in *remoteexecution.ExecuteRequest) (*longrunning.Operation, error) {
	log.Print("Attempted to call Execution.Execute")
	return nil, errors.New("Fail!")
}

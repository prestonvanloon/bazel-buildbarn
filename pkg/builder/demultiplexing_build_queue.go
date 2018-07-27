package builder

import (
	"fmt"
	"strings"

	"golang.org/x/net/context"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	"google.golang.org/genproto/googleapis/longrunning"
	watcher "google.golang.org/genproto/googleapis/watcher/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type demultiplexingBuildQueue struct {
	backends map[string]BuildQueue
}

func NewDemultiplexingBuildQueue(backends map[string]BuildQueue) BuildQueue {
	return &demultiplexingBuildQueue{
		backends: backends,
	}
}

func (bq *demultiplexingBuildQueue) Execute(ctx context.Context, request *remoteexecution.ExecuteRequest) (*longrunning.Operation, error) {
	if strings.ContainsRune(request.InstanceName, '|') {
		return nil, status.Errorf(codes.InvalidArgument, "Instance name cannot contain pipe character")
	}
	backend, ok := bq.backends[request.InstanceName]
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "Unknown instance name")
	}
	operation, err := backend.Execute(ctx, request)
	if err != nil {
		return nil, err
	}
	operationCopy := *operation
	operationCopy.Name = fmt.Sprintf("%s|%s", request.InstanceName, operation.Name)
	return &operationCopy, nil
}

func (bq *demultiplexingBuildQueue) Watch(in *watcher.Request, out watcher.Watcher_WatchServer) error {
	target := strings.SplitN(in.Target, "|", 2)
	if len(target) != 2 {
		return status.Errorf(codes.InvalidArgument, "Unable to extract instance name from watch request")
	}
	backend, ok := bq.backends[target[0]]
	if !ok {
		return status.Errorf(codes.InvalidArgument, "Unknown instance name")
	}
	requestCopy := *in
	requestCopy.Target = target[1]
	return backend.Watch(&requestCopy, out)
}

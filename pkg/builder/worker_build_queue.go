package builder

import (
	"github.com/EdSchouten/bazel-buildbarn/pkg/proto/scheduler"

	"golang.org/x/net/context"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	"google.golang.org/genproto/googleapis/longrunning"
	watcher "google.golang.org/genproto/googleapis/watcher/v1"
)

type WorkerBuildQueue struct {
}

func NewWorkerBuildQueue() *WorkerBuildQueue {
	return &WorkerBuildQueue{}
}

func (bq *WorkerBuildQueue) Execute(ctx context.Context, request *remoteexecution.ExecuteRequest) (*longrunning.Operation, error) {
	// TODO(edsch): Implement.
	return nil, nil
}

func (bq *WorkerBuildQueue) Watch(in *watcher.Request, out watcher.Watcher_WatchServer) error {
	// TODO(edsch): Implement.
	return nil
}

func (bq *WorkerBuildQueue) GetWork(stream scheduler.Scheduler_GetWorkServer) error {
	// TODO(edsch): Implement.
	return nil
}

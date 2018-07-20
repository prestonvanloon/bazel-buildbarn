package builder

import (
	"errors"

	"github.com/EdSchouten/bazel-buildbarn/pkg/blobstore"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	"google.golang.org/genproto/googleapis/longrunning"
	watcher "google.golang.org/genproto/googleapis/watcher/v1"

	"golang.org/x/net/context"
)

type synchronousBuildQueue struct {
	buildExecutor BuildExecutor
	actionCache   blobstore.BlobAccess
}

func NewSynchronousBuildQueue(buildExecutor BuildExecutor, actionCache blobstore.BlobAccess) BuildQueue {
	return &synchronousBuildQueue{
		buildExecutor: buildExecutor,
	}
}

func (bq *synchronousBuildQueue) Execute(ctx context.Context, request *remoteexecution.ExecuteRequest) (*longrunning.Operation, error) {
	return nil, errors.New("Not implemented")
}

func (bq *synchronousBuildQueue) Watch(in *watcher.Request, out watcher.Watcher_WatchServer) error {
	return errors.New("Not implemented")
}

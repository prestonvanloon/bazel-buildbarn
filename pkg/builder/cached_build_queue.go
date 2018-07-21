package builder

import (
	"github.com/EdSchouten/bazel-buildbarn/pkg/blobstore"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	"google.golang.org/genproto/googleapis/longrunning"
	watcher "google.golang.org/genproto/googleapis/watcher/v1"

	"golang.org/x/net/context"
)

type cachedBuildQueue struct {
	actionCache blobstore.BlobAccess
	fallback    BuildQueue
}

func NewCachedBuildQueue(actionCache blobstore.BlobAccess, fallback BuildQueue) BuildQueue {
	return &cachedBuildQueue{
		actionCache: actionCache,
		fallback:    fallback,
	}
}

func (bq *cachedBuildQueue) Execute(ctx context.Context, request *remoteexecution.ExecuteRequest) (*longrunning.Operation, error) {
	if !request.SkipCacheLookup {
		// TODO(edsch): Inspect the action cache!
	}
	return bq.fallback.Execute(ctx, request)
}

func (bq *cachedBuildQueue) Watch(in *watcher.Request, out watcher.Watcher_WatchServer) error {
	return bq.fallback.Watch(in, out)
}

package builder

import (
	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	"google.golang.org/genproto/googleapis/longrunning"
	watcher "google.golang.org/genproto/googleapis/watcher/v1"

	"golang.org/x/net/context"
)

type BuildQueue interface {
	Execute(ctx context.Context, request *remoteexecution.ExecuteRequest) (*longrunning.Operation, error)
	Watch(in *watcher.Request, out watcher.Watcher_WatchServer) error
}

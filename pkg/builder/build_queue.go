package builder

import (
	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	watcher "google.golang.org/genproto/googleapis/watcher/v1"
)

type BuildQueue interface {
	remoteexecution.ExecutionServer
	watcher.WatcherServer
}

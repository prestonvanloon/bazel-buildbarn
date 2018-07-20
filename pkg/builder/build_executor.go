package builder

import (
	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
)

type BuildExecutor interface {
	Execute(request *remoteexecution.ExecuteRequest) (*remoteexecution.ExecuteResponse, error)
}

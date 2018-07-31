package builder

import (
	"context"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	"google.golang.org/grpc/status"
)

func convertErrorToExecuteResponse(err error) *remoteexecution.ExecuteResponse {
	return &remoteexecution.ExecuteResponse{Status: status.Convert(err).Proto()}
}

type BuildExecutor interface {
	Execute(ctx context.Context, request *remoteexecution.ExecuteRequest) *remoteexecution.ExecuteResponse
}

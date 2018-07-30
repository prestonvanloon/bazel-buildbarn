package builder

import (
	"golang.org/x/net/context"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	"google.golang.org/grpc/status"
)

func ConvertErrorToExecuteResponse(err error) *remoteexecution.ExecuteResponse {
	return &remoteexecution.ExecuteResponse{Status: status.Convert(err).Proto()}
}

type BuildExecutor interface {
	Execute(ctx context.Context, request *remoteexecution.ExecuteRequest) *remoteexecution.ExecuteResponse
}

package builder

import (
	"io"

	"golang.org/x/net/context"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	"google.golang.org/genproto/googleapis/longrunning"
	watcher "google.golang.org/genproto/googleapis/watcher/v1"
	"google.golang.org/grpc"
)

type grpcClientConnBuildQueue struct {
	executionClient remoteexecution.ExecutionClient
	watcherClient   watcher.WatcherClient
}

func NewGrpcClientConnBuildQueue(client *grpc.ClientConn) BuildQueue {
	return &grpcClientConnBuildQueue{
		executionClient: remoteexecution.NewExecutionClient(client),
		watcherClient:   watcher.NewWatcherClient(client),
	}
}

func (bq *grpcClientConnBuildQueue) Execute(ctx context.Context, request *remoteexecution.ExecuteRequest) (*longrunning.Operation, error) {
	return bq.executionClient.Execute(ctx, request)
}

func (bq *grpcClientConnBuildQueue) Watch(in *watcher.Request, out watcher.Watcher_WatchServer) error {
	client, err := bq.watcherClient.Watch(out.Context(), in)
	if err != nil {
		return err
	}
	for {
		changeBatch, err := client.Recv()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if err := out.Send(changeBatch); err != nil {
			return err
		}
	}
}

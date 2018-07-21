package builder

import (
	"errors"
	"log"

	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	"github.com/golang/protobuf/ptypes"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	"google.golang.org/genproto/googleapis/longrunning"
	watcher "google.golang.org/genproto/googleapis/watcher/v1"

	"golang.org/x/net/context"
)

type synchronousBuildQueue struct {
	buildExecutor      BuildExecutor
	deduplicationKeyer util.DigestKeyer
}

func NewSynchronousBuildQueue(buildExecutor BuildExecutor, deduplicationKeyer util.DigestKeyer) BuildQueue {
	return &synchronousBuildQueue{
		buildExecutor:      buildExecutor,
		deduplicationKeyer: deduplicationKeyer,
	}
}

func (bq *synchronousBuildQueue) Execute(ctx context.Context, request *remoteexecution.ExecuteRequest) (*longrunning.Operation, error) {
	digest, err := util.DigestFromMessage(request.Action)
	if err != nil {
		return nil, err
	}
	// Use the action digest to deduplicate identical execution requests.
	name, err := bq.deduplicationKeyer(request.InstanceName, digest)
	if err != nil {
		return nil, err
	}
	metadata, err := ptypes.MarshalAny(&remoteexecution.ExecuteOperationMetadata{
		Stage:        remoteexecution.ExecuteOperationMetadata_QUEUED,
		ActionDigest: digest,
		// TODO(edsch): Do we need StdoutStreamName and StderrStreamName? Bazel doesn't seem to use them.
	})
	if err != nil {
		return nil, err
	}

	// TODO(edsch): Actually enqueue the execution request!

	return &longrunning.Operation{
		Name:     name,
		Metadata: metadata,
	}, nil
}

func (bq *synchronousBuildQueue) Watch(in *watcher.Request, out watcher.Watcher_WatchServer) error {
	log.Print("Hi: ", in)
	return errors.New("Not implemented")
}

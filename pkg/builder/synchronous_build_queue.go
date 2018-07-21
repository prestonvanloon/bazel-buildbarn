package builder

import (
	"log"
	"sync"
	"time"

	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	"github.com/golang/protobuf/ptypes"
	"github.com/satori/go.uuid"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	"google.golang.org/genproto/googleapis/longrunning"
	watcher "google.golang.org/genproto/googleapis/watcher/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"golang.org/x/net/context"
)

type synchronousBuildJob struct {
	name             string
	actionDigest     *remoteexecution.Digest
	deduplicationKey string
	executeRequest   remoteexecution.ExecuteRequest

	executeResponse *remoteexecution.ExecuteResponse
	executeError    error
}

func (bj *synchronousBuildJob) getCurrentState() *longrunning.Operation {
	metadata, err := ptypes.MarshalAny(&remoteexecution.ExecuteOperationMetadata{
		Stage:        remoteexecution.ExecuteOperationMetadata_QUEUED,
		ActionDigest: bj.actionDigest,
		// TODO(edsch): Do we need StdoutStreamName and StderrStreamName? Bazel doesn't seem to use them.
	})
	if err != nil {
		log.Fatal("Failed to marshal execute operation metadata: ", err)
	}
	operation := &longrunning.Operation{
		Name:     bj.name,
		Metadata: metadata,
	}
	if bj.executeResponse != nil {
		operation.Done = true
		response, err := ptypes.MarshalAny(bj.executeResponse)
		if err != nil {
			log.Fatal("Failed to marshal execute response: ", err)
		}
		operation.Result = &longrunning.Operation_Response{Response: response}
	} else if bj.executeError != nil {
		s, _ := status.FromError(bj.executeError)
		operation.Done = true
		operation.Result = &longrunning.Operation_Error{Error: s.Proto()}
	}
	return operation
}

func (bj *synchronousBuildJob) waitForTransition() {
	if bj.executeResponse != nil || bj.executeError != nil {
		return
	}

	// TODO(edsch): Implement this properly.
	time.Sleep(time.Second * 5)
}

type synchronousBuildQueue struct {
	buildExecutor      BuildExecutor
	deduplicationKeyer util.DigestKeyer

	jobsLock                    sync.Mutex
	jobsNameMap                 map[string]*synchronousBuildJob
	jobsPending                 []*synchronousBuildJob
	jobsPendingInsertionWakeup  *sync.Cond
	jobsPendingDeduplicationMap map[string]*synchronousBuildJob
}

func NewSynchronousBuildQueue(buildExecutor BuildExecutor, deduplicationKeyer util.DigestKeyer) BuildQueue {
	bq := &synchronousBuildQueue{
		buildExecutor:      buildExecutor,
		deduplicationKeyer: deduplicationKeyer,

		jobsNameMap:                 map[string]*synchronousBuildJob{},
		jobsPendingDeduplicationMap: map[string]*synchronousBuildJob{},
	}
	bq.jobsPendingInsertionWakeup = sync.NewCond(&bq.jobsLock)
	return bq
}

func (bq *synchronousBuildQueue) Execute(ctx context.Context, request *remoteexecution.ExecuteRequest) (*longrunning.Operation, error) {
	actionDigest, err := util.DigestFromMessage(request.Action)
	if err != nil {
		return nil, err
	}
	deduplicationKey, err := bq.deduplicationKeyer(request.InstanceName, actionDigest)
	if err != nil {
		return nil, err
	}

	bq.jobsLock.Lock()
	defer bq.jobsLock.Unlock()

	job, ok := bq.jobsPendingDeduplicationMap[deduplicationKey]
	if !ok {
		job = &synchronousBuildJob{
			name:             uuid.Must(uuid.NewV4()).String(),
			actionDigest:     actionDigest,
			deduplicationKey: deduplicationKey,
			executeRequest:   *request,
		}
		bq.jobsNameMap[job.name] = job
		bq.jobsPending = append(bq.jobsPending, job)
		bq.jobsPendingInsertionWakeup.Signal()
		bq.jobsPendingDeduplicationMap[deduplicationKey] = job
	}
	return job.getCurrentState(), nil
}

func (bq *synchronousBuildQueue) Watch(in *watcher.Request, out watcher.Watcher_WatchServer) error {
	bq.jobsLock.Lock()
	defer bq.jobsLock.Unlock()

	job, ok := bq.jobsNameMap[in.Target]
	if !ok {
		return status.Errorf(codes.NotFound, "Build job with name %s not found", in.Target)
	}

	for {
		state := job.getCurrentState()
		log.Print("Returning state: ", state)
		stateAny, err := ptypes.MarshalAny(state)
		if err != nil {
			return err
		}
		if err := out.Send(&watcher.ChangeBatch{
			Changes: []*watcher.Change{
				{
					State: watcher.Change_EXISTS,
					Data:  stateAny,
				},
			},
		}); err != nil {
			return err
		}
		if state.Done {
			return nil
		}
		job.waitForTransition()
	}
}

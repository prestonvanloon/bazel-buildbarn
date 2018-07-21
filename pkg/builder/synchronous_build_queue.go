package builder

import (
	"sync"

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
	deduplicationKey string
	executeRequest   remoteexecution.ExecuteRequest
}

func (bj *synchronousBuildJob) getCurrentState() *longrunning.Operation {
	// TODO(edsch): Implement!
	return nil
}

func (bj *synchronousBuildJob) waitForTransition() {
	// TODO(edsch): Implement!
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
			deduplicationKey: deduplicationKey,
			executeRequest:   *request,
		}
		bq.jobsNameMap[job.name] = job
		bq.jobsPending = append(bq.jobsPending, job)
		bq.jobsPendingInsertionWakeup.Signal()
		bq.jobsPendingDeduplicationMap[job.deduplicationKey] = job
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
		stateAny, err := ptypes.MarshalAny(state)
		if err != nil {
			return err
		}
		out.Send(&watcher.ChangeBatch{
			Changes: []*watcher.Change{
				{
					State: watcher.Change_EXISTS,
					Data:  stateAny,
				},
			},
		})
		if state.Done {
			return nil
		}
		job.waitForTransition()
	}
}

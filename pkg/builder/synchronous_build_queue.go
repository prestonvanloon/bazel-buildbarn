package builder

import (
	"log"
	"sync"

	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	"github.com/golang/protobuf/ptypes"
	"github.com/satori/go.uuid"

	"golang.org/x/net/context"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	"google.golang.org/genproto/googleapis/longrunning"
	watcher "google.golang.org/genproto/googleapis/watcher/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type synchronousBuildJob struct {
	name             string
	actionDigest     *remoteexecution.Digest
	deduplicationKey string
	executeRequest   remoteexecution.ExecuteRequest

	stage                   remoteexecution.ExecuteOperationMetadata_Stage
	executeResponse         *remoteexecution.ExecuteResponse
	executeTransitionWakeup *sync.Cond
}

func (job *synchronousBuildJob) getCurrentState() *longrunning.Operation {
	metadata, err := ptypes.MarshalAny(&remoteexecution.ExecuteOperationMetadata{
		Stage:        job.stage,
		ActionDigest: job.actionDigest,
	})
	if err != nil {
		log.Fatal("Failed to marshal execute operation metadata: ", err)
	}
	operation := &longrunning.Operation{
		Name:     job.name,
		Metadata: metadata,
	}
	if job.executeResponse != nil {
		operation.Done = true
		response, err := ptypes.MarshalAny(job.executeResponse)
		if err != nil {
			log.Fatal("Failed to marshal execute response: ", err)
		}
		operation.Result = &longrunning.Operation_Response{Response: response}
	}
	return operation
}

// TODO(edsch): Should take a context.
// TODO(edsch): Should wake up periodically.
func (job *synchronousBuildJob) waitForTransition() {
	if job.executeResponse == nil {
		job.executeTransitionWakeup.Wait()
	}
}

type SynchronousBuildQueue struct {
	buildExecutor      BuildExecutor
	deduplicationKeyer util.DigestKeyer
	jobsPendingMax     uint

	jobsLock                   sync.Mutex
	jobsNameMap                map[string]*synchronousBuildJob
	jobsDeduplicationMap       map[string]*synchronousBuildJob
	jobsPending                []*synchronousBuildJob
	jobsPendingInsertionWakeup *sync.Cond
}

func NewSynchronousBuildQueue(buildExecutor BuildExecutor, deduplicationKeyer util.DigestKeyer, jobsPendingMax uint) *SynchronousBuildQueue {
	bq := &SynchronousBuildQueue{
		buildExecutor:      buildExecutor,
		deduplicationKeyer: deduplicationKeyer,
		jobsPendingMax:     jobsPendingMax,

		jobsNameMap:          map[string]*synchronousBuildJob{},
		jobsDeduplicationMap: map[string]*synchronousBuildJob{},
	}
	bq.jobsPendingInsertionWakeup = sync.NewCond(&bq.jobsLock)
	return bq
}

func (bq *SynchronousBuildQueue) Execute(ctx context.Context, request *remoteexecution.ExecuteRequest) (*longrunning.Operation, error) {
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

	job, ok := bq.jobsDeduplicationMap[deduplicationKey]
	if !ok {
		if uint(len(bq.jobsPending)) >= bq.jobsPendingMax {
			return nil, status.Errorf(codes.Unavailable, "Too many jobs pending")
		}

		job = &synchronousBuildJob{
			name:             uuid.NewV4().String(),
			actionDigest:     actionDigest,
			deduplicationKey: deduplicationKey,
			executeRequest:   *request,
			stage:            remoteexecution.ExecuteOperationMetadata_QUEUED,
			executeTransitionWakeup: sync.NewCond(&bq.jobsLock),
		}
		bq.jobsNameMap[job.name] = job
		bq.jobsDeduplicationMap[deduplicationKey] = job
		bq.jobsPending = append(bq.jobsPending, job)
		bq.jobsPendingInsertionWakeup.Signal()
	}
	return job.getCurrentState(), nil
}

func (bq *SynchronousBuildQueue) Watch(in *watcher.Request, out watcher.Watcher_WatchServer) error {
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

func (bq *SynchronousBuildQueue) Run() {
	bq.jobsLock.Lock()
	defer bq.jobsLock.Unlock()

	// TODO(edsch): Purge jobs from the jobsNameMap after some amount of time.
	for {
		// Extract job from queue.
		for len(bq.jobsPending) == 0 {
			bq.jobsPendingInsertionWakeup.Wait()
		}
		job := bq.jobsPending[0]
		bq.jobsPending = bq.jobsPending[1:]

		// Perform execution of the job.
		// TODO(edsch): Set up a proper context with a timeout.
		job.stage = remoteexecution.ExecuteOperationMetadata_EXECUTING
		bq.jobsLock.Unlock()
		executeResponse := bq.buildExecutor.Execute(context.Background(), &job.executeRequest)
		bq.jobsLock.Lock()

		// Mark completion.
		delete(bq.jobsDeduplicationMap, job.deduplicationKey)
		job.stage = remoteexecution.ExecuteOperationMetadata_COMPLETED
		job.executeResponse = executeResponse
		job.executeTransitionWakeup.Broadcast()
	}
}

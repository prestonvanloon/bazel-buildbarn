package builder

import (
	"log"
	"sync"

	"github.com/EdSchouten/bazel-buildbarn/pkg/proto/scheduler"
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

type workerBuildJob struct {
	name             string
	actionDigest     *remoteexecution.Digest
	deduplicationKey string
	executeRequest   remoteexecution.ExecuteRequest

	stage                   remoteexecution.ExecuteOperationMetadata_Stage
	executeResponse         *remoteexecution.ExecuteResponse
	executeTransitionWakeup *sync.Cond
}

func (job *workerBuildJob) getCurrentState() *longrunning.Operation {
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
func (job *workerBuildJob) waitForTransition() {
	if job.executeResponse == nil {
		job.executeTransitionWakeup.Wait()
	}
}

type WorkerBuildQueue struct {
	deduplicationKeyer util.DigestKeyer
	jobsPendingMax     uint

	jobsLock                   sync.Mutex
	jobsNameMap                map[string]*workerBuildJob
	jobsDeduplicationMap       map[string]*workerBuildJob
	jobsPending                []*workerBuildJob
	jobsPendingInsertionWakeup *sync.Cond
}

func NewWorkerBuildQueue(deduplicationKeyer util.DigestKeyer, jobsPendingMax uint) *WorkerBuildQueue {
	bq := &WorkerBuildQueue{
		deduplicationKeyer: deduplicationKeyer,
		jobsPendingMax:     jobsPendingMax,

		jobsNameMap:          map[string]*workerBuildJob{},
		jobsDeduplicationMap: map[string]*workerBuildJob{},
	}
	bq.jobsPendingInsertionWakeup = sync.NewCond(&bq.jobsLock)
	return bq
}

func (bq *WorkerBuildQueue) Execute(ctx context.Context, request *remoteexecution.ExecuteRequest) (*longrunning.Operation, error) {
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
		// TODO(edsch): Maybe let the number of workers influence this?
		if uint(len(bq.jobsPending)) >= bq.jobsPendingMax {
			return nil, status.Errorf(codes.Unavailable, "Too many jobs pending")
		}

		job = &workerBuildJob{
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

func (bq *WorkerBuildQueue) Watch(in *watcher.Request, out watcher.Watcher_WatchServer) error {
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

func executeOnWorker(stream scheduler.Scheduler_GetWorkServer, request *remoteexecution.ExecuteRequest) *remoteexecution.ExecuteResponse {
	// TODO(edsch): Any way we can set a timeout here?
	if err := stream.Send(request); err != nil {
		return ConvertErrorToExecuteResponse(err)
	}
	response, err := stream.Recv()
	if err != nil {
		return ConvertErrorToExecuteResponse(err)
	}
	return response
}

func (bq *WorkerBuildQueue) GetWork(stream scheduler.Scheduler_GetWorkServer) error {
	bq.jobsLock.Lock()
	defer bq.jobsLock.Unlock()

	// TODO(edsch): Purge jobs from the jobsNameMap after some amount of time.
	for {
		// Wait for jobs to appear.
		// TODO(edsch): sync.Cond.WaitWithContext() would be helpful here.
		for len(bq.jobsPending) == 0 {
			bq.jobsPendingInsertionWakeup.Wait()
		}
		if err := stream.Context().Err(); err != nil {
			bq.jobsPendingInsertionWakeup.Signal()
			return err
		}

		// Extract job from queue.
		job := bq.jobsPending[0]
		bq.jobsPending = bq.jobsPending[1:]
		job.stage = remoteexecution.ExecuteOperationMetadata_EXECUTING

		// Perform execution of the job.
		bq.jobsLock.Unlock()
		executeResponse := executeOnWorker(stream, &job.executeRequest)
		bq.jobsLock.Lock()

		// Mark completion.
		delete(bq.jobsDeduplicationMap, job.deduplicationKey)
		job.stage = remoteexecution.ExecuteOperationMetadata_COMPLETED
		job.executeResponse = executeResponse
		job.executeTransitionWakeup.Broadcast()
	}
}

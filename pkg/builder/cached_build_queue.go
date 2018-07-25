package builder

import (
	"log"

	"github.com/EdSchouten/bazel-buildbarn/pkg/ac"
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	"github.com/golang/protobuf/ptypes"
	"github.com/satori/go.uuid"

	"golang.org/x/net/context"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	"google.golang.org/genproto/googleapis/longrunning"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type cachedBuildQueue struct {
	BuildQueue

	actionCache ac.ActionCache
}

func NewCachedBuildQueue(fallback BuildQueue, actionCache ac.ActionCache) BuildQueue {
	return &cachedBuildQueue{
		BuildQueue: fallback,

		actionCache: actionCache,
	}
}

func (bq *cachedBuildQueue) Execute(ctx context.Context, request *remoteexecution.ExecuteRequest) (*longrunning.Operation, error) {
	if !request.SkipCacheLookup {
		digest, err := util.DigestFromMessage(request.Action)
		if err != nil {
			return nil, err
		}
		result, err := bq.actionCache.GetActionResult(ctx, request.InstanceName, digest)
		if err == nil {
			// Found action in action cache. Return it immediately.
			metadata, err := ptypes.MarshalAny(&remoteexecution.ExecuteOperationMetadata{
				Stage:        remoteexecution.ExecuteOperationMetadata_COMPLETED,
				ActionDigest: digest,
			})
			response, err := ptypes.MarshalAny(&remoteexecution.ExecuteResponse{
				Result:       result,
				CachedResult: true,
			})
			if err != nil {
				log.Fatal("Failed to marshal execute response: ", err)
			}
			return &longrunning.Operation{
				Name:     uuid.NewV4().String(),
				Metadata: metadata,
				Done:     true,
				Result:   &longrunning.Operation_Response{Response: response},
			}, nil
		} else if status.Code(err) != codes.NotFound {
			return nil, err
		}
	}
	return bq.BuildQueue.Execute(ctx, request)
}

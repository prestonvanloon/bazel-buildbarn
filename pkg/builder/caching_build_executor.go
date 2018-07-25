package builder

import (
	"github.com/EdSchouten/bazel-buildbarn/pkg/ac"
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"

	"golang.org/x/net/context"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
)

type cachingBuildExecutor struct {
	base        BuildExecutor
	actionCache ac.ActionCache
}

func NewCachingBuildExecutor(base BuildExecutor, actionCache ac.ActionCache) BuildExecutor {
	return &cachingBuildExecutor{
		base:        base,
		actionCache: actionCache,
	}
}

func (be *cachingBuildExecutor) Execute(ctx context.Context, request *remoteexecution.ExecuteRequest) (*remoteexecution.ExecuteResponse, error) {
	response, err := be.base.Execute(ctx, request)
	if err != nil {
		return nil, err
	}
	if !request.Action.DoNotCache && response.Result.ExitCode == 0 {
		digest, err := util.DigestFromMessage(request.Action)
		if err != nil {
			return nil, err
		}
		if err := be.actionCache.PutActionResult(ctx, request.InstanceName, digest, response.Result); err != nil {
			return nil, err
		}
	}
	return response, nil
}

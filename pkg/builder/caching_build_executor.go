package builder

import (
	"github.com/EdSchouten/bazel-buildbarn/pkg/blobstore"
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
)

type cachingBuildExecutor struct {
	base       BuildExecutor
	blobAccess blobstore.BlobAccess
}

func NewCachingBuildExecutor(base BuildExecutor, blobAccess blobstore.BlobAccess) BuildExecutor {
	return &cachingBuildExecutor{
		base:       base,
		blobAccess: blobAccess,
	}
}

func (be *cachingBuildExecutor) Execute(request *remoteexecution.ExecuteRequest) (*remoteexecution.ExecuteResponse, error) {
	response, err := be.base.Execute(request)
	if err != nil {
		return nil, err
	}
	if !request.Action.DoNotCache {
		digest, err := util.DigestFromMessage(request.Action)
		if err != nil {
			return nil, err
		}
		if err := blobstore.PutMessageToBlobAccess(be.blobAccess, request.InstanceName, digest, response.Result); err != nil {
			return nil, err
		}
	}
	return response, nil
}

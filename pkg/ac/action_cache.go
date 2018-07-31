package ac

import (
	"context"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
)

type ActionCache interface {
	GetActionResult(ctx context.Context, instance string, digest *remoteexecution.Digest) (*remoteexecution.ActionResult, error)
	PutActionResult(ctx context.Context, instance string, digest *remoteexecution.Digest, result *remoteexecution.ActionResult) error
}

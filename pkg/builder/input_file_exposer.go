package builder

import (
	"golang.org/x/net/context"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
)

type InputFileExposer interface {
	Expose(ctx context.Context, instance string, digest *remoteexecution.Digest, outputPath string, isExecutable bool) error
}

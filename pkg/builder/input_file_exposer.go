package builder

import (
	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
)

type InputFileExposer interface {
	Expose(instance string, digest *remoteexecution.Digest, base string, isExecutable bool) error
}

package cas

import (
	"context"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
)

type ContentAddressableStorage interface {
	GetCommand(ctx context.Context, instance string, digest *remoteexecution.Digest) (*remoteexecution.Command, error)
	GetDirectory(ctx context.Context, instance string, digest *remoteexecution.Digest) (*remoteexecution.Directory, error)
	GetFile(ctx context.Context, instance string, digest *remoteexecution.Digest, outputPath string, isExecutable bool) error
	PutFile(ctx context.Context, instance string, path string) (*remoteexecution.Digest, bool, error)
	PutTree(ctx context.Context, instance string, tree *remoteexecution.Tree) (*remoteexecution.Digest, error)
}

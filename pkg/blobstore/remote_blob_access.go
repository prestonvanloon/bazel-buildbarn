package blobstore

import (
	"context"
	"fmt"
	"golang.org/x/net/context/ctxhttp"
	"io"
	"net/http"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
)

type remoteBlobAccess struct {
	address string
	prefix  string
}

func NewRemoteBlobAccess(address, prefix string) BlobAccess {
	return &remoteBlobAccess{
		address: address,
		prefix:  prefix,
	}
}

func (ba *remoteBlobAccess) Get(ctx context.Context, instance string, digest *remoteexecution.Digest) io.ReadCloser {
	resp, err := ctxhttp.Get(ctx, http.DefaultClient, fmt.Sprintf("%s/%s/%s", ba.address, ba.prefix, digest.GetHash()))
	if err != nil {
		// todo
	}

	return resp.Body
}

func (ba *remoteBlobAccess) Put(ctx context.Context, instance string, digest *remoteexecution.Digest, r io.ReadCloser) error {
	_, err := ctxhttp.Post(ctx, http.DefaultClient, fmt.Sprintf("%s/%s/%s", ba.address, ba.prefix, digest.GetHash()), "todo-body-type", r)

	return err
}

func (ba *remoteBlobAccess) FindMissing(ctx context.Context, instance string, digests []*remoteexecution.Digest) ([]*remoteexecution.Digest, error) {
	var missing []*remoteexecution.Digest
	for _, digest := range digests {
		url := fmt.Sprintf("%s/%s/%s", ba.address, ba.prefix, digest.GetHash())
		resp, err := ctxhttp.Head(ctx, http.DefaultClient, url)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != 200 {
			missing = append(missing, digest)
		}
	}
	return missing, nil
}

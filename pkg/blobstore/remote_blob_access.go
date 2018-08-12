package blobstore

import (
	"context"
	"fmt"
	"golang.org/x/net/context/ctxhttp"
	"io"
	"net/http"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	url := fmt.Sprintf("%s/%s/%s", ba.address, ba.prefix, digest.GetHash())
	resp, err := ctxhttp.Get(ctx, http.DefaultClient, url)
	if err != nil {
		fmt.Printf("Error getting digest. %s\n", err)
		return &errorReader{err: err}
	}

	if resp.StatusCode == http.StatusNotFound {
		return &errorReader{err: status.Errorf(codes.NotFound, url)}
	}

	if resp.StatusCode != http.StatusOK {
		return &errorReader{err: status.Errorf(codes.NotFound, "Unexpected status code from remote cache: %d - %s", resp.StatusCode, http.StatusText(resp.StatusCode))}
	}

	return resp.Body
}

func (ba *remoteBlobAccess) Put(ctx context.Context, instance string, digest *remoteexecution.Digest, r io.ReadCloser) error {
	url := fmt.Sprintf("%s/%s/%s", ba.address, ba.prefix, digest.GetHash())
	req, err := http.NewRequest(http.MethodPut, url, r)
	if err != nil {
		return err
	}
	// req.ContentLength = digest.GetSizeBytes()

	_, err = ctxhttp.Do(ctx, http.DefaultClient, req)
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

		if resp.StatusCode != http.StatusOK {
			missing = append(missing, digest)
		}
	}

	return missing, nil
}

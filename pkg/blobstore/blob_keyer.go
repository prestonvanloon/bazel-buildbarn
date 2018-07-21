package blobstore

import (
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
)

type BlobKeyer func(instance string, digest *remoteexecution.Digest) (string, error)

func KeyBlobWithInstance(instance string, digest *remoteexecution.Digest) (string, error) {
	digestString, err := util.DigestToString(digest)
	if err != nil {
		return "", err
	}
	return digestString + "|" + instance, nil
}

func KeyBlobWithoutInstance(_ string, digest *remoteexecution.Digest) (string, error) {
	return util.DigestToString(digest)
}

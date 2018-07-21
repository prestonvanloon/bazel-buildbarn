package util

import (
	"errors"
	"fmt"
	"strings"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
)

type DigestKeyer func(instance string, digest *remoteexecution.Digest) (string, error)

func KeyDigestWithInstance(instance string, digest *remoteexecution.Digest) (string, error) {
	if strings.ContainsRune(digest.Hash, '|') {
		return "", errors.New("Blob hash cannot contain pipe character")
	}
	if strings.ContainsRune(instance, '|') {
		return "", errors.New("Instance name cannot contain pipe character")
	}
	return fmt.Sprintf("%s|%d|%s", digest.Hash, digest.SizeBytes, instance), nil
}

func KeyDigestWithoutInstance(_ string, digest *remoteexecution.Digest) (string, error) {
	if strings.ContainsRune(digest.Hash, '|') {
		return "", errors.New("Blob hash cannot contain pipe character")
	}
	return fmt.Sprintf("%s|%d", digest.Hash, digest.SizeBytes), nil
}

package util

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/golang/protobuf/proto"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
)

func DigestFromMessage(pb proto.Message) (*remoteexecution.Digest, error) {
	data, err := proto.Marshal(pb)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(data)
	return &remoteexecution.Digest{
		Hash:      hex.EncodeToString(hash[:]),
		SizeBytes: int64(len(data)),
	}, nil
}

func DigestToString(digest *remoteexecution.Digest) (string, error) {
	for _, c := range digest.Hash {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return "", errors.New("Blob hash can only contain hexadecimal characters")
		}
	}
	return fmt.Sprintf("%s|%d", digest.Hash, digest.SizeBytes), nil
}

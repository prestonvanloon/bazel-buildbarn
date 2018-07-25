package util

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/golang/protobuf/proto"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
)

func DigestFromData(data []byte) *remoteexecution.Digest {
	hash := sha256.Sum256(data)
	return &remoteexecution.Digest{
		Hash:      hex.EncodeToString(hash[:]),
		SizeBytes: int64(len(data)),
	}
}

func DigestFromMessage(pb proto.Message) (*remoteexecution.Digest, error) {
	data, err := proto.Marshal(pb)
	if err != nil {
		return nil, err
	}
	return DigestFromData(data), nil
}

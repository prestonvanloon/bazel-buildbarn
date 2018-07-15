package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"

	"github.com/EdSchouten/bazel-buildbarn/pkg/cas"

	"golang.org/x/net/context"
	"google.golang.org/genproto/googleapis/bytestream"
	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
)

// parseResourceName parses resource name strings in one of the following two forms:
//
// - uploads/${uuid}/blobs/${hash}/${size}
// - ${instance}/uploads/${uuid}/blobs/${hash}/${size}
//
// In the process, the hash, size and instance are extracted.
func parseResourceName(resourceName string) (string, *remoteexecution.Digest) {
	fields := strings.FieldsFunc(resourceName, func(r rune) bool { return r == '/' })
	l := len(fields)
	if (l != 5 && l != 6) || fields[l-5] != "uploads" || fields[l-3] != "blobs" {
		return "", nil
	}
	size, err := strconv.ParseInt(fields[l-1], 10, 64)
	if err != nil {
		return "", nil
	}
	instance := ""
	if l == 6 {
		instance = fields[0]
	}
	return instance, &remoteexecution.Digest{
		Hash:      fields[l-2],
		SizeBytes: size,
	}
}

type byteStreamServer struct {
	blobAccess cas.BlobAccess
}

func NewByteStreamServer(blobAccess cas.BlobAccess) bytestream.ByteStreamServer {
	return &byteStreamServer{
		blobAccess: blobAccess,
	}
}

func (s *byteStreamServer) Read(in *bytestream.ReadRequest, out bytestream.ByteStream_ReadServer) error {
	log.Print("Attempted to call ByteStream.Read")
	return errors.New("Fail!")
}

func (s *byteStreamServer) Write(stream bytestream.ByteStream_WriteServer) error {
	// Store blob through blob access.
	request, err := stream.Recv()
	if err != nil {
		return err
	}
	instance, digest := parseResourceName(request.ResourceName)
	if digest == nil {
		return errors.New("Unsupported resource naming scheme")
	}
	w, err := s.blobAccess.Put(instance, digest)
	if err != nil {
		log.Print(err)
		return err
	}

	var writeOffset int64
	for {
		// Write chunk of data.
		if request.WriteOffset != writeOffset {
			w.Abandon()
			return fmt.Errorf("Attempted to write at offset %d, while %d was expected", request.WriteOffset, writeOffset)
		}
		n, err := w.Write(request.Data)
		writeOffset += int64(n)
		if err != nil {
			w.Abandon()
			log.Print(err)
			return err
		}

		// Obtain next chunk.
		request, err = stream.Recv()
		if err == io.EOF {
			return w.Close()
		}
		if err != nil {
			w.Abandon()
			log.Print(err)
			return err
		}
	}
}

func (s *byteStreamServer) QueryWriteStatus(ctx context.Context, in *bytestream.QueryWriteStatusRequest) (*bytestream.QueryWriteStatusResponse, error) {
	log.Print("Attempted to call ByteStream.QueryWriteStatus")
	return nil, errors.New("Fail!")
}

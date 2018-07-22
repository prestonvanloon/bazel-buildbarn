package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"

	"github.com/EdSchouten/bazel-buildbarn/pkg/blobstore"

	"golang.org/x/net/context"

	"google.golang.org/genproto/googleapis/bytestream"
	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	readChunkSize = 4096
)

// parseResourceNameRead parses resource name strings in one of the following two forms:
//
// - blobs/${hash}/${size}
// - ${instance}/blobs/${hash}/${size}
//
// In the process, the hash, size and instance are extracted.
func parseResourceNameRead(resourceName string) (string, *remoteexecution.Digest) {
	fields := strings.FieldsFunc(resourceName, func(r rune) bool { return r == '/' })
	l := len(fields)
	if (l != 3 && l != 4) || fields[l-3] != "blobs" {
		return "", nil
	}
	size, err := strconv.ParseInt(fields[l-1], 10, 64)
	if err != nil {
		return "", nil
	}
	instance := ""
	if l == 4 {
		instance = fields[0]
	}
	return instance, &remoteexecution.Digest{
		Hash:      fields[l-2],
		SizeBytes: size,
	}
}

// parseResourceNameWrite parses resource name strings in one of the following two forms:
//
// - uploads/${uuid}/blobs/${hash}/${size}
// - ${instance}/uploads/${uuid}/blobs/${hash}/${size}
//
// In the process, the hash, size and instance are extracted.
func parseResourceNameWrite(resourceName string) (string, *remoteexecution.Digest) {
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
	blobAccess blobstore.BlobAccess
}

func NewByteStreamServer(blobAccess blobstore.BlobAccess) bytestream.ByteStreamServer {
	return &byteStreamServer{
		blobAccess: blobAccess,
	}
}

func (s *byteStreamServer) Read(in *bytestream.ReadRequest, out bytestream.ByteStream_ReadServer) error {
	if in.ReadOffset != 0 || in.ReadLimit != 0 {
		return status.Error(codes.Unimplemented, "This service does not support downloading directory trees")
	}

	instance, digest := parseResourceNameRead(in.ResourceName)
	if digest == nil {
		return errors.New("Unsupported resource naming scheme")
	}
	r, err := s.blobAccess.Get(instance, digest)
	if err != nil {
		return err
	}

	for {
		var readBuf [readChunkSize]byte
		n, err := r.Read(readBuf[:])
		if err != nil && err != io.EOF {
			return err
		}
		if n > 0 {
			if err := out.Send(&bytestream.ReadResponse{Data: readBuf[:n]}); err != nil {
				return err
			}
		}
		if err == io.EOF {
			return nil
		}
	}
}

func (s *byteStreamServer) Write(stream bytestream.ByteStream_WriteServer) error {
	// Store blob through blob access.
	request, err := stream.Recv()
	if err != nil {
		return err
	}
	instance, digest := parseResourceNameWrite(request.ResourceName)
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

package main

import (
	"errors"
	"io"
	"log"

	"github.com/EdSchouten/bazel-buildbarn/pkg/cas"

	"golang.org/x/net/context"
	"google.golang.org/genproto/googleapis/bytestream"
)

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
	request, err := stream.Recv()
	if err != nil {
		return err
	}

	log.Print("Attempted to call ByteStream.Write ", request.ResourceName)

	for {
		_, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Print(err)
			return err
		}
	}
	log.Print("Success")
	return nil
}

func (s *byteStreamServer) QueryWriteStatus(ctx context.Context, in *bytestream.QueryWriteStatusRequest) (*bytestream.QueryWriteStatusResponse, error) {
	log.Print("Attempted to call ByteStream.QueryWriteStatus")
	return nil, errors.New("Fail!")
}

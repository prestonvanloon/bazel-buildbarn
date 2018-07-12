package main

import (
	"errors"
	"log"

	"golang.org/x/net/context"
	"google.golang.org/genproto/googleapis/bytestream"
)

type ByteStreamServer struct {
}

func (s *ByteStreamServer) Read(in *bytestream.ReadRequest, out bytestream.ByteStream_ReadServer) error {
	log.Print("Attempted to call ByteStream.Read")
	return errors.New("Fail!")
}

func (s *ByteStreamServer) Write(stream bytestream.ByteStream_WriteServer) error {
	request, err := stream.Recv()
	if err != nil {
		return err
	}

	log.Print("Attempted to call ByteStream.Write ", request.ResourceName)
	return errors.New("Fail!")
}

func (s *ByteStreamServer) QueryWriteStatus(ctx context.Context, in *bytestream.QueryWriteStatusRequest) (*bytestream.QueryWriteStatusResponse, error) {
	log.Print("Attempted to call ByteStream.QueryWriteStatus")
	return nil, errors.New("Fail!")
}

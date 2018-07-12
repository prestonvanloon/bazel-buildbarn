package main

import (
	"log"
	"net"

	"google.golang.org/genproto/googleapis/bytestream"
	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	"google.golang.org/grpc"
)

func main() {
	sock, err := net.Listen("tcp", ":8980")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	remoteexecution.RegisterActionCacheServer(s, &ActionCacheServer{})
	remoteexecution.RegisterContentAddressableStorageServer(s, &ContentAddressableStorageServer{})
	bytestream.RegisterByteStreamServer(s, &ByteStreamServer{})
	remoteexecution.RegisterExecutionServer(s, &ExecutionServer{})
	if err := s.Serve(sock); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

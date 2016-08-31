//go:generate protoc -I ./protobuf/ ./protobuf/main.proto --go_out=plugins=grpc:protobuf

package main

import (
	"net"

	log "github.com/Sirupsen/logrus"
	pb "github.com/larskluge/babl-storage/protobuf"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type server struct{}

func (s *server) Info(ctx context.Context, in *pb.Empty) (*pb.InfoResponse, error) {
	return &pb.InfoResponse{Version: "v0"}, nil
}

func main() {
	address := ":4443"

	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.WithFields(log.Fields{"error": err, "address": address}).Fatal("Failed to listen at port")
	}

	maxMsgSize := 1024 * 1024 * 2 // 2 MB max message size
	opts := []grpc.ServerOption{grpc.MaxMsgSize(maxMsgSize)}

	s := grpc.NewServer(opts...)
	pb.RegisterStorageServer(s, &server{})
	s.Serve(lis)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

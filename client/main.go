package main

import (
	"log"

	pb "github.com/larskluge/babl-storage/protobuf"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const address = "localhost:4443"

func main() {
	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewStorageClient(conn)

	// Contact the server and print out its response.
	r, err := c.Info(context.Background(), &pb.Empty{})
	if err != nil {
		log.Fatalf("no info: %v", err)
	}
	log.Printf("Version: %s", r.Version)
}

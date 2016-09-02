package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	log "github.com/Sirupsen/logrus"
	pb "github.com/larskluge/babl-storage/protobuf"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type server struct{}

func StartGrpcServer() {
	log.SetLevel(log.DebugLevel)
	lis, err := net.Listen("tcp", GrpcAddress)
	if err != nil {
		log.WithFields(log.Fields{"error": err, "address": GrpcAddress}).Fatal("Failed to listen at port")
	}

	opts := []grpc.ServerOption{grpc.MaxMsgSize(MaxMsgSize)}
	s := grpc.NewServer(opts...)
	pb.RegisterStorageServer(s, &server{})
	s.Serve(lis)
}

func (s *server) Info(ctx context.Context, in *pb.Empty) (*pb.InfoResponse, error) {
	return &pb.InfoResponse{Version: "v0"}, nil
}

func (s *server) Upload(stream pb.Storage_UploadServer) error {
	id := GenBlobId()
	key := blobKey(id)
	if cache.Exists(key) {
		log.WithFields(log.Fields{"blob_key": key}).Error("Newly generated blob key is already in use, this should not happen")
		return errors.New(fmt.Sprintf("Newly generated blob key '%s' is already in use, this should not happen", key))
	}

	log.WithFields(log.Fields{"blob_id": id}).Info("Incoming upload")

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		blob := []byte{}
		chunks := 0
		for {
			r, err := stream.Recv()
			if err == io.EOF {
				log.Info("Upload successfully received")
				err = stream.Send(&pb.UploadResponse{
					TestOneof: &pb.UploadResponse_Status{
						Status: &pb.UploadComplete{
							Success: true,
						},
					},
				})
				check(err)
				break
			} else {
				check(err)
				log.WithFields(log.Fields{"chunk_size": len(r.Chunk), "chunks": chunks}).Debug("Chunk received")
				blob = append(blob, r.Chunk...)
				writeBlob(id, r.Chunk)
				chunks += 1
			}
		}
		log.WithFields(log.Fields{"blob_size": len(blob), "chunks": chunks}).Info("Upload completed")
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		err := stream.Send(&pb.UploadResponse{
			TestOneof: &pb.UploadResponse_Blob{
				Blob: &pb.BlobInfo{
					BlobId:  id,
					BlobUrl: "http://localhost:8080/" + blobKey(id),
				},
			},
		})
		check(err)
		wg.Done()
	}()

	wg.Wait()

	return nil
}

func writeBlob(id uint64, blob []byte) error {
	r, w, err := cache.Get(blobKey(id))
	check(err)
	if w == nil {
		panic("did not get a writer..?")
	}
	w.Write(blob)
	w.Close()
	r.Close()
	return nil
}

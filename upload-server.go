package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	pb "github.com/larskluge/babl-storage/protobuf"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type server struct{}

func StartGrpcServer() {
	lis, err := net.Listen("tcp", UploadServerAddress)
	if err != nil {
		log.WithFields(log.Fields{"error": err, "address": UploadServerAddress}).Fatal("Failed to listen at port")
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
	start := time.Now()
	chunks := 0
	bytesWritten := 0
	var wg sync.WaitGroup

	id := GenBlobId()
	key := blobKey(id)
	if cache.Exists(key) {
		log.WithFields(log.Fields{"blob_key": key}).Error("Newly generated blob key is already in use, this should not happen")
		return errors.New(fmt.Sprintf("Newly generated blob key '%s' is already in use, this should not happen", key))
	}

	wg.Add(1)
	go func() {
		_, blob, err := cache.Get(blobKey(id))
		check(err)
		if blob == nil {
			panic("did not get a writer..?")
		}
		for {
			r, err := stream.Recv()
			if err == io.EOF {
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
				n, err := blob.Write(r.Chunk)
				check(err)
				// blob.Flush()
				bytesWritten += n
				chunks += 1
			}
		}
		blob.Close()
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		err := stream.Send(&pb.UploadResponse{
			TestOneof: &pb.UploadResponse_Blob{
				Blob: &pb.BlobInfo{
					BlobId:  id,
					BlobUrl: "http://babl.sh" + FileServerAddress + "/" + blobKey(id),
				},
			},
		})
		check(err)
		wg.Done()
	}()

	wg.Wait()
	elapsed_ms := time.Since(start).Nanoseconds() / 1e6
	log.WithFields(log.Fields{"blob_id": id, "blob_size": bytesWritten, "chunks": chunks, "duration_ms": elapsed_ms}).Info("Blob upload completed")

	return nil
}

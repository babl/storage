package main

import (
	"errors"
	"fmt"
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

func (s *server) Info(ctx context.Context, in *pb.InfoRequest) (*pb.InfoResponse, error) {
	return &pb.InfoResponse{Version: Version}, nil
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
			check(err)
			log.WithFields(log.Fields{"chunk_size": len(r.Chunk), "chunks": chunks}).Debug("Chunk received")
			n, err := blob.Write(r.Chunk)
			check(err)
			// blob.Flush()
			bytesWritten += n
			chunks += 1

			if r.Complete {
				success := r.TotalBytesSent == uint64(bytesWritten)
				errMsg := ""
				if success {
					log.Info("Upload completed successful")
				} else {
					errMsg = fmt.Sprintf("Client reports different blob size (%d bytes) than written to disk on server side (%d).", r.TotalBytesSent, bytesWritten)
					log.Error(errMsg)
				}
				// final upload response
				err = stream.Send(&pb.UploadResponse{
					BlobId:   id,
					BlobUrl:  BlobUrl(id),
					Complete: true,
					Success:  success,
					Error:    errMsg,
				})
				check(err)
				break
			}
		}
		blob.Close()
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		err := stream.Send(&pb.UploadResponse{
			BlobId:   id,
			BlobUrl:  BlobUrl(id),
			Complete: false,
		})
		check(err)
		wg.Done()
	}()

	wg.Wait()
	elapsed_ms := time.Since(start).Nanoseconds() / 1e6
	log.WithFields(log.Fields{"blob_id": id, "blob_size": bytesWritten, "chunks": chunks, "duration_ms": elapsed_ms}).Info("Blob upload complete")

	return nil
}

func BlobUrl(blobId uint64) string {
	return "http://babl.sh" + FileServerAddress + "/" + blobKey(blobId)
}

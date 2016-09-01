//go:generate protoc -I ./protobuf/ ./protobuf/main.proto --go_out=plugins=grpc:protobuf

package main

import (
	"io"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	pb "github.com/larskluge/babl-storage/protobuf"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"gopkg.in/djherbis/fscache.v0"
)

type server struct{}

var random = rand.New(rand.NewSource(time.Now().UnixNano()))
var cache fscache.Cache

func (s *server) Info(ctx context.Context, in *pb.Empty) (*pb.InfoResponse, error) {
	return &pb.InfoResponse{Version: "v0"}, nil
}

func GenBlobId() uint64 {
	return uint64(random.Uint32())<<32 + uint64(random.Uint32())
}

func (s *server) Upload(stream pb.Storage_UploadServer) error {
	id := GenBlobId()
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
				chunks += 1
			}
		}
		log.WithFields(log.Fields{"blob_size": len(blob), "chunks": chunks}).Info("Upload completed")
		writeBlob(id, blob)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		err := stream.Send(&pb.UploadResponse{
			TestOneof: &pb.UploadResponse_Blob{
				Blob: &pb.BlobInfo{
					BlobId:  id,
					BlobUrl: "http://localhost:4443/blobs/" + blobKey(id),
				},
			},
		})
		check(err)
		wg.Done()
	}()

	wg.Wait()

	return nil
}

func blobKey(id uint64) string {
	return strconv.FormatUint(id, 16)
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

func getBlob(id uint64) io.Reader {
	r, _, err := cache.Get(blobKey(id))
	check(err)
	return r
}

func main() {
	address := ":4443"

	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.WithFields(log.Fields{"error": err, "address": address}).Fatal("Failed to listen at port")
	}

	cache, err = fscache.New("./cache", 0755, 1*time.Minute)
	check(err)

	maxMsgSize := 1024 * 1024 * 2 // 2 MB max message size
	opts := []grpc.ServerOption{grpc.MaxMsgSize(maxMsgSize)}

	log.Printf("Server started at %s\n", address)
	s := grpc.NewServer(opts...)
	pb.RegisterStorageServer(s, &server{})
	s.Serve(lis)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

package uploader

import (
	"bufio"
	"io"
	"sync"

	log "github.com/Sirupsen/logrus"
	pb "github.com/larskluge/babl-storage/protobuf"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const chunkSize = 1024 * 100 // 100 kb

func Upload(address string, blob io.Reader) error {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Error("did not connect: %v", err)
		return err
	}
	defer conn.Close()
	c := pb.NewStorageClient(conn)

	// Upload
	stream, err := c.Upload(context.Background())
	check(err)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				break
			} else {
				check(err)
				switch resp.TestOneof.(type) {
				case *pb.UploadResponse_Blob:
					blob := resp.GetBlob()
					log.WithFields(log.Fields{"blob_id": blob.BlobId, "blob_url": blob.BlobUrl}).Info("Upload Id")
				case *pb.UploadResponse_Status:
					if resp.GetStatus().Success {
						log.Println("Server confirmed upload successful")
					} else {
						panic("Server: upload not successful")
					}
				}
			}
		}
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		reader := bufio.NewReader(blob)
		bytesRead := 0

		for {
			chunk := make([]byte, chunkSize)
			n, err := reader.Read(chunk)
			bytesRead += n

			lastChunk := false
			if err == io.EOF {
				lastChunk = true
				err = nil
			}
			check(err)

			if n > 0 {
				if n < chunkSize {
					chunk = chunk[:n]
				}

				req := pb.UploadRequest{
					Chunk:    chunk,
					Complete: false,
				}

				err = stream.Send(&req)
				check(err)
			}

			if lastChunk {
				break
			}
		}

		err = stream.CloseSend()
		check(err)

		wg.Done()
		log.WithFields(log.Fields{"bytes_read_from_stdin": bytesRead}).Info("Upload done")
	}()

	wg.Wait()
	return nil
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
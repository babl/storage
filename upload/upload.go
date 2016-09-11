package upload

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

type Upload struct {
	Id       uint64
	Url      string
	Complete bool
	Success  bool
	Error    string

	conn           grpc.ClientConn
	stream         pb.Storage_UploadClient
	blob           io.Reader
	completionCond sync.Cond
}

func New(address string, blob io.Reader) (*Upload, error) {
	var m sync.Mutex
	m.Lock()
	obj := Upload{Complete: false, Success: false, Error: "", completionCond: *sync.NewCond(&m)}
	err := obj.startUploading(address, blob)
	if err == nil {
		log.WithFields(log.Fields{"blob_id": obj.Id, "blob_url": obj.Url}).Info("Store large payload externally")
		return &obj, nil
	} else {
		return nil, err
	}
}

func (up *Upload) handleIncomingData(metadataAvailable *sync.Cond) {
	for {
		resp, err := up.stream.Recv()
		check(err)
		if resp.Complete {
			up.Complete = true
			up.Success = resp.Success
			up.completionCond.Broadcast()
			if resp.Success {
				break
			} else {
				panic("Server: upload not successful")
			}
		} else {
			up.Id = resp.BlobId
			up.Url = resp.BlobUrl
			metadataAvailable.Broadcast()
		}
	}
}

func (up *Upload) handleOutgoingData(blob io.Reader) {
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

		if n < chunkSize {
			chunk = chunk[:n]
		}

		req := pb.UploadRequest{
			Chunk:          chunk,
			TotalBytesSent: uint64(bytesRead),
			Complete:       lastChunk,
		}

		err = up.stream.Send(&req)
		check(err)

		if lastChunk {
			break
		}
	}

	err := up.stream.CloseSend()
	check(err)
}

func (up *Upload) startUploading(address string, blob io.Reader) error {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return err
	}
	up.conn = *conn
	c := pb.NewStorageClient(conn)

	// Upload
	stream, err := c.Upload(context.Background())
	check(err)
	up.stream = stream

	var m sync.Mutex
	m.Lock()
	metadataAvailable := sync.NewCond(&m)
	go up.handleIncomingData(metadataAvailable)

	go up.handleOutgoingData(blob)

	metadataAvailable.Wait()
	return nil
}

func (up *Upload) WaitForCompletion() bool {
	if !up.Complete {
		up.completionCond.Wait()
		up.conn.Close() // TODO: make sure close is called only and at least once
	}
	if !up.Success {
		log.WithFields(log.Fields{"blob_id": up.Id, "blob_url": up.Url}).Error("Large payload upload failed")
	}
	return up.Success
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
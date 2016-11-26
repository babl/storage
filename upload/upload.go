package upload

import (
	"bufio"
	"errors"
	"io"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	. "github.com/larskluge/babl-storage/blob"
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
		log.WithFields(log.Fields{"blob_key": BlobKey(obj.Id), "blob_url": obj.Url}).Info("Store large payload externally")
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
	if err != nil {
		log.WithError(err).Warn("upload: handleOutgoingData: closing stream unsuccessful")
	}
}

func (up *Upload) startUploading(address string, blob io.Reader) error {
	var errc = make(chan error, 1)
	defer close(errc)

	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return err
	}
	up.conn = *conn
	c := pb.NewStorageClient(conn)

	// Upload
	stream, err := c.Upload(context.Background())
	check(err) // FIXME what if babl-storage is down ?
	up.stream = stream

	var m sync.Mutex
	m.Lock()
	metadataAvailable := sync.NewCond(&m)
	go up.handleIncomingData(metadataAvailable)
	go up.handleOutgoingData(blob)

	timeout := time.AfterFunc(10*time.Second, func() {
		errc <- errors.New("Timeout failure trying to get upload metadata")
	})
	defer timeout.Stop()

	go func(metadataAvailable *sync.Cond) {
		metadataAvailable.Wait()
		errc <- nil
	}(metadataAvailable)

	return <-errc
}

func (up *Upload) WaitForCompletion() bool {
	if !up.Complete {
		up.completionCond.Wait()
		err := up.conn.Close() // TODO: make sure close is called only and at least once
		if err != nil {
			log.WithError(err).Error("Closing upload connection failed")
		}
	}
	if !up.Success {
		log.WithFields(log.Fields{"blob_key": BlobKey(up.Id), "blob_url": up.Url}).Error("Large payload upload failed")
	}
	return up.Success
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

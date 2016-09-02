//go:generate protoc -I ./protobuf/ ./protobuf/main.proto --go_out=plugins=grpc:protobuf

package main

import (
	"io"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/djherbis/fscache.v0"
)

const (
	MaxMsgSize  = 1024 * 1024 * 2 // 2 MB max message size
	GrpcAddress = ":4443"
)

var (
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
	cache  fscache.Cache
)

func GenBlobId() uint64 {
	return uint64(random.Uint32())<<32 + uint64(random.Uint32())
}

func blobKey(id uint64) string {
	return strconv.FormatUint(id, 16)
}

func getBlob(id uint64) io.Reader {
	r, _, err := cache.Get(blobKey(id))
	check(err)
	return r
}

func main() {
	c := make(chan os.Signal, 1)

	var err error
	cache, err = fscache.New("./cache", 0755, 1*time.Minute)
	check(err)

	log.Infof("File Server starts at :8080")
	go func() {
		StartFileServer()
		c <- syscall.SIGTERM
	}()

	log.Infof("Grpc Server starts at %s", GrpcAddress)
	go func() {
		StartGrpcServer()
		c <- syscall.SIGTERM
	}()

	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	os.Exit(1)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

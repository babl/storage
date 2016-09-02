//go:generate protoc -I ./protobuf/ ./protobuf/main.proto --go_out=plugins=grpc:protobuf

package main

import (
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/djherbis/fscache.v0"
)

const (
	MaxMsgSize     = 1024 * 1024 * 2 // 2 MB max message size
	GrpcAddress    = ":4443"
	KeepUploadsFor = 15 * time.Second
)

var (
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
	cache  fscache.Cache
)

func main() {
	c := make(chan os.Signal, 1)

	var err error
	cache, err = fscache.New("./cache", 0755, KeepUploadsFor)
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

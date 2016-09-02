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
	MaxMsgSize          = 1024 * 1024 * 2 // 2 MB max message size
	UploadServerAddress = ":4443"
	FileServerAddress   = ":4442"
	KeepUploadsFor      = 15 * time.Second
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

	log.Infof("File Server starts at %s", FileServerAddress)
	go func() {
		StartFileServer()
		c <- syscall.SIGTERM
	}()

	log.Infof("Upload Server starts at %s", UploadServerAddress)
	go func() {
		StartGrpcServer()
		c <- syscall.SIGTERM
	}()

	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	os.Exit(1)
}

//go:generate protoc -I ./protobuf/ ./protobuf/main.proto --go_out=plugins=grpc:protobuf

package main

import (
	"flag"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/djherbis/fscache.v0"
)

const (
	Version             = "0.1.0"
	MaxMsgSize          = 1024 * 1024 * 2 // 2 MB max message size
	UploadServerAddress = ":4443"
	FileServerAddress   = "localhost:4442"
	KeepUploadsFor      = 1 * time.Hour
)

var (
	debugFlag     = flag.Bool("debug", false, "Debug mode")
	logFormatFlag = flag.String("log-format", "default", "Log format, options: default, json")

	random = rand.New(rand.NewSource(time.Now().UnixNano()))
	cache  fscache.Cache
)

func main() {
	flag.Parse()
	if *debugFlag {
		log.SetLevel(log.DebugLevel)
	}
	if *logFormatFlag == "json" {
		log.SetFormatter(&log.JSONFormatter{})
	}

	var err error
	cache, err = fscache.New("./cache", 0755, KeepUploadsFor)
	check(err)

	c := make(chan os.Signal, 1)

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

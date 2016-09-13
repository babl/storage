//go:generate protoc -I ./protobuf/ ./protobuf/main.proto --go_out=plugins=grpc:protobuf

package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/djherbis/fscache.v0"
)

const (
	Version        = "0.1.1"
	MaxMsgSize     = 1024 * 1024 * 2 // 2 MB max message size
	KeepUploadsFor = 1 * time.Hour
)

var (
	fileServerAddressFlag   = flag.String("file-server-address", ":4442", "Address to start the file server at")
	uploadServerAddressFlag = flag.String("upload-server-address", ":4443", "Address to start the upload server at")
	blobUrlTmplFlag         = flag.String("blob-url-template", "http://localhost:4442/%s", "Template for public accessible blob url, %s is replaced with blob key")
	cacheDirFlag            = flag.String("cache-dir", "./cache", "Path to cache directory used for upload blob storage")
	logFormatFlag           = flag.String("log-format", "default", "Log format, options: default, json")
	debugFlag               = flag.Bool("debug", false, "Debug mode")
	versionFlag             = flag.Bool("version", false, "Print version and exit")

	random = rand.New(rand.NewSource(time.Now().UnixNano()))
	cache  fscache.Cache
)

func main() {
	flag.Parse()

	if *versionFlag {
		fmt.Println(Version)
		os.Exit(0)
	}

	if *debugFlag {
		log.SetLevel(log.DebugLevel)
	}
	if *logFormatFlag == "json" {
		log.SetFormatter(&log.JSONFormatter{})
	}

	var err error
	cache, err = fscache.New(*cacheDirFlag, 0755, KeepUploadsFor)
	check(err)

	c := make(chan os.Signal, 1)

	log.Infof("File Server starts at %s", *fileServerAddressFlag)
	go func() {
		StartFileServer()
		c <- syscall.SIGTERM
	}()

	log.Infof("Upload Server starts at %s", *uploadServerAddressFlag)
	go func() {
		StartGrpcServer()
		c <- syscall.SIGTERM
	}()

	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	os.Exit(1)
}

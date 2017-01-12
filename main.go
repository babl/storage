//go:generate protoc -I ./protobuf/ ./protobuf/main.proto --go_out=plugins=grpc:protobuf

package main

import (
	"flag"
	"fmt"
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
	restartTimeoutFlag      = flag.String("restart-timeout", "0s", "Timeout after each babl-storage restarts (defaults to none)")

	cache          fscache.Cache
	RestartTimeout time.Duration
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

	if RestartTimeout, err = time.ParseDuration(*restartTimeoutFlag); err != nil {
		panic("restart-timeout: Restart timeout not a valid duration")
	}

	c := make(chan os.Signal, 1)

	go func() {
		StartFileServer()
		c <- syscall.SIGTERM
	}()

	go func() {
		StartGrpcServer()
		c <- syscall.SIGTERM
	}()

	if nullRestartTimeout, _ := time.ParseDuration("0s"); nullRestartTimeout != RestartTimeout {
		scheduleRestart()
	}

	log.WithFields(log.Fields{"version": Version, "file_server_address": *fileServerAddressFlag, "upload_server_address": *uploadServerAddressFlag, "blob_url_template": *blobUrlTmplFlag, "debug": *debugFlag}).Warn("Babl Storage Started")
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	os.Exit(1)
}

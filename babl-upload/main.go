package main

import (
	"flag"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/larskluge/babl-storage/upload"
	"github.com/mattn/go-isatty"
)

var endpointFlag = flag.String("endpoint", "localhost:4443", "Connect to endpoint")

func main() {
	flag.Parse()
	if isatty.IsTerminal(os.Stdin.Fd()) {
		panic("No stdin attached")
	}

	upload, err := upload.New(*endpointFlag, os.Stdin)
	check(err)
	log.WithFields(log.Fields{"blob_id": upload.Id, "blob_url": upload.Url}).Info("Upload Id")
	success := upload.WaitForCompletion()
	if success {
		log.Info("Server confirmed upload successful")
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

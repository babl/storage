package main

import (
	"flag"
	"os"

	"github.com/larskluge/babl-storage/uploader"
	"github.com/mattn/go-isatty"
)

var endpointFlag = flag.String("endpoint", "localhost:4443", "Connect to endpoint")

func main() {
	flag.Parse()
	if isatty.IsTerminal(os.Stdin.Fd()) {
		panic("No stdin attached")
	}

	err := uploader.Upload(*endpointFlag, os.Stdin)
	check(err)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

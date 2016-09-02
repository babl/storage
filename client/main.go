package main

import (
	"os"

	"github.com/larskluge/babl-storage/uploader"
	"github.com/mattn/go-isatty"
)

const address = "localhost:4443"

func main() {
	if isatty.IsTerminal(os.Stdin.Fd()) {
		panic("No stdin attached")
	}

	err := uploader.Upload(address, os.Stdin)
	check(err)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

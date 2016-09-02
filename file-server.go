package main

import (
	"fmt"
	"io"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
)

func StartFileServer() {
	http.HandleFunc("/", handler)
	err := http.ListenAndServe(FileServerAddress, nil)
	check(err)
}

func handler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	key := r.URL.Path[1:]
	blob, err := getBlobStream(key)
	check(err)
	status := http.StatusNotFound
	if blob == nil {
		w.WriteHeader(status)
		fmt.Fprintf(w, "Blob %s not found.", key)
	} else {
		status = http.StatusOK
		io.Copy(w, blob)
	}

	elapsed_ms := time.Since(start).Nanoseconds() / 1e6
	log.WithFields(log.Fields{"status": status, "key": key, "duration_ms": elapsed_ms}).Info("File request served")
}
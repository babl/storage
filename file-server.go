package main

import (
	"fmt"
	"io"
	"net/http"

	log "github.com/Sirupsen/logrus"
)

func StartFileServer() {
	http.HandleFunc("/", handler)
	err := http.ListenAndServe(":8080", nil)
	check(err)
}

func handler(w http.ResponseWriter, r *http.Request) {
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
	log.Infof("GET %d %s", status, r.URL.Path)
}

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
	log.Infof("GET %s", r.URL.Path)
	key := r.URL.Path[1:]
	blob, err := getBlobStream(key)
	check(err)

	if blob == nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Blob %s not found.", key)
	} else {
		io.Copy(w, blob)
	}

	log.Infof("done w/ GET %s", r.URL.Path)
}

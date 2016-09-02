package main

import (
	"io"
	"strconv"
)

func GenBlobId() uint64 {
	return uint64(random.Uint32())<<32 + uint64(random.Uint32())
}

func blobKey(id uint64) string {
	return strconv.FormatUint(id, 16)
}

func getBlob(id uint64) io.Reader {
	r, _, err := cache.Get(blobKey(id))
	check(err)
	return r
}

func getBlobStream(key string) (io.Reader, error) {
	if cache.Exists(key) {
		r, _, err := cache.Get(key)
		return r, err
	} else {
		return nil, nil
	}
}

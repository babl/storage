package blob

import (
	"math/rand"
	"strconv"
	"time"
)

var random = rand.New(rand.NewSource(time.Now().UnixNano()))

func GenBlobId() uint64 {
	return uint64(random.Uint32())<<32 + uint64(random.Uint32())
}

func BlobKey(id uint64) string {
	return strconv.FormatUint(id, 16)
}

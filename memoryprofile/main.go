package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"log"
	"net/http"
	_ "net/http/pprof"
	"strconv"
	"sync"
)

const (
	defaultBytes = 4 << 20
	maxBytes     = 16 << 20
	chunkBytes   = 32 << 10
)

var (
	pattern    = []byte("0123456789abcdef")
	retained   [][]byte
	retainedMu sync.Mutex
)

type response struct {
	Bytes  int    `json:"bytes"`
	SHA256 string `json:"sha256"`
}

func main() {
	http.HandleFunc("/slow", slowHandler)
	http.HandleFunc("/fast", fastHandler)
	log.Println("listening on :8081 (pprof at /debug/pprof/)")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

func requestedBytes(w http.ResponseWriter, r *http.Request) (int, bool) {
	bytes := defaultBytes
	if raw := r.URL.Query().Get("bytes"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > maxBytes {
			http.Error(w, fmt.Sprintf("bytes must be an integer from 1 to %d", maxBytes), http.StatusBadRequest)
			return 0, false
		}
		bytes = parsed
	}
	return bytes, true
}

func slowHandler(w http.ResponseWriter, r *http.Request) {
	n, ok := requestedBytes(w, r)
	if !ok {
		return
	}

	// Intentionally allocate one request-sized buffer and retain it forever,
	// simulating an unbounded in-memory cache. Heap profiles will show this
	// allocation as both high alloc_space and growing inuse_space.
	data := make([]byte, n)
	fill(data)
	digest := sha256.Sum256(data)
	retainedMu.Lock()
	retained = append(retained, data)
	retainedMu.Unlock()
	writeResponse(w, n, digest[:])
}

func fastHandler(w http.ResponseWriter, r *http.Request) {
	n, ok := requestedBytes(w, r)
	if !ok {
		return
	}

	// Hash fixed-size chunks rather than building or retaining the full payload.
	h := sha256.New()
	writePayload(h, n)
	writeResponse(w, n, h.Sum(nil))
}

func fill(dst []byte) {
	for i := range dst {
		dst[i] = pattern[i%len(pattern)]
	}
}

func writePayload(h hash.Hash, n int) {
	var chunk [chunkBytes]byte
	fill(chunk[:])
	for n > 0 {
		length := min(n, len(chunk))
		_, _ = h.Write(chunk[:length])
		n -= length
	}
}

func writeResponse(w http.ResponseWriter, n int, digest []byte) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response{Bytes: n, SHA256: hex.EncodeToString(digest)})
}

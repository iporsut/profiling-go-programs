package main

import (
	"encoding/json"
	"log"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"strconv"
	"sync"
)

const maxWorkers = 200

type response struct {
	Workers int `json:"workers"`
	Result  int `json:"result"`
}

func main() {
	// Enable these profiles explicitly; they are disabled or sampled sparsely by
	// default because collecting them has runtime overhead.
	runtime.SetBlockProfileRate(1)
	runtime.SetMutexProfileFraction(1)

	http.HandleFunc("/block/slow", blockSlowHandler)
	http.HandleFunc("/block/fast", blockFastHandler)
	http.HandleFunc("/mutex/slow", mutexSlowHandler)
	http.HandleFunc("/mutex/fast", mutexFastHandler)
	http.HandleFunc("/goroutine/slow", goroutineSlowHandler)
	http.HandleFunc("/goroutine/fast", goroutineFastHandler)
	log.Println("listening on :8082 (pprof at /debug/pprof/)")
	log.Fatal(http.ListenAndServe(":8082", nil))
}

func workerCount(w http.ResponseWriter, r *http.Request) (int, bool) {
	n := 50
	if raw := r.URL.Query().Get("workers"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > maxWorkers {
			http.Error(w, "workers must be an integer from 1 to 200", http.StatusBadRequest)
			return 0, false
		}
		n = parsed
	}
	return n, true
}

func blockSlowHandler(w http.ResponseWriter, r *http.Request) {
	n, ok := workerCount(w, r)
	if !ok {
		return
	}
	results := make(chan int)
	for i := 0; i < n; i++ {
		go func(value int) { results <- value }(i)
	}
	// Only one result is received. The remaining senders block forever.
	writeJSON(w, response{Workers: n, Result: <-results})
}

func blockFastHandler(w http.ResponseWriter, r *http.Request) {
	n, ok := workerCount(w, r)
	if !ok {
		return
	}
	results := make(chan int, n)
	for i := 0; i < n; i++ {
		go func(value int) { results <- value }(i)
	}
	total := 0
	for i := 0; i < n; i++ {
		total += <-results
	}
	writeJSON(w, response{Workers: n, Result: total})
}

func mutexSlowHandler(w http.ResponseWriter, r *http.Request) {
	n, ok := workerCount(w, r)
	if !ok {
		return
	}
	var mu sync.Mutex
	shared := 0
	var group sync.WaitGroup
	for i := 0; i < n; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for j := 0; j < 20_000; j++ {
				mu.Lock()
				shared++
				mu.Unlock()
			}
		}()
	}
	group.Wait()
	writeJSON(w, response{Workers: n, Result: shared})
}

func mutexFastHandler(w http.ResponseWriter, r *http.Request) {
	n, ok := workerCount(w, r)
	if !ok {
		return
	}
	partials := make([]int, n)
	var group sync.WaitGroup
	for i := range partials {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			for j := 0; j < 20_000; j++ {
				partials[index]++
			}
		}(i)
	}
	group.Wait()
	total := 0
	for _, value := range partials {
		total += value
	}
	writeJSON(w, response{Workers: n, Result: total})
}

func goroutineSlowHandler(w http.ResponseWriter, r *http.Request) {
	n, ok := workerCount(w, r)
	if !ok {
		return
	}
	neverClosed := make(chan struct{})
	for i := 0; i < n; i++ {
		go func() { <-neverClosed }()
	}
	writeJSON(w, response{Workers: n, Result: n})
}

func goroutineFastHandler(w http.ResponseWriter, r *http.Request) {
	n, ok := workerCount(w, r)
	if !ok {
		return
	}
	done := make(chan struct{})
	var group sync.WaitGroup
	for i := 0; i < n; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			select {
			case <-done:
				return
			}
		}()
	}
	close(done)
	group.Wait()
	writeJSON(w, response{Workers: n, Result: 0})
}

func writeJSON(w http.ResponseWriter, value response) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

package main

import (
	"encoding/json"
	"log"
	"net/http"
	_ "net/http/pprof"
	"strconv"
	"time"
)

const defaultN = 40

type response struct {
	Algorithm string        `json:"algorithm"`
	N         int           `json:"n"`
	Result    uint64        `json:"result"`
	Elapsed   time.Duration `json:"elapsed_ns"`
}

func main() {
	http.HandleFunc("/slow", workload("recursive", fibonacciRecursive))
	http.HandleFunc("/fast", workload("iterative", fibonacciIterative))
	log.Println("listening on :8080 (pprof at /debug/pprof/)")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func workload(name string, calculate func(int) uint64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		n := defaultN
		if raw := r.URL.Query().Get("n"); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed < 0 || parsed > 45 {
				http.Error(w, "n must be an integer from 0 to 45", http.StatusBadRequest)
				return
			}
			n = parsed
		}

		start := time.Now()
		result := calculate(n)
		elapsed := time.Since(start)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response{
			Algorithm: name,
			N:         n,
			Result:    result,
			Elapsed:   elapsed,
		})
	}
}

// fibonacciRecursive intentionally repeats work: its exponential call tree is
// the CPU bottleneck this demo is designed to make visible in a CPU profile.
func fibonacciRecursive(n int) uint64 {
	if n < 2 {
		return uint64(n)
	}
	return fibonacciRecursive(n-1) + fibonacciRecursive(n-2)
}

// fibonacciIterative computes the same result in linear time and constant space.
func fibonacciIterative(n int) uint64 {
	var previous, current uint64
	for i := 0; i < n; i++ {
		previous, current = current, previous+current
	}
	return previous
}

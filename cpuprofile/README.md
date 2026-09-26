# Go CPU profiling demo

This HTTP service exposes the same Fibonacci workload in two forms:

- `/slow?n=40` uses naive recursion and deliberately repeats calculations.
- `/fast?n=40` uses an iterative algorithm with linear work.
- `/debug/pprof/` exposes Go's standard profiling handlers.

Run it with `go run .`, then load the slow endpoint from another terminal:

```sh
for i in $(seq 1 8); do curl -s 'http://localhost:8080/slow?n=40' >/dev/null & done
wait
```

Collect a 10-second CPU profile while generating load:

```sh
go tool pprof -http=:0 http://localhost:8080/debug/pprof/profile?seconds=10
```

In the pprof web view, inspect the flame graph or top functions. The recursive
Fibonacci function should dominate. The repeated subproblems in that call tree
are the reason it consumes so much CPU.

Switch the load URL to `/fast?n=40` and capture another profile to compare.
The iterative endpoint computes the same result without the exponential call
tree. You can also compare response timings directly:

```sh
curl -s 'http://localhost:8080/slow?n=40'
curl -s 'http://localhost:8080/fast?n=40'
```

The endpoints accept `n` from 0 to 45; the default is 40. The bound keeps the
intentional slow implementation usable as a local demo.

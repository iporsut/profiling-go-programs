# Go concurrency profiling demo

Run this service from this directory with `go run .`. It listens on port 8082
and enables Go's block and mutex profile sampling at startup. The standard
`/debug/pprof/` handlers also expose goroutine profiles.

Each issue has a paired endpoint:

| Profile | Problem endpoint | Improved endpoint | What to look for |
| --- | --- | --- | --- |
| Block | `/block/slow` | `/block/fast` | Slow leaves channel senders waiting; fast receives every result. |
| Mutex | `/mutex/slow` | `/mutex/fast` | Slow updates one shared counter under a lock; fast counts locally and combines after workers finish. |
| Goroutine | `/goroutine/slow` | `/goroutine/fast` | Slow leaves goroutines waiting on a channel that is never closed; fast signals and joins its workers. |

The endpoints accept `workers` from 1 to 200 and default to 50. For a mutex
profile, generate contention with concurrent requests:

```sh
for i in $(seq 1 4); do curl -s 'http://localhost:8082/mutex/slow?workers=100' >/dev/null & done
wait
```

Capture and inspect profiles with pprof:

```sh
go tool pprof -http=:0 'http://localhost:8082/debug/pprof/block'
go tool pprof -http=:0 'http://localhost:8082/debug/pprof/mutex'
go tool pprof -http=:0 'http://localhost:8082/debug/pprof/goroutine'
```

Block and mutex profiles are cumulative, so collect one after exercising the
matching slow endpoint, then restart the service before comparing the paired
fast endpoint. The slow goroutine endpoint intentionally leaks up to 200
goroutines per request; use it only in this local demo, and restart the service
to clear them before examining the fast endpoint.

# Go memory profiling demo

This HTTP service runs on port 8081 and exposes two endpoints that process the
same deterministic payload:

- `/slow?bytes=4194304` allocates a full payload buffer and retains it forever,
  simulating an unbounded in-memory cache.
- `/fast?bytes=4194304` hashes the payload in fixed-size chunks and keeps only a
  small working buffer.
- `/debug/pprof/` exposes Go's standard profiling handlers.

Run it from this directory with `go run .`. Each request defaults to 4 MiB and
accepts a size from 1 byte through 16 MiB.

Generate several slow requests to grow live heap:

```sh
for i in $(seq 1 8); do curl -s 'http://localhost:8081/slow' >/dev/null; done
```

Open the heap profile in pprof:

```sh
go tool pprof -http=:0 'http://localhost:8081/debug/pprof/heap?gc=1'
```

Inspect `inuse_space` to find the retained buffers and `alloc_space` to see
cumulative allocation volume. The slow handler should be prominent. Compare
against `/fast` by restarting the service first, then issue several `/fast`
requests and capture another profile. Restarting clears the intentionally
retained buffers from the slow run.

The endpoints return the payload size and SHA-256 digest rather than returning
the payload itself. Both digests should match for the same size, while the fast
handler avoids allocating or keeping a full request-sized byte slice.

# force-gzip upstream (minimal)

Minimal Go upstream server used to **force compressed responses** for reproducing
the `ProcessStreamingResponseBody` behavior.

## Endpoints

- `GET /healthz` -> plain `ok`
- `GET /gzip/json` -> always returns gzip-compressed JSON (`Content-Encoding: gzip`)
- `GET /gzip/sse` -> always returns gzip-compressed SSE stream (`Content-Encoding: gzip`)

## Quick verification

```
# Expect: Content-Encoding: gzip
curl -i http://127.0.0.1:18080/gzip/json

# Stream endpoint (compressed SSE)
curl -i "http://127.0.0.1:18080/gzip/sse?chunks=3&delayMs=200"
```

Tip: use `--compressed` if your client should auto-decompress when displaying body.

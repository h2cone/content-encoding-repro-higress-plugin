# content-encoding-repro-higress-plugin

Minimal plugin to verify when `ProcessStreamingResponseBody` is executed or skipped.

## What this plugin verifies

The key condition is the **response header** `Content-Encoding`:

- If response has `Content-Encoding` (for example `gzip`), callback may be skipped.
- If response has no `Content-Encoding`, callback is executed normally.

`Accept-Encoding` on request is only an indirect factor (it may cause upstream to choose compressed response).

## What it logs

- Request `Accept-Encoding`
- Response `content-encoding` and `:status`
- Final result in `onHttpStreamDone`
  - `callback=EXECUTED`
  - `callback=NOT_EXECUTED`

## Build

Windows (PowerShell):

```powershell
$env:GOOS="wasip1"; $env:GOARCH="wasm"; go build -buildmode=c-shared -o main.wasm ./
```

macOS / Linux (bash/zsh):

```bash
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o main.wasm ./
```

## Repro (precise)

1. Attach this plugin to a test route and enable `debugMode`.
2. Route to an upstream that always returns compressed response (`Content-Encoding: gzip`).
3. Send request (request `Accept-Encoding` can be empty or non-empty).
4. Check gateway log `stream done` line, expected: `callback=NOT_EXECUTED` with `response.content-encoding="gzip"`.
5. Route to an upstream/endpoint that returns uncompressed response (`Content-Encoding` absent).
6. Check gateway log again, expected: `callback=EXECUTED` with `response.content-encoding=""`.

The `upstream` folder provides a minimal Go service for step 2:

- `upstream/main.go`
- `upstream/README.md`

## Example Config

```json
{
  "debugMode": true
}
```

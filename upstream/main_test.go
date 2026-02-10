package main

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleGzipJSON_AlwaysCompressed(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/gzip/json", nil)

	handleGzipJSON(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", response.StatusCode)
	}
	if got := response.Header.Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("unexpected content-encoding: %q", got)
	}
	if got := response.Header.Get("Vary"); !strings.Contains(got, "Accept-Encoding") {
		t.Fatalf("unexpected vary header: %q", got)
	}

	reader, err := gzip.NewReader(response.Body)
	if err != nil {
		t.Fatalf("create gzip reader failed: %v", err)
	}
	defer reader.Close()

	plain, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read gzip body failed: %v", err)
	}
	text := string(plain)
	if !strings.Contains(text, `"ok":true`) {
		t.Fatalf("unexpected body: %s", text)
	}
}

func TestHandleGzipSSE_AlwaysCompressed(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/gzip/sse?chunks=2&delayMs=0", nil)

	handleGzipSSE(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", response.StatusCode)
	}
	if got := response.Header.Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("unexpected content-encoding: %q", got)
	}
	if got := response.Header.Get("Content-Type"); !strings.Contains(got, "text/event-stream") {
		t.Fatalf("unexpected content-type: %q", got)
	}

	reader, err := gzip.NewReader(response.Body)
	if err != nil {
		t.Fatalf("create gzip reader failed: %v", err)
	}
	defer reader.Close()

	plain, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read gzip body failed: %v", err)
	}
	text := string(plain)
	if !strings.Contains(text, "data:") {
		t.Fatalf("unexpected sse body: %s", text)
	}
	if !strings.Contains(text, "[DONE]") {
		t.Fatalf("missing done marker: %s", text)
	}
}

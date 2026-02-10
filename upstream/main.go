package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type jsonResponse struct {
	OK        bool   `json:"ok"`
	Path      string `json:"path"`
	Timestamp string `json:"timestamp"`
}

func main() {
	port := getEnv("PORT", "18080")
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("ok"))
	})

	mux.HandleFunc("/gzip/json", handleGzipJSON)
	mux.HandleFunc("/gzip/sse", handleGzipSSE)

	address := ":" + port
	log.Printf("force-gzip upstream listening on %s", address)
	log.Printf("endpoints: GET /healthz, /gzip/json, /gzip/sse")

	if err := http.ListenAndServe(address, logRequest(mux)); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

func handleGzipJSON(writer http.ResponseWriter, request *http.Request) {
	payload := jsonResponse{
		OK:        true,
		Path:      request.URL.Path,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		http.Error(writer, "marshal error", http.StatusInternalServerError)
		return
	}

	writeGzip(writer, http.StatusOK, "application/json; charset=utf-8", body)
}

func handleGzipSSE(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")
	writer.Header().Set("X-Accel-Buffering", "no")

	gzipWriter, err := startGzip(writer, "text/event-stream; charset=utf-8")
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	defer gzipWriter.Close()

	flusher, ok := writer.(http.Flusher)
	if !ok {
		http.Error(writer, "streaming not supported", http.StatusInternalServerError)
		return
	}

	chunks := getChunkCount(request, 3)
	delay := getChunkDelay(request, 300*time.Millisecond)

	for index := 1; index <= chunks; index++ {
		line := fmt.Sprintf("data: {\"index\":%d,\"time\":\"%s\"}\n\n", index, time.Now().UTC().Format(time.RFC3339Nano))
		if _, err = gzipWriter.Write([]byte(line)); err != nil {
			log.Printf("write sse chunk failed: %v", err)
			return
		}
		if err = gzipWriter.Flush(); err != nil {
			log.Printf("flush gzip chunk failed: %v", err)
			return
		}
		flusher.Flush()

		if index < chunks {
			time.Sleep(delay)
		}
	}

	if _, err = gzipWriter.Write([]byte("data: [DONE]\n\n")); err != nil {
		log.Printf("write done chunk failed: %v", err)
		return
	}
	_ = gzipWriter.Flush()
	flusher.Flush()
}

func writeGzip(writer http.ResponseWriter, statusCode int, contentType string, body []byte) {
	gzipWriter, err := startGzip(writer, contentType)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	defer gzipWriter.Close()

	writer.WriteHeader(statusCode)
	if _, err = gzipWriter.Write(body); err != nil {
		log.Printf("write gzip body failed: %v", err)
	}
}

func startGzip(writer http.ResponseWriter, contentType string) (*gzip.Writer, error) {
	writer.Header().Set("Content-Type", contentType)
	writer.Header().Set("Content-Encoding", "gzip")
	writer.Header().Set("Vary", "Accept-Encoding")

	gzipWriter, err := gzip.NewWriterLevel(writer, gzip.BestSpeed)
	if err != nil {
		return nil, fmt.Errorf("create gzip writer failed: %w", err)
	}
	return gzipWriter, nil
}

func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		log.Printf("%s %s accept-encoding=%q", request.Method, request.URL.String(), request.Header.Get("Accept-Encoding"))
		next.ServeHTTP(writer, request)
	})
}

func getChunkCount(request *http.Request, fallback int) int {
	raw := strings.TrimSpace(request.URL.Query().Get("chunks"))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	if value > 100 {
		return 100
	}
	return value
}

func getChunkDelay(request *http.Request, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(request.URL.Query().Get("delayMs"))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return fallback
	}
	if value > 10_000 {
		value = 10_000
	}
	return time.Duration(value) * time.Millisecond
}

func getEnv(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

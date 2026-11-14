package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

func main() {
	host := "127.0.0.1"
	port := 8090
	for i := 1; i < len(os.Args)-1; i++ {
		switch os.Args[i] {
		case "--host":
			host = strings.TrimSpace(os.Args[i+1])
		case "--port":
			if parsed, err := strconv.Atoi(strings.TrimSpace(os.Args[i+1])); err == nil {
				port = parsed
			}
		}
	}
	// Record argv in the working directory so wrapper tests can assert the launch flags.
	if b, err := json.Marshal(os.Args[1:]); err == nil {
		_ = os.WriteFile("fake-llama-argv.json", b, 0o644)
	}
	alias := "fake"
	for i := 1; i < len(os.Args)-1; i++ {
		if os.Args[i] == "--alias" {
			alias = os.Args[i+1]
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
	})
	mux.HandleFunc("/v1/embeddings", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Input []string `json:"input"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		data := make([]map[string]any, len(req.Input))
		for i := range req.Input {
			data[i] = map[string]any{"index": i, "embedding": []float32{0.1, 0.2, 0.3}}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	})
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model":   alias,
			"choices": []map[string]any{{"index": 0, "finish_reason": "stop", "message": map[string]any{"role": "assistant", "content": "pong"}}},
		})
	})
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("llama_requests_total 1\n"))
	})

	server := &http.Server{Addr: host + ":" + strconv.Itoa(port), Handler: mux}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signals
		_ = server.Shutdown(context.Background())
	}()
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		os.Exit(1)
	}
}

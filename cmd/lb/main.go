package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/josevitorrodriguess/load-balancer-cli/internal/config/logger"
)

func main() {
	log := logger.New(logger.Config{
		Level:  getEnv("LOG_LEVEL", "debug"),
		Format: getEnv("LOG_FORMAT", "text"),
	})

	slog.SetDefault(log)

	slog.Info("starting load balancer", "port", "8080")

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("request received",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
		)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	if err := http.ListenAndServe(":8080", mux); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
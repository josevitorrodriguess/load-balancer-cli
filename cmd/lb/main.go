package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/josevitorrodriguess/load-balancer-cli/internal/balancer"
	"github.com/josevitorrodriguess/load-balancer-cli/internal/cli"
	appconfig "github.com/josevitorrodriguess/load-balancer-cli/internal/config"
	"github.com/josevitorrodriguess/load-balancer-cli/internal/config/logger"
	"github.com/josevitorrodriguess/load-balancer-cli/internal/health"
	"github.com/josevitorrodriguess/load-balancer-cli/internal/proxy"
)

func main() {
	log := logger.New(logger.Config{
		Level:  getEnv("LOG_LEVEL", "debug"),
		Format: getEnv("LOG_FORMAT", "text"),
	})

	slog.SetDefault(log)

	configPath := getEnv("CONFIG_FILE", appconfig.DefaultFile)

	if err := cli.Run(os.Args[1:], configPath, runServer); err != nil {
		slog.Error("cli failed", "error", err)
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

func runServer(startConfig cli.StartConfig) error {
	if startConfig.DemoMode || demoModeEnabled() {
		startDemoBackends()
	}

	cfg, err := appconfig.Load(startConfig.ConfigPath)
	if err != nil {
		return err
	}

	lb, err := appconfig.NewBalancer(cfg)
	if err != nil {
		return err
	}

	reloadable := balancer.NewReloadable(lb)

	mux := http.NewServeMux()

	if err := proxy.StartProxyWithConfig(mux, reloadable, appconfig.ProxyConfig(cfg)); err != nil {
		return err
	}

	health.New(
		reloadable,
		cfg.Timeouts.HealthInterval.Duration,
		cfg.Timeouts.HealthTimeout.Duration,
	).Start()

	reloader, err := appconfig.NewReloader(startConfig.ConfigPath, cfg, reloadable, func(next appconfig.Config) error {
		updated, err := appconfig.NewBalancerWithState(next, reloadable.Backends())
		if err != nil {
			return err
		}

		reloadable.Swap(updated)
		return nil
	})
	if err != nil {
		return err
	}

	reloader.Start()

	addr := ":" + cfg.Port
	slog.Info("load balancer started", "port", cfg.Port, "config_file", startConfig.ConfigPath)

	return http.ListenAndServe(addr, mux)
}

func demoModeEnabled() bool {
	return strings.EqualFold(getEnv("DEMO_MODE", "false"), "true")
}

func startDemoBackends() {
	startDemoBackend("9001", "backend-1")
	startDemoBackend("9002", "backend-2")
}

func startDemoBackend(port, name string) {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(fmt.Sprintf("%s handled %s %s\n", name, r.Method, r.URL.Path)))
	})

	go func() {
		addr := ":" + port
		slog.Info("demo backend started", "backend_url", "http://localhost"+addr)

		if err := http.ListenAndServe(addr, mux); err != nil {
			slog.Error("server failed", "backend_url", "http://localhost"+addr, "error", err)
			os.Exit(1)
		}
	}()
}

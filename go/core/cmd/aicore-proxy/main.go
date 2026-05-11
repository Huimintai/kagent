package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/kagent-dev/kagent/go/core/internal/aicoreproxy"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	port := 9090
	if p := os.Getenv("AICORE_PROXY_PORT"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			port = v
		}
	}

	cfg := aicoreproxy.Config{
		Port:          port,
		Provider:      os.Getenv("AICORE_PROXY_PROVIDER"),
		BaseURL:       os.Getenv("SAP_AI_CORE_BASE_URL"),
		AuthURL:       os.Getenv("SAP_AI_CORE_AUTH_URL"),
		ClientID:      os.Getenv("SAP_AI_CORE_CLIENT_ID"),
		ClientSecret:  os.Getenv("SAP_AI_CORE_CLIENT_SECRET"),
		ResourceGroup: os.Getenv("SAP_AI_CORE_RESOURCE_GROUP"),
		Model:         os.Getenv("SAP_AI_CORE_MODEL"),
	}

	proxy, err := aicoreproxy.New(cfg, log)
	if err != nil {
		log.Error("failed to create proxy", "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := proxy.ListenAndServe(ctx); err != nil {
		log.Error("proxy exited", "error", err)
		os.Exit(1)
	}
}

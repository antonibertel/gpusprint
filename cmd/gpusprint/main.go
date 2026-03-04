package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/antonibertel/gpusprint/internal/config"
	"github.com/antonibertel/gpusprint/internal/detect"
	"github.com/antonibertel/gpusprint/internal/export"
	"github.com/antonibertel/gpusprint/internal/hal"
	"github.com/antonibertel/gpusprint/internal/kube"
	"github.com/antonibertel/gpusprint/internal/kubelet"
	"github.com/antonibertel/gpusprint/internal/sampler"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", "err", err)
		os.Exit(1)
	}

	// Initialize structured logger
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}
	if cfg.LogFormat == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(handler))

	slog.Info("Starting gpusprint hardware agent", "node", cfg.NodeName, "cluster", cfg.ClusterName)

	var provider hal.AcceleratorProvider

	for {
		p, err := detect.Detect()
		if err == nil {
			provider = p
			break
		}

		slog.Warn("No Accelerator Provider available. Sleeping before retrying...", "interval", cfg.DiscoveryRetryInterval, "err", err)
		time.Sleep(cfg.DiscoveryRetryInterval)
	}

	defer provider.Close()

	if err := provider.Init(); err != nil {
		slog.Error("Failed to initialize accelerator provider", "err", err)
		os.Exit(1)
	}

	devices, err := provider.Devices()
	if err != nil {
		slog.Error("Failed to get accelerator devices", "err", err)
		os.Exit(1)
	}

	for _, dev := range devices {
		slog.Info("Discovered Accelerator", "uuid", dev.UUID, "vendor", dev.Vendor, "model", dev.Model)
	}

	var klProvider kubelet.Provider
	kl := kubelet.NewClient(cfg.KubeletSocket)
	if err := kl.Connect(ctx); err != nil {
		slog.Warn("Could not connect to kubelet socket, will proceed without pod mapping", "err", err)
	} else {
		defer kl.Close()
		klProvider = kl
	}

	var inf kube.PodProvider
	manager := kube.NewInformerManager(cfg.NodeName)
	if err := manager.Start(ctx); err != nil {
		slog.Warn("Could not start k8s informer, will proceed without metadata enrichment", "err", err)
	}
	inf = manager

	// ── HTTP server (health checks + optional /metrics) ─────────────────
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	var exporters []export.Exporter

	if cfg.PrometheusEnabled {
		promExp := export.NewPrometheusExporter()
		if err := promExp.Start(ctx); err != nil {
			slog.Error("Failed to start Prometheus exporter", "err", err)
		} else {
			mux.Handle("/metrics", promExp.Handler())
			exporters = append(exporters, promExp)
			defer promExp.Close()
		}
	}

	if cfg.PubSubEnabled {
		psExp := export.NewPubSubExporter(cfg)
		if err := psExp.Start(ctx); err != nil {
			slog.Error("Failed to start PubSub exporter", "err", err)
		} else {
			exporters = append(exporters, psExp)
			defer psExp.Close()
		}
	}

	if cfg.OTLPEnabled {
		otlpExp := export.NewOTLPExporter(cfg)
		if err := otlpExp.Start(ctx); err != nil {
			slog.Error("Failed to start OTLP exporter", "err", err)
		} else {
			exporters = append(exporters, otlpExp)
			defer otlpExp.Close()
		}
	}

	httpSrv := &http.Server{Addr: cfg.HTTPAddr, Handler: mux}
	go func() {
		slog.Info("Starting HTTP server", "addr", cfg.HTTPAddr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server failed", "err", err)
		}
	}()
	defer httpSrv.Close()

	smp := sampler.New(cfg, provider, klProvider, inf, exporters...)
	if err := smp.Run(ctx); err != nil {
		slog.Error("Sampler loop terminated with error", "err", err)
		os.Exit(1)
	}

	slog.Info("gpusprint shutdown complete")
}

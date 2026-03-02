package sampler

import (
	"context"
	"log/slog"
	"time"

	"github.com/antonibertel/gpusprint/internal/config"
	"github.com/antonibertel/gpusprint/internal/enrichment"
	"github.com/antonibertel/gpusprint/internal/export"
	"github.com/antonibertel/gpusprint/internal/hal"
	"github.com/antonibertel/gpusprint/internal/kube"
	"github.com/antonibertel/gpusprint/internal/kubelet"
)

type Loop struct {
	cfg       *config.Config
	provider  hal.AcceleratorProvider
	kubelet   kubelet.Provider
	informer  kube.PodProvider
	exporters []export.Exporter
}

func New(cfg *config.Config, provider hal.AcceleratorProvider, kl kubelet.Provider, inf kube.PodProvider, exporters ...export.Exporter) *Loop {
	return &Loop{
		cfg:       cfg,
		provider:  provider,
		kubelet:   kl,
		informer:  inf,
		exporters: exporters,
	}
}

func (l *Loop) Run(ctx context.Context) error {
	ticker := time.NewTicker(l.cfg.SampleInterval)
	defer ticker.Stop()

	slog.Info("Starting sampling loop", "interval", l.cfg.SampleInterval)

	// Perform initial sample immediately
	l.sample(ctx)

	for {
		select {
		case <-ctx.Done():
			slog.Info("Stopping sampling loop")
			return nil
		case <-ticker.C:
			l.sample(ctx)
		}
	}
}

func (l *Loop) sample(ctx context.Context) {
	metrics, err := l.provider.Metrics()
	if err != nil {
		slog.Error("Failed to collect accelerator metrics", "err", err)
		return
	}

	var podMap map[string][]kubelet.PodInfo
	if l.kubelet != nil {
		pm, err := l.kubelet.GetAcceleratorMapping(ctx)
		if err != nil {
			slog.Warn("Failed to fetch kubelet pod mapping", "err", err)
		} else {
			podMap = pm
		}
	}

	snapshot := enrichment.Enrich(metrics, podMap, l.informer, l.cfg.ClusterName, l.cfg.NodeName, l.cfg.TeamLabelKey, l.cfg.OwnerLabelKey)

	for _, exp := range l.exporters {
		if err := exp.Export(ctx, snapshot); err != nil {
			slog.Error("Failed to export metrics", "err", err)
		}
	}

	if l.cfg.DevelopmentMode {
		for _, hw := range snapshot.Hardware {
			slog.Info("Hardware Metric",
				"uuid", hw.UUID,
				"utilization_percent", hw.UtilizationPercent,
				"memory_used_bytes", hw.MemoryUsedBytes,
				"memory_total_bytes", hw.MemoryTotalBytes,
			)
		}
		for _, alloc := range snapshot.Allocations {
			slog.Info("Allocation Info",
				"uuid", alloc.UUID,
				"pod_namespace", alloc.PodNamespace,
				"pod_name", alloc.PodName,
				"container_name", alloc.ContainerName,
				"workload_kind", alloc.WorkloadKind,
				"workload_name", alloc.WorkloadName,
				"team", alloc.Team,
				"owner", alloc.Owner,
			)
		}
	}
}

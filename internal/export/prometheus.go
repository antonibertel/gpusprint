package export

import (
	"context"
	"errors"
	"net/http"
	"sync"

	"github.com/antonibertel/gpusprint/internal/config"
	"github.com/antonibertel/gpusprint/internal/enrichment"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type PrometheusExporter struct {
	addr string
	srv  *http.Server

	utilizationGauge *prometheus.GaugeVec
	memUsedGauge     *prometheus.GaugeVec
	memTotalGauge    *prometheus.GaugeVec
	allocationInfo   *prometheus.GaugeVec

	mu sync.Mutex
}

func NewPrometheusExporter(cfg *config.Config) *PrometheusExporter {
	hwLabels := []string{"cluster", "node", "uuid", "vendor", "model"}
	allocLabels := []string{"uuid", "pod_namespace", "pod_name", "container_name", "workload_kind", "workload_name", "team", "owner"}

	return &PrometheusExporter{
		addr: cfg.PrometheusAddr,
		utilizationGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gpusprint_utilization_percent",
				Help: "GPU compute utilization as a percentage. One row per physical GPU.",
			},
			hwLabels,
		),
		memUsedGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gpusprint_memory_used_bytes",
				Help: "GPU memory currently consumed in bytes. One row per physical GPU.",
			},
			hwLabels,
		),
		memTotalGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gpusprint_memory_total_bytes",
				Help: "GPU total memory capacity in bytes. One row per physical GPU.",
			},
			hwLabels,
		),
		allocationInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gpusprint_allocation_info",
				Help: "GPU-to-pod allocation mapping. Value is always 1. Join on uuid to correlate with hardware metrics.",
			},
			allocLabels,
		),
	}
}

func (pe *PrometheusExporter) Start(ctx context.Context) error {
	prometheus.MustRegister(pe.utilizationGauge)
	prometheus.MustRegister(pe.memUsedGauge)
	prometheus.MustRegister(pe.memTotalGauge)
	prometheus.MustRegister(pe.allocationInfo)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	pe.srv = &http.Server{
		Addr:    pe.addr,
		Handler: mux,
	}

	go func() {
		if err := pe.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			// This would ideally be logged via slog, but keeping simple
		}
	}()

	return nil
}

func (pe *PrometheusExporter) Export(ctx context.Context, snapshot enrichment.Snapshot) error {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	pe.utilizationGauge.Reset()
	pe.memUsedGauge.Reset()
	pe.memTotalGauge.Reset()
	pe.allocationInfo.Reset()

	// Hardware metrics: one row per physical GPU, no pod labels
	for _, hw := range snapshot.Hardware {
		lbls := prometheus.Labels{
			"cluster": snapshot.Cluster,
			"node":    snapshot.Node,
			"uuid":    hw.UUID,
			"vendor":  hw.Vendor,
			"model":   hw.Model,
		}

		pe.utilizationGauge.With(lbls).Set(hw.UtilizationPercent)
		pe.memUsedGauge.With(lbls).Set(float64(hw.MemoryUsedBytes))
		pe.memTotalGauge.With(lbls).Set(float64(hw.MemoryTotalBytes))
	}

	// Allocation info: one row per (GPU, pod, container) binding
	for _, alloc := range snapshot.Allocations {
		lbls := prometheus.Labels{
			"uuid":           alloc.UUID,
			"pod_namespace":  alloc.PodNamespace,
			"pod_name":       alloc.PodName,
			"container_name": alloc.ContainerName,
			"workload_kind":  alloc.WorkloadKind,
			"workload_name":  alloc.WorkloadName,
			"team":           alloc.Team,
			"owner":          alloc.Owner,
		}

		pe.allocationInfo.With(lbls).Set(1)
	}

	return nil
}

func (pe *PrometheusExporter) Close() error {
	if pe.srv != nil {
		return pe.srv.Close()
	}
	return nil
}

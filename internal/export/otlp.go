package export

import (
	"context"
	"fmt"
	"sync"

	"github.com/antonibertel/gpusprint/internal/config"
	"github.com/antonibertel/gpusprint/internal/enrichment"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

type OTLPExporter struct {
	cfg *config.Config

	provider *sdkmetric.MeterProvider
	meter    metric.Meter

	utilizationGauge metric.Float64ObservableGauge
	memUsedGauge     metric.Float64ObservableGauge
	memTotalGauge    metric.Float64ObservableGauge
	allocationGauge  metric.Float64ObservableGauge

	mu              sync.Mutex
	currentSnapshot enrichment.Snapshot
}

func NewOTLPExporter(cfg *config.Config) *OTLPExporter {
	return &OTLPExporter{
		cfg: cfg,
	}
}

func (oe *OTLPExporter) Start(ctx context.Context) error {
	var exporter sdkmetric.Exporter
	var err error

	if oe.cfg.OTLPProtocol == "http/protobuf" {
		opts := []otlpmetrichttp.Option{
			otlpmetrichttp.WithEndpoint(oe.cfg.OTLPEndpoint),
			otlpmetrichttp.WithInsecure(),
			otlpmetrichttp.WithTimeout(oe.cfg.OTLPExportTimeout),
		}
		exporter, err = otlpmetrichttp.New(ctx, opts...)
	} else {
		opts := []otlpmetricgrpc.Option{
			otlpmetricgrpc.WithEndpoint(oe.cfg.OTLPEndpoint),
			otlpmetricgrpc.WithInsecure(),
			otlpmetricgrpc.WithTimeout(oe.cfg.OTLPExportTimeout),
		}
		exporter, err = otlpmetricgrpc.New(ctx, opts...)
	}

	if err != nil {
		return fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	reader := sdkmetric.NewPeriodicReader(exporter,
		sdkmetric.WithInterval(oe.cfg.OTLPExportInterval),
		sdkmetric.WithTimeout(oe.cfg.OTLPExportTimeout),
	)
	oe.provider = sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	oe.meter = oe.provider.Meter("gpusprint")

	oe.utilizationGauge, err = oe.meter.Float64ObservableGauge(
		"gpusprint.utilization.percent",
		metric.WithDescription("GPU Utilization as a percentage."),
	)
	if err != nil {
		return err
	}

	oe.memUsedGauge, err = oe.meter.Float64ObservableGauge(
		"gpusprint.memory.used.bytes",
		metric.WithDescription("GPU Memory Used in bytes."),
	)
	if err != nil {
		return err
	}

	oe.memTotalGauge, err = oe.meter.Float64ObservableGauge(
		"gpusprint.memory.total.bytes",
		metric.WithDescription("GPU Memory Total in bytes."),
	)
	if err != nil {
		return err
	}

	oe.allocationGauge, err = oe.meter.Float64ObservableGauge(
		"gpusprint.allocation.info",
		metric.WithDescription("GPU-to-pod allocation mapping. Value is always 1."),
	)
	if err != nil {
		return err
	}

	_, err = oe.meter.RegisterCallback(oe.observeMetrics, oe.utilizationGauge, oe.memUsedGauge, oe.memTotalGauge, oe.allocationGauge)
	if err != nil {
		return err
	}

	return nil
}

func (oe *OTLPExporter) observeMetrics(ctx context.Context, observer metric.Observer) error {
	oe.mu.Lock()
	snap := oe.currentSnapshot
	oe.mu.Unlock()

	for _, hw := range snap.Hardware {
		attrs := metric.WithAttributes(
			attribute.String("cluster", snap.Cluster),
			attribute.String("node", snap.Node),
			attribute.String("uuid", hw.UUID),
			attribute.String("vendor", hw.Vendor),
			attribute.String("model", hw.Model),
		)

		observer.ObserveFloat64(oe.utilizationGauge, hw.UtilizationPercent, attrs)
		observer.ObserveFloat64(oe.memUsedGauge, float64(hw.MemoryUsedBytes), attrs)
		observer.ObserveFloat64(oe.memTotalGauge, float64(hw.MemoryTotalBytes), attrs)
	}

	for _, alloc := range snap.Allocations {
		attrs := metric.WithAttributes(
			attribute.String("uuid", alloc.UUID),
			attribute.String("pod_namespace", alloc.PodNamespace),
			attribute.String("pod_name", alloc.PodName),
			attribute.String("container_name", alloc.ContainerName),
			attribute.String("workload_kind", alloc.WorkloadKind),
			attribute.String("workload_name", alloc.WorkloadName),
			attribute.String("team", alloc.Team),
			attribute.String("owner", alloc.Owner),
		)

		observer.ObserveFloat64(oe.allocationGauge, 1, attrs)
	}

	return nil
}

func (oe *OTLPExporter) Export(ctx context.Context, snapshot enrichment.Snapshot) error {
	oe.mu.Lock()
	oe.currentSnapshot = snapshot
	oe.mu.Unlock()
	return nil
}

func (oe *OTLPExporter) Close() error {
	if oe.provider != nil {
		return oe.provider.Shutdown(context.Background())
	}
	return nil
}

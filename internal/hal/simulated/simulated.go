// Package simulated provides a fake AcceleratorProvider that produces
// realistic-looking GPU metrics without any hardware dependency.
//
// The provider is purely a hardware simulator — it knows nothing about pods
// or workloads.  Pod-to-device mapping comes from the real kubelet
// pod-resources API, and pod metadata from the real K8s informer.
//
// Device IDs use the pattern "GPU-sim-N" to match the fake device plugin.
//
// Configure via environment:
//
//	NUM_GPUS       — number of devices to report (default 8, must match device plugin)
//	DEVICE_PREFIX  — UUID prefix (default "GPU-sim-")
package simulated

import (
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"strconv"
	"time"

	"github.com/antonibertel/gpusprint/internal/hal"
)

// gpu describes a single simulated device's hardware model and utilization behavior.
type gpu struct {
	model      string
	memTotalGB uint64
	baseUtil   float64 // mean utilization %
	utilJitter float64 // sine swing ±
	baseMemPct float64 // mean memory fraction 0–1
	memJitter  float64 // sine swing ±
}

// catalog of GPU hardware. Devices rotate through this list by index.
var catalog = []gpu{
	{"NVIDIA H100 80GB HBM3", 80, 93, 5, 0.91, 0.04},
	{"NVIDIA H100 80GB HBM3", 80, 87, 7, 0.86, 0.05},
	{"NVIDIA A100 80GB SXM4", 80, 70, 20, 0.65, 0.15},
	{"NVIDIA A100 80GB PCIe", 80, 45, 25, 0.50, 0.12},
	{"NVIDIA L4 24GB", 24, 30, 18, 0.35, 0.10},
	{"NVIDIA L4 24GB", 24, 55, 25, 0.45, 0.15},
	{"NVIDIA T4 16GB", 16, 20, 12, 0.25, 0.08},
	{"NVIDIA A10G 24GB", 24, 38, 20, 0.40, 0.12},
	{"NVIDIA A10G 24GB", 24, 1, 1, 0.55, 0.02},        // zombie
	{"NVIDIA H100 80GB HBM3", 80, 0, 0, 0.0, 0.0},     // idle
	{"NVIDIA A100 80GB PCIe", 80, 60, 30, 0.55, 0.20}, // bursty
	{"NVIDIA T4 16GB", 16, 10, 8, 0.15, 0.05},         // light inference
}

type simulatedProvider struct {
	numGPUs int
	prefix  string
}

func New() hal.AcceleratorProvider {
	n := 8
	if v := os.Getenv("NUM_GPUS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			n = parsed
		}
	}
	prefix := "GPU-sim-"
	if v := os.Getenv("DEVICE_PREFIX"); v != "" {
		prefix = v
	}
	return &simulatedProvider{numGPUs: n, prefix: prefix}
}

func (s *simulatedProvider) Init() error { return nil }
func (s *simulatedProvider) Close()      {}

func (s *simulatedProvider) Devices() ([]hal.AcceleratorDevice, error) {
	out := make([]hal.AcceleratorDevice, s.numGPUs)
	for i := range s.numGPUs {
		g := catalog[i%len(catalog)]
		out[i] = hal.AcceleratorDevice{
			UUID:   fmt.Sprintf("%s%d", s.prefix, i),
			Vendor: "nvidia",
			Model:  g.model,
		}
	}
	return out, nil
}

func (s *simulatedProvider) Metrics() ([]hal.AcceleratorMetrics, error) {
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		nodeName = "localhost"
	}

	now := time.Now().Unix()
	slowPhase := float64(now%600) / 600.0
	fastNoise := rand.Float64()*2 - 1

	out := make([]hal.AcceleratorMetrics, s.numGPUs)
	for i := range s.numGPUs {
		g := catalog[i%len(catalog)]

		phase := (slowPhase + float64(i)*0.13) * 2 * math.Pi
		sine := math.Sin(phase)

		util := clamp(g.baseUtil+g.utilJitter*sine+g.utilJitter*0.15*fastNoise, 0, 100)
		memPct := clamp(g.baseMemPct+g.memJitter*sine+g.memJitter*0.10*fastNoise, 0, 1)
		memTotal := g.memTotalGB * 1024 * 1024 * 1024

		out[i] = hal.AcceleratorMetrics{
			UUID:               fmt.Sprintf("%s%d@%s", s.prefix, i, nodeName),
			Vendor:             "nvidia",
			Model:              g.model,
			UtilizationPercent: util,
			MemoryUsedBytes:    uint64(float64(memTotal) * memPct),
			MemoryTotalBytes:   memTotal,
		}
	}
	return out, nil
}

func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }

// Compile-time interface check.
var _ hal.AcceleratorProvider = (*simulatedProvider)(nil)

package simulated

import (
	"fmt"
	"math/rand/v2"
	"sync"

	"github.com/antonibertel/gpusprint/internal/hal"
)

// gpuProfile defines the baseline behavior of a simulated GPU.
type gpuProfile struct {
	uuid       string
	vendor     string
	model      string
	memTotalGB uint64
	baseUtil   float64 // baseline utilization %
	utilJitter float64 // +/- random jitter on utilization
	baseMemPct float64 // baseline memory usage as fraction of total
	memJitter  float64 // +/- random jitter on memory fraction
}

var profiles = []gpuProfile{
	// GPU-sim-0: Heavy training job (alice's first GPU) — pegged near max
	{"GPU-sim-0", "nvidia", "NVIDIA H100 80GB HBM3", 80, 95.0, 4.0, 0.96, 0.03},
	// GPU-sim-1: Training job (alice's second GPU) — lightly used (data loading / gradient sync)
	{"GPU-sim-1", "nvidia", "NVIDIA H100 80GB HBM3", 80, 8.0, 6.0, 0.08, 0.04},
	// GPU-sim-2: Shared notebook GPU (bob + charlie via time-slicing) — moderate interactive use
	{"GPU-sim-2", "nvidia", "NVIDIA A100 80GB PCIe", 80, 40.0, 20.0, 0.45, 0.15},
	// GPU-sim-3: Allocated but idle inference backend (zombie)
	{"GPU-sim-3", "nvidia", "NVIDIA A100 80GB PCIe", 80, 0.5, 0.5, 0.01, 0.01},
	// GPU-sim-4: Batch processing — bursty pattern, moderate baseline
	{"GPU-sim-4", "nvidia", "NVIDIA L4 24GB", 24, 35.0, 25.0, 0.55, 0.20},
	// GPU-sim-5: Unallocated, completely idle
	{"GPU-sim-5", "nvidia", "NVIDIA L4 24GB", 24, 0.0, 0.0, 0.0, 0.0},
	// GPU-sim-6: Unallocated, completely idle
	{"GPU-sim-6", "nvidia", "NVIDIA A10G 24GB", 24, 0.0, 0.0, 0.0, 0.0},
	// GPU-sim-7: Unallocated, completely idle
	{"GPU-sim-7", "nvidia", "NVIDIA A10G 24GB", 24, 0.0, 0.0, 0.0, 0.0},
}

type simulatedProvider struct {
	mu sync.Mutex
}

func New() hal.AcceleratorProvider {
	return &simulatedProvider{}
}

func (d *simulatedProvider) Init() error {
	return nil
}

func (d *simulatedProvider) Devices() ([]hal.AcceleratorDevice, error) {
	var devices []hal.AcceleratorDevice
	for _, p := range profiles {
		devices = append(devices, hal.AcceleratorDevice{
			UUID:   p.uuid,
			Vendor: p.vendor,
			Model:  p.model,
		})
	}
	return devices, nil
}

func (d *simulatedProvider) Metrics() ([]hal.AcceleratorMetrics, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	var metrics []hal.AcceleratorMetrics
	for _, p := range profiles {
		memTotal := p.memTotalGB * 1024 * 1024 * 1024

		util := jitter(p.baseUtil, p.utilJitter)
		memPct := jitter(p.baseMemPct, p.memJitter)
		memUsed := uint64(float64(memTotal) * memPct)

		metrics = append(metrics, hal.AcceleratorMetrics{
			UUID:               p.uuid,
			Vendor:             p.vendor,
			Model:              p.model,
			UtilizationPercent: util,
			MemoryUsedBytes:    memUsed,
			MemoryTotalBytes:   memTotal,
		})
	}
	return metrics, nil
}

func (d *simulatedProvider) Close() {
	// no-op
}

// jitter returns base ± random jitter, clamped to [0, 100] for percentages and [0, 1] for fractions.
func jitter(base, j float64) float64 {
	if j == 0 {
		return base
	}
	v := base + (rand.Float64()*2-1)*j
	return max(0, min(v, maxForBase(base)))
}

func maxForBase(base float64) float64 {
	// If base looks like a percentage (>1), clamp at 100; otherwise at 1.0
	if base > 1.0 {
		return 100.0
	}
	return 1.0
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// Compile-time interface check.
var _ hal.AcceleratorProvider = (*simulatedProvider)(nil)

func init() {
	// Assign realistic UUIDs so they look like real NVIDIA UUIDs
	for i := range profiles {
		profiles[i].uuid = fmt.Sprintf("GPU-sim-%d", i)
	}
}

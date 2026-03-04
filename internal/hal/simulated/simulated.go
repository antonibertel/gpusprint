// Package simulated provides a fake AcceleratorProvider that produces
// realistic-looking GPU metrics without any hardware dependency.
//
// The simulated cluster models common real-world patterns:
//
//   - Multi-GPU workloads: a single training job owned by "alice" spans GPUs 0–3,
//     pegging the compute GPUs (0, 1) near max while the gradient-sync GPU (2)
//     idles most of the time and the checkpoint GPU (3) bursts occasionally.
//
//   - Shared / time-sliced GPUs: GPU 4 is shared between "bob" and "charlie"
//     via MIG / time-slicing; utilization oscillates as they take turns.
//
//   - Partial utilization: GPU 5 runs a small inference service that rarely
//     saturates the card.
//
//   - Zombie / allocated-but-idle: GPU 6 is allocated to a job that finished
//     without releasing its reservation.
//
//   - Completely free: GPU 7 is unallocated and truly idle.
package simulated

import (
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"time"

	"github.com/antonibertel/gpusprint/internal/hal"
)

// scenario describes the simulated behavior of one GPU.
type scenario struct {
	uuid       string
	vendor     string
	model      string
	memTotalGB uint64
	// baseUtil is the mean utilization percentage.
	baseUtil float64
	// utilJitter is the peak swing around baseUtil driven by the sine wave.
	utilJitter float64
	// baseMemPct and memJitter work the same way for memory (0–1 fraction).
	baseMemPct float64
	memJitter  float64
}

// gpus is the fixed set of simulated devices.  The slice index doubles as the
// device ordinal, which keeps the per-device sine phase nicely separated.
var gpus = []scenario{
	// ── Alice's distributed training job (GPUs 0-3) ──────────────────────────
	// GPU 0: compute-heavy forward/backward pass → near-saturated
	{"GPU-sim-0", "nvidia", "NVIDIA H100 80GB HBM3", 80, 94.0, 5.0, 0.92, 0.04},
	// GPU 1: compute, slightly lower because it also handles all-reduce comms
	{"GPU-sim-1", "nvidia", "NVIDIA H100 80GB HBM3", 80, 85.0, 8.0, 0.88, 0.05},
	// GPU 2: gradient-sync / parameter server — mostly idle with bursts
	{"GPU-sim-2", "nvidia", "NVIDIA H100 80GB HBM3", 80, 12.0, 10.0, 0.80, 0.05},
	// GPU 3: checkpoint I/O — occasional burst, long idle stretches
	{"GPU-sim-3", "nvidia", "NVIDIA H100 80GB HBM3", 80, 5.0, 30.0, 0.78, 0.10},

	// ── Bob & Charlie share GPU 4 via time-slicing ────────────────────────────
	// Utilization swings between ~20 % and ~80 % as their turns alternate.
	{"GPU-sim-4", "nvidia", "NVIDIA A100 80GB PCIe", 80, 50.0, 30.0, 0.50, 0.20},

	// ── Dave's inference service (GPU 5) — lightly loaded ────────────────────
	// Serves sporadic requests; typically 20–40 % util.
	{"GPU-sim-5", "nvidia", "NVIDIA L4 24GB", 24, 28.0, 14.0, 0.30, 0.10},

	// ── Zombie GPU (6) — allocated, job exited, memory still resident ─────────
	{"GPU-sim-6", "nvidia", "NVIDIA A10G 24GB", 24, 0.5, 0.5, 0.60, 0.02},

	// ── Free GPU (7) — completely idle, no allocation ─────────────────────────
	{"GPU-sim-7", "nvidia", "NVIDIA A10G 24GB", 24, 0.0, 0.0, 0.0, 0.0},
}

// simulatedProvider implements hal.AcceleratorProvider.
type simulatedProvider struct{}

// New returns a new simulated provider.
func New() hal.AcceleratorProvider { return &simulatedProvider{} }

func (s *simulatedProvider) Init() error { return nil }
func (s *simulatedProvider) Close()      {}

func (s *simulatedProvider) Devices() ([]hal.AcceleratorDevice, error) {
	out := make([]hal.AcceleratorDevice, len(gpus))
	for i, g := range gpus {
		out[i] = hal.AcceleratorDevice{UUID: g.uuid, Vendor: g.vendor, Model: g.model}
	}
	return out, nil
}

func (s *simulatedProvider) Metrics() ([]hal.AcceleratorMetrics, error) {
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		nodeName = "localhost"
	}

	// A slow (~10 min) cycle shared across all GPUs on this node, plus a fast
	// noise term that changes every scrape.  Both are cheap to compute.
	slowPhase := float64(time.Now().Unix()%600) / 600.0 // 0→1 over 10 min
	fastNoise := rand.Float64()*2 - 1                   // -1→1, re-drawn each call

	out := make([]hal.AcceleratorMetrics, len(gpus))
	for i, g := range gpus {
		// Each device gets its own phase offset so neighbouring GPUs don't
		// pulse in lock-step — mimics asynchronous workload progress.
		phase := (slowPhase + float64(i)*0.13) * 2 * math.Pi
		sine := math.Sin(phase) // -1 to 1

		util := clamp(g.baseUtil+g.utilJitter*sine+g.utilJitter*0.15*fastNoise, 0, 100)
		memPct := clamp(g.baseMemPct+g.memJitter*sine+g.memJitter*0.10*fastNoise, 0, 1)

		memTotal := g.memTotalGB * 1024 * 1024 * 1024
		out[i] = hal.AcceleratorMetrics{
			// Qualify the UUID with the node name so metrics from different
			// nodes in the same cluster remain globally unique.
			UUID:               fmt.Sprintf("%s@%s", g.uuid, nodeName),
			Vendor:             g.vendor,
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

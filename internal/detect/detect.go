package detect

import (
	"fmt"
	"log/slog"

	"github.com/antonibertel/gpusprint/internal/config"
	"github.com/antonibertel/gpusprint/internal/hal"
	"github.com/antonibertel/gpusprint/internal/hal/nvidia"
	"github.com/antonibertel/gpusprint/internal/hal/simulated"
)

func Detect() (hal.AcceleratorProvider, error) {
	// 1. Try native NVIDIA driver
	nv := nvidia.New()
	if err := nv.Init(); err == nil {
		slog.Info("Successfully initialized NVIDIA NVML provider")
		return nv, nil
	} else {
		slog.Debug("NVIDIA provider init failed", "err", err)
	}

	// 2. Check if we are allowed to fall back to simulated via global process env
	if config.Global != nil && config.Global.DevelopmentMode {
		slog.Warn("Development mode enabled. No native accelerators found, falling back to simulated provider")
		return simulated.New(), nil
	}

	return nil, fmt.Errorf("no supported accelerator devices found natively on this node")
}

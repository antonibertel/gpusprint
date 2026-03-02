package nvidia

import (
	"fmt"
	"log/slog"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"github.com/antonibertel/gpusprint/internal/hal"
)

type Provider struct{}

func New() *Provider {
	return &Provider{}
}

func (p *Provider) Init() error {
	ret := nvml.Init()
	if ret != nvml.SUCCESS {
		return fmt.Errorf("failed to initialize NVML: %v", nvml.ErrorString(ret))
	}

	version, ret := nvml.SystemGetDriverVersion()
	if ret == nvml.SUCCESS {
		slog.Info("Successfully initialized NVIDIA NVML", "driver_version", version)
	}

	return nil
}

func (p *Provider) Devices() ([]hal.AcceleratorDevice, error) {
	count, ret := nvml.DeviceGetCount()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("failed to get device count: %v", nvml.ErrorString(ret))
	}

	var devices []hal.AcceleratorDevice
	for i := 0; i < count; i++ {
		device, ret := nvml.DeviceGetHandleByIndex(i)
		if ret != nvml.SUCCESS {
			slog.Warn("Failed to get handle for device", "index", i, "err", nvml.ErrorString(ret))
			continue
		}

		uuid, ret := device.GetUUID()
		if ret != nvml.SUCCESS {
			continue
		}

		name, ret := device.GetName()
		if ret != nvml.SUCCESS {
			name = "Unknown NVIDIA Accelerator"
		}

		devices = append(devices, hal.AcceleratorDevice{
			UUID:   uuid,
			Vendor: "nvidia",
			Model:  name,
		})
	}

	return devices, nil
}

func (p *Provider) Metrics() ([]hal.AcceleratorMetrics, error) {
	count, ret := nvml.DeviceGetCount()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("failed to get device count: %v", nvml.ErrorString(ret))
	}

	var metrics []hal.AcceleratorMetrics
	for i := 0; i < count; i++ {
		device, ret := nvml.DeviceGetHandleByIndex(i)
		if ret != nvml.SUCCESS {
			continue
		}

		uuid, ret := device.GetUUID()
		if ret != nvml.SUCCESS {
			continue
		}

		util, ret := device.GetUtilizationRates()
		utilPercent := float64(0)
		if ret == nvml.SUCCESS {
			utilPercent = float64(util.Gpu)
		}

		mem, ret := device.GetMemoryInfo()
		memUsed := uint64(0)
		memTotal := uint64(0)
		if ret == nvml.SUCCESS {
			memUsed = mem.Used
			memTotal = mem.Total
		}

		name, ret := device.GetName()
		if ret != nvml.SUCCESS {
			name = "Unknown NVIDIA Accelerator"
		}

		metrics = append(metrics, hal.AcceleratorMetrics{
			UUID:               uuid,
			Vendor:             "nvidia",
			Model:              name,
			UtilizationPercent: utilPercent,
			MemoryUsedBytes:    memUsed,
			MemoryTotalBytes:   memTotal,
		})
	}

	return metrics, nil
}

func (p *Provider) Close() {
	ret := nvml.Shutdown()
	if ret != nvml.SUCCESS {
		slog.Error("failed to shutdown NVML", "err", nvml.ErrorString(ret))
	}
}

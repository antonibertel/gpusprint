package hal

type AcceleratorDevice struct {
	UUID   string
	Vendor string // "nvidia", "amd", "intel", "simulated"
	Model  string
}

type AcceleratorMetrics struct {
	UUID               string
	Vendor             string
	Model              string
	UtilizationPercent float64
	MemoryUsedBytes    uint64
	MemoryTotalBytes   uint64
}

type AcceleratorProvider interface {
	// Init probes the driver; returns error if driver unavailable.
	Init() error
	// Devices returns all Accelerators visible on this node.
	Devices() ([]AcceleratorDevice, error)
	// Metrics returns current telemetry for all Accelerators.
	Metrics() ([]AcceleratorMetrics, error)
	// Close releases any handles.
	Close()
}

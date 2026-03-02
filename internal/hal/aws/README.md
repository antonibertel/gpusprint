# AWS Inferentia & Trainium (Neuron)

AWS custom silicon (Inferentia/Trainium) does not currently provide a unified, standardized Go library comparable to NVIDIA's `nvml`.

### Implementation Approach

To implement the `AcceleratorProvider` interface for AWS chips:

1. **Filesystem Polling**: Read telemetry directly from the raw Linux `sysfs` exposed by the AWS Neuron driver.
   - Example paths: `/sys/class/misc/neuron*` or `/sys/devices/virtual/misc/neuron*`
   - You will need to parse the device UUIDs and read utilization percentage and memory consumption out of these text files.

2. **Neuron-Monitor Sidecar**: Alternatively, AWS provides an agent called `neuron-monitor`.
   - Your `Init()` function would verify the existence of the `neuron-monitor` socket locally.
   - The `Metrics()` function would dial the Unix socket and parse the JSON stream to extract utilization metrics.

# Google Cloud TPUs

Google TPUs on GKE are typically managed as Linux character devices attached to the container (e.g., `/dev/accel0`, `/dev/accel1`). There is no widely used "Go TPU Library" for direct hardware interaction.

### Implementation Approach

To implement the `AcceleratorProvider` interface for Google TPUs:

1. **Device Identification**:
   - The `Devices()` function can inspect the `/dev/accel*` paths or the `/sys/class/accel/` directory to discover the number of attached TPUs.

2. **Metrics Collection**:
   - TPUs are heavily managed by the `libtpu` library. You can often query a local metadata API or parse the Linux filesystem (`/sys/class/accel/`) to extract the actual hardware utilization percentage and HBM (High Bandwidth Memory) usage.
   - In GKE, the device plugin itself sometimes exposes a local gRPC or HTTP socket for basic telemetry that the provider can dial into in the `Metrics()` loop.

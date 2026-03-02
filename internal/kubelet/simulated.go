package kubelet

import "context"

type SimulatedProvider struct{}

func NewSimulatedProvider() Provider {
	return &SimulatedProvider{}
}

func (d *SimulatedProvider) GetAcceleratorMapping(ctx context.Context) (map[string][]PodInfo, error) {
	return map[string][]PodInfo{
		// Pod reserving 2 GPUs (GPU-sim-0 and GPU-sim-1)
		"GPU-sim-0": {{Namespace: "ai-ns", Name: "training-job-alice", ContainerName: "main"}},
		"GPU-sim-1": {{Namespace: "ai-ns", Name: "training-job-alice", ContainerName: "main"}},

		// Some pods requesting fractions of GPU (GPU-sim-2)
		"GPU-sim-2": {
			{Namespace: "ds-ns", Name: "jupyter-bob", ContainerName: "notebook"},
			{Namespace: "ds-ns", Name: "jupyter-charlie", ContainerName: "notebook"},
		},

		// Reserved but idling (GPU-sim-3)
		"GPU-sim-3": {{Namespace: "platform-ns", Name: "idle-backend", ContainerName: "api"}},

		// Low usage burstable pod (GPU-sim-4)
		"GPU-sim-4": {{Namespace: "ds-ns", Name: "batch-processor-diana", ContainerName: "worker"}},

		// GPUs 5-7 are unallocated (not in the map)
	}, nil
}

func (d *SimulatedProvider) Close() {
	// no-op
}

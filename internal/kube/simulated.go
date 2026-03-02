package kube

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SimulatedPodProvider struct{}

func NewSimulatedPodProvider() PodProvider {
	return &SimulatedPodProvider{}
}

func (s *SimulatedPodProvider) GetPod(namespace, name string) (*corev1.Pod, error) {
	// Generate mock pods based on the namespace and name queried (which corresponds to our simulated Kubelet map)

	labels := make(map[string]string)
	var kind, workloadName string

	switch name {
	case "training-job-alice":
		// Pod reserving 2 GPUs
		labels["team"] = "ai-research"
		labels["owner"] = "alice"
		kind = "Job"
		workloadName = "alpha-fold-training"
	case "jupyter-bob":
		// Shared fractional GPU
		labels["team"] = "data-science"
		labels["owner"] = "bob"
		kind = "StatefulSet"
		workloadName = "jupyter-workspace-bob"
	case "jupyter-charlie":
		// Shared fractional GPU
		labels["team"] = "data-science"
		labels["owner"] = "charlie"
		kind = "StatefulSet"
		workloadName = "jupyter-workspace-charlie"
	case "idle-backend":
		// Reserved but idling
		labels["team"] = "platform"
		labels["owner"] = "alice" // Alice owns both the training job and this backend
		kind = "Deployment"
		workloadName = "gpu-inference-api"
	case "batch-processor-diana":
		// Low usage burstable
		labels["team"] = "data-science"
		labels["owner"] = "diana"
		kind = "Job"
		workloadName = "nightly-batch-diana"
	default:
		// Fallback
		labels["team"] = "unknown"
		labels["owner"] = "unknown"
	}

	mockPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
	}

	if kind != "" && workloadName != "" {
		mockPod.OwnerReferences = []metav1.OwnerReference{
			{
				Kind: kind,
				Name: workloadName,
			},
		}
	}

	return mockPod, nil
}

func (s *SimulatedPodProvider) HasSynced() bool {
	return true
}

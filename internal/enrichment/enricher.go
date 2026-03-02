package enrichment

import (
	"github.com/antonibertel/gpusprint/internal/hal"
	"github.com/antonibertel/gpusprint/internal/kube"
	"github.com/antonibertel/gpusprint/internal/kubelet"
)

// HardwareSnapshot — one per physical GPU, no pod labels.
type HardwareSnapshot struct {
	hal.AcceleratorMetrics
}

// AllocationInfo — one per (GPU, pod, container) binding.
type AllocationInfo struct {
	UUID          string
	PodNamespace  string
	PodName       string
	ContainerName string
	WorkloadKind  string
	WorkloadName  string
	Team          string
	Owner         string
}

// Snapshot is the combined output of one enrichment tick.
type Snapshot struct {
	Cluster     string
	Node        string
	Hardware    []HardwareSnapshot
	Allocations []AllocationInfo
}

func Enrich(hw []hal.AcceleratorMetrics, podMap map[string][]kubelet.PodInfo, k8s kube.PodProvider, cluster, node, teamKey, ownerKey string) Snapshot {
	snap := Snapshot{
		Cluster: cluster,
		Node:    node,
	}

	for _, metric := range hw {
		// Hardware layer: one row per physical GPU UUID, no pod info
		snap.Hardware = append(snap.Hardware, HardwareSnapshot{
			AcceleratorMetrics: metric,
		})

		// Allocation layer: one row per (GPU, pod, container) binding
		podSet, ok := podMap[metric.UUID]
		if !ok || len(podSet) == 0 {
			continue
		}

		for _, info := range podSet {
			alloc := AllocationInfo{
				UUID:          metric.UUID,
				PodNamespace:  info.Namespace,
				PodName:       info.Name,
				ContainerName: info.ContainerName,
			}

			if k8s != nil {
				pod, err := k8s.GetPod(info.Namespace, info.Name)
				if err == nil && pod != nil {
					kind, name := kube.ExtractWorkloadMeta(pod)
					alloc.WorkloadKind = kind
					alloc.WorkloadName = name

					labels := kube.ExtractLabels(pod, teamKey, ownerKey)
					alloc.Team = labels[teamKey]
					alloc.Owner = labels[ownerKey]
				}
			}

			snap.Allocations = append(snap.Allocations, alloc)
		}
	}

	return snap
}

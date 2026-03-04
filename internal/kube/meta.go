package kube

import (
	corev1 "k8s.io/api/core/v1"
)

// standard Kubernetes recommended label, set by Helm/Kustomize automatically.
const workloadNameLabel = "app.kubernetes.io/name"

// ExtractWorkloadMeta returns the workload kind and name for a pod.
// Prefers the app.kubernetes.io/name label (stable across rollouts) over the
// direct OwnerReference, which for Deployments resolves to the ephemeral ReplicaSet.
func ExtractWorkloadMeta(pod *corev1.Pod) (kind, name string) {
	if labelName, ok := pod.Labels[workloadNameLabel]; ok && labelName != "" {
		kind = "Deployment"
		if len(pod.OwnerReferences) > 0 && pod.OwnerReferences[0].Kind != "ReplicaSet" {
			kind = pod.OwnerReferences[0].Kind
		}
		return kind, labelName
	}
	if len(pod.OwnerReferences) > 0 {
		return pod.OwnerReferences[0].Kind, pod.OwnerReferences[0].Name
	}
	return "Pod", pod.Name
}

func ExtractLabels(pod *corev1.Pod, keys ...string) map[string]string {
	res := make(map[string]string)
	for _, k := range keys {
		if val, ok := pod.Labels[k]; ok {
			res[k] = val
		}
	}
	return res
}

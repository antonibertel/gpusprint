package kube

import (
	corev1 "k8s.io/api/core/v1"
)

func ExtractWorkloadMeta(pod *corev1.Pod) (kind, name string) {
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

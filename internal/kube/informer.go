package kube

import (
	"context"
	"fmt"
	"log/slog"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

type PodProvider interface {
	GetPod(namespace, name string) (*corev1.Pod, error)
	HasSynced() bool
}

type InformerManager struct {
	nodeName  string
	podLister corev1listers.PodLister
}

func NewInformerManager(nodeName string) *InformerManager {
	return &InformerManager{
		nodeName: nodeName,
	}
}

// transformPod strips a Pod object down to only the fields we read:
// ObjectMeta.Name, Namespace, Labels, and OwnerReferences.
// This dramatically reduces the per-pod memory footprint in the informer cache.
func transformPod(obj any) (any, error) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return obj, nil
	}

	stripped := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:            pod.Name,
			Namespace:       pod.Namespace,
			Labels:          pod.Labels,
			OwnerReferences: pod.OwnerReferences,
		},
	}
	return stripped, nil
}

func (m *InformerManager) Start(ctx context.Context) error {
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fallback for local testing outside cluster
		kubeconfig := clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename()
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return fmt.Errorf("failed to build kube config: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// Only watch pods on this node to minimize memory footprint and API load
	tweakListOptions := func(options *metav1.ListOptions) {
		if m.nodeName != "" && m.nodeName != "localhost" {
			options.FieldSelector = fields.OneTermEqualSelector("spec.nodeName", m.nodeName).String()
		}
	}

	factory := informers.NewSharedInformerFactoryWithOptions(clientset, 0,
		informers.WithTweakListOptions(tweakListOptions),
	)

	podInformer := factory.Core().V1().Pods()

	// Strip cached pods to metadata-only to bound memory usage
	if err := podInformer.Informer().SetTransform(transformPod); err != nil {
		return fmt.Errorf("failed to set pod transform: %w", err)
	}

	slog.Info("Starting K8s pod informer", "nodeName", m.nodeName)
	factory.Start(ctx.Done())

	// Block until the initial cache sync completes so the lister is ready
	for gvr, ok := range factory.WaitForCacheSync(ctx.Done()) {
		if !ok {
			return fmt.Errorf("failed to sync informer cache for %v", gvr)
		}
	}

	m.podLister = podInformer.Lister()
	slog.Info("K8s pod informer synced")

	// Register a shutdown log. The factory stops when ctx is cancelled.
	go func() {
		<-ctx.Done()
		slog.Info("K8s pod informer stopped")
	}()

	return nil
}

// GetPod retrieves a pod from the informer's in-memory cache.
func (m *InformerManager) GetPod(namespace, name string) (*corev1.Pod, error) {
	return m.podLister.Pods(namespace).Get(name)
}

// HasSynced reports whether the informer cache has completed its initial sync.
func (m *InformerManager) HasSynced() bool {
	return m.podLister != nil
}

// compile-time check: ensure SetTransform signature is compatible
var _ cache.TransformFunc = transformPod

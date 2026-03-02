package kubelet

import (
	"context"
	"fmt"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	podresourcesv1 "k8s.io/kubelet/pkg/apis/podresources/v1"
)

type Provider interface {
	GetAcceleratorMapping(ctx context.Context) (map[string][]PodInfo, error)
	Close()
}

type Client struct {
	socketPath string
	conn       *grpc.ClientConn
	client     podresourcesv1.PodResourcesListerClient
}

type PodInfo struct {
	Namespace     string
	Name          string
	ContainerName string
}

func NewClient(socketPath string) *Client {
	return &Client{
		socketPath: socketPath,
	}
}

func (c *Client) Connect(ctx context.Context) error {
	addr := fmt.Sprintf("unix://%s", c.socketPath)
	slog.Info("Dialing kubelet podresources api", "addr", addr)

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to dial kubelet socket: %w", err)
	}

	c.conn = conn
	c.client = podresourcesv1.NewPodResourcesListerClient(conn)
	return nil
}

func (c *Client) GetAcceleratorMapping(ctx context.Context) (map[string][]PodInfo, error) {
	if c.client == nil {
		return nil, fmt.Errorf("kubelet client not connected")
	}

	resp, err := c.client.List(ctx, &podresourcesv1.ListPodResourcesRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pod resources: %w", err)
	}

	mapping := make(map[string][]PodInfo)

	for _, pod := range resp.PodResources {
		for _, container := range pod.Containers {
			for _, dev := range container.Devices {
				// We care about nvidia.com/gpu or potentially others in the future
				// For now, let's map any device ID we find.
				for _, devID := range dev.DeviceIds {
					mapping[devID] = append(mapping[devID], PodInfo{
						Namespace:     pod.Namespace,
						Name:          pod.Name,
						ContainerName: container.Name,
					})
				}
			}
		}
	}

	return mapping, nil
}

func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

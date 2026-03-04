// Fake GPU device plugin for local Kubernetes testing.
//
// Registers fake nvidia.com/gpu devices with the kubelet so that:
//   - Pods requesting nvidia.com/gpu get scheduled on nodes without real GPUs
//   - The kubelet pod-resources API returns real device→pod mappings
//   - gpusprint's HAL simulator can match its metrics to real pod allocations
//
// Usage:
//
//	Deploy as a DaemonSet that mounts /var/lib/kubelet/device-plugins.
//	Set NUM_GPUS (default 8) and DEVICE_PREFIX (default "GPU-sim-") via env.
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

func main() {
	numGPUs := 8
	if v := os.Getenv("NUM_GPUS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			numGPUs = n
		}
	}
	prefix := "GPU-sim-"
	if v := os.Getenv("DEVICE_PREFIX"); v != "" {
		prefix = v
	}

	devices := make([]*pluginapi.Device, numGPUs)
	for i := range numGPUs {
		devices[i] = &pluginapi.Device{
			ID:     fmt.Sprintf("%s%d", prefix, i),
			Health: pluginapi.Healthy,
		}
	}

	plugin := &fakePlugin{devices: devices}

	socketPath := filepath.Join(pluginapi.DevicePluginPath, "fake-nvidia-gpu.sock")
	os.Remove(socketPath)

	lis, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	srv := grpc.NewServer()
	pluginapi.RegisterDevicePluginServer(srv, plugin)

	go func() {
		if err := srv.Serve(lis); err != nil {
			log.Fatalf("serve: %v", err)
		}
	}()

	// Register with the kubelet.
	conn, err := grpc.NewClient(
		"unix://"+pluginapi.KubeletSocket,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("dial kubelet: %v", err)
	}

	client := pluginapi.NewRegistrationClient(conn)
	if _, err := client.Register(context.Background(), &pluginapi.RegisterRequest{
		Version:      pluginapi.Version,
		Endpoint:     "fake-nvidia-gpu.sock",
		ResourceName: "nvidia.com/gpu",
	}); err != nil {
		log.Fatalf("register: %v", err)
	}
	conn.Close()

	log.Printf("registered %d fake GPUs (nvidia.com/gpu) with kubelet", numGPUs)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	log.Println("shutting down")
	srv.GracefulStop()
}

// fakePlugin implements pluginapi.DevicePluginServer.
type fakePlugin struct {
	pluginapi.UnimplementedDevicePluginServer
	devices []*pluginapi.Device
}

func (f *fakePlugin) GetDevicePluginOptions(context.Context, *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return &pluginapi.DevicePluginOptions{}, nil
}

func (f *fakePlugin) ListAndWatch(_ *pluginapi.Empty, stream grpc.ServerStreamingServer[pluginapi.ListAndWatchResponse]) error {
	if err := stream.Send(&pluginapi.ListAndWatchResponse{Devices: f.devices}); err != nil {
		return err
	}
	// Block until the stream is closed.
	<-stream.Context().Done()
	return nil
}

func (f *fakePlugin) Allocate(_ context.Context, req *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	resp := &pluginapi.AllocateResponse{}
	for range req.ContainerRequests {
		resp.ContainerResponses = append(resp.ContainerResponses, &pluginapi.ContainerAllocateResponse{})
	}
	return resp, nil
}

func (f *fakePlugin) GetPreferredAllocation(context.Context, *pluginapi.PreferredAllocationRequest) (*pluginapi.PreferredAllocationResponse, error) {
	return &pluginapi.PreferredAllocationResponse{}, nil
}

func (f *fakePlugin) PreStartContainer(context.Context, *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return &pluginapi.PreStartContainerResponse{}, nil
}

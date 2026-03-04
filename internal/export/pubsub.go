package export

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"log/slog"

	"cloud.google.com/go/pubsub"
	"github.com/antonibertel/gpusprint/internal/config"
	"github.com/antonibertel/gpusprint/internal/enrichment"
)

type PubSubExporter struct {
	cfg          *config.Config
	projectID    string
	hwTopicID    string
	allocTopicID string
	client       *pubsub.Client
	hwTopic      *pubsub.Topic
	allocTopic   *pubsub.Topic
}

func NewPubSubExporter(cfg *config.Config) *PubSubExporter {
	return &PubSubExporter{
		cfg:          cfg,
		projectID:    cfg.PubSubProject,
		hwTopicID:    cfg.PubSubHardwareTopic,
		allocTopicID: cfg.PubSubAllocationTopic,
	}
}

func (ps *PubSubExporter) Start(ctx context.Context) error {
	if ps.projectID == "" || ps.hwTopicID == "" || ps.allocTopicID == "" {
		return fmt.Errorf("pubsub project and both topics must be provided")
	}

	client, err := pubsub.NewClient(ctx, ps.projectID)
	if err != nil {
		return fmt.Errorf("failed to create pubsub client: %w", err)
	}

	ps.client = client
	ps.hwTopic = client.Topic(ps.hwTopicID)
	ps.allocTopic = client.Topic(ps.allocTopicID)

	publishSettings := pubsub.PublishSettings{
		DelayThreshold:    ps.cfg.PubSubPublishDelay,
		CountThreshold:    ps.cfg.PubSubCountThreshold,
		ByteThreshold:     ps.cfg.PubSubByteThreshold,
		BufferedByteLimit: ps.cfg.PubSubBufferedByteLimit,
		FlowControlSettings: pubsub.FlowControlSettings{
			MaxOutstandingMessages: ps.cfg.PubSubCountThreshold,
			MaxOutstandingBytes:    ps.cfg.PubSubBufferedByteLimit,
			LimitExceededBehavior:  pubsub.FlowControlSignalError,
		},
	}
	ps.hwTopic.PublishSettings = publishSettings
	ps.allocTopic.PublishSettings = publishSettings

	return nil
}

type pubsubHardwareMessage struct {
	Timestamp          time.Time `json:"timestamp"`
	Cluster            string    `json:"cluster"`
	Node               string    `json:"node"`
	UUID               string    `json:"uuid"`
	Vendor             string    `json:"vendor"`
	Model              string    `json:"model"`
	UtilizationPercent float64   `json:"utilization_percent"`
	MemoryUsedBytes    uint64    `json:"memory_used_bytes"`
	MemoryTotalBytes   uint64    `json:"memory_total_bytes"`
}

type pubsubAllocationMessage struct {
	Timestamp     time.Time `json:"timestamp"`
	UUID          string    `json:"uuid"`
	PodNamespace  string    `json:"pod_namespace"`
	PodName       string    `json:"pod_name"`
	ContainerName string    `json:"container_name"`
	WorkloadKind  string    `json:"workload_kind"`
	WorkloadName  string    `json:"workload_name"`
	Team          string    `json:"team"`
	Owner         string    `json:"owner"`
}

func (ps *PubSubExporter) Export(ctx context.Context, snapshot enrichment.Snapshot) error {
	if ps.hwTopic == nil || ps.allocTopic == nil {
		return fmt.Errorf("pubsub exporter not started")
	}

	now := time.Now()

	for _, hw := range snapshot.Hardware {
		msg := pubsubHardwareMessage{
			Timestamp:          now,
			Cluster:            snapshot.Cluster,
			Node:               snapshot.Node,
			UUID:               hw.UUID,
			Vendor:             hw.Vendor,
			Model:              hw.Model,
			UtilizationPercent: hw.UtilizationPercent,
			MemoryUsedBytes:    hw.MemoryUsedBytes,
			MemoryTotalBytes:   hw.MemoryTotalBytes,
		}
		data, err := json.Marshal(msg)
		if err != nil {
			slog.Error("Failed to marshal hardware message", "uuid", hw.UUID, "err", err)
			continue
		}
		res := ps.hwTopic.Publish(ctx, &pubsub.Message{Data: data})
		go func(r *pubsub.PublishResult) {
			if _, err := r.Get(ctx); err != nil {
				slog.Error("Failed to publish hardware to pubsub", "err", err)
			}
		}(res)
	}

	for _, alloc := range snapshot.Allocations {
		msg := pubsubAllocationMessage{
			Timestamp:     now,
			UUID:          alloc.UUID,
			PodNamespace:  alloc.PodNamespace,
			PodName:       alloc.PodName,
			ContainerName: alloc.ContainerName,
			WorkloadKind:  alloc.WorkloadKind,
			WorkloadName:  alloc.WorkloadName,
			Team:          alloc.Team,
			Owner:         alloc.Owner,
		}
		data, err := json.Marshal(msg)
		if err != nil {
			slog.Error("Failed to marshal allocation message", "uuid", alloc.UUID, "err", err)
			continue
		}
		res := ps.allocTopic.Publish(ctx, &pubsub.Message{Data: data})
		go func(r *pubsub.PublishResult) {
			if _, err := r.Get(ctx); err != nil {
				slog.Error("Failed to publish allocation to pubsub", "err", err)
			}
		}(res)
	}

	return nil
}

func (ps *PubSubExporter) Close() error {
	// Flush pending messages before stopping topics
	if ps.hwTopic != nil {
		ps.hwTopic.Flush()
		ps.hwTopic.Stop()
	}
	if ps.allocTopic != nil {
		ps.allocTopic.Flush()
		ps.allocTopic.Stop()
	}
	if ps.client != nil {
		return ps.client.Close()
	}
	return nil
}

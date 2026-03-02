package export

import (
	"context"
	"encoding/json"
	"fmt"

	"log/slog"

	"cloud.google.com/go/pubsub"
	"github.com/antonibertel/gpusprint/internal/config"
	"github.com/antonibertel/gpusprint/internal/enrichment"
)

type PubSubExporter struct {
	cfg       *config.Config
	projectID string
	topicID   string
	client    *pubsub.Client
	topic     *pubsub.Topic
}

func NewPubSubExporter(cfg *config.Config) *PubSubExporter {
	return &PubSubExporter{
		cfg:       cfg,
		projectID: cfg.PubSubProject,
		topicID:   cfg.PubSubTopic,
	}
}

func (ps *PubSubExporter) Start(ctx context.Context) error {
	if ps.projectID == "" || ps.topicID == "" {
		return fmt.Errorf("pubsub project and topic must be provided")
	}

	client, err := pubsub.NewClient(ctx, ps.projectID)
	if err != nil {
		return fmt.Errorf("failed to create pubsub client: %w", err)
	}

	ps.client = client
	ps.topic = client.Topic(ps.topicID)

	ps.topic.PublishSettings = pubsub.PublishSettings{
		DelayThreshold:    ps.cfg.PubSubPublishDelay,
		CountThreshold:    ps.cfg.PubSubCountThreshold,
		ByteThreshold:     ps.cfg.PubSubByteThreshold,
		BufferedByteLimit: ps.cfg.PubSubBufferedByteLimit,
	}

	return nil
}

func (ps *PubSubExporter) Export(ctx context.Context, snapshot enrichment.Snapshot) error {
	if ps.topic == nil {
		return fmt.Errorf("pubsub exporter not started")
	}

	data, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	res := ps.topic.Publish(ctx, &pubsub.Message{
		Data: data,
	})

	// We do not wait for res.Get(ctx) here to avoid blocking the sampler loop.
	// PubSub will batch according to the PublishSettings above in the background.
	go func() {
		// Use a detached context so publishing finishes even if sampler ctx cancels
		_, err := res.Get(context.Background())
		if err != nil {
			slog.Error("Failed to publish to pubsub background worker", "err", err)
		}
	}()

	return nil
}

func (ps *PubSubExporter) Close() error {
	var err error
	if ps.topic != nil {
		ps.topic.Stop()
	}
	if ps.client != nil {
		err = ps.client.Close()
	}
	return err
}

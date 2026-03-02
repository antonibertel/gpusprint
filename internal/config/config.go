package config

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	NodeName               string        `envconfig:"NODE_NAME" default:"localhost"`
	ClusterName            string        `envconfig:"CLUSTER_NAME" default:"local-cluster"`
	SampleInterval         time.Duration `envconfig:"SAMPLE_INTERVAL" default:"10s"`
	KubeletSocket          string        `envconfig:"KUBELET_SOCKET" default:"/var/lib/kubelet/pod-resources/kubelet.sock"`
	TeamLabelKey           string        `envconfig:"TEAM_LABEL_KEY" default:"team"`
	OwnerLabelKey          string        `envconfig:"OWNER_LABEL_KEY" default:"owner"`
	DevelopmentMode        bool          `envconfig:"DEVELOPMENT_MODE" default:"false"`
	DiscoveryRetryInterval time.Duration `envconfig:"DISCOVERY_RETRY_INTERVAL" default:"30s"`

	// Prometheus pull endpoint
	PrometheusEnabled bool   `envconfig:"PROMETHEUS_ENABLED" default:"true"`
	PrometheusAddr    string `envconfig:"PROMETHEUS_ADDR" default:":9400"`

	// Pub/Sub push
	PubSubEnabled           bool          `envconfig:"PUBSUB_ENABLED" default:"false"`
	PubSubProject           string        `envconfig:"PUBSUB_PROJECT"`
	PubSubTopic             string        `envconfig:"PUBSUB_TOPIC"`
	PubSubPublishDelay      time.Duration `envconfig:"PUBSUB_PUBLISH_DELAY" default:"1s"`
	PubSubCountThreshold    int           `envconfig:"PUBSUB_COUNT_THRESHOLD" default:"100"`
	PubSubByteThreshold     int           `envconfig:"PUBSUB_BYTE_THRESHOLD" default:"1000000"`       // 1MB
	PubSubBufferedByteLimit int           `envconfig:"PUBSUB_BUFFERED_BYTE_LIMIT" default:"10000000"` // 10MB

	// OTLP push
	OTLPEnabled  bool   `envconfig:"OTLP_ENABLED" default:"false"`
	OTLPEndpoint string `envconfig:"OTLP_ENDPOINT"`
	OTLPProtocol string `envconfig:"OTLP_PROTOCOL" default:"grpc"` // grpc or http/protobuf

	LogLevel  slog.Level `envconfig:"LOG_LEVEL" default:"info"`
	LogFormat string     `envconfig:"LOG_FORMAT" default:"json"`
}

var Global *Config

func Load() (*Config, error) {
	var c Config
	err := envconfig.Process("", &c)
	if err != nil {
		return nil, fmt.Errorf("failed to process config: %w", err)
	}
	Global = &c
	return &c, nil
}

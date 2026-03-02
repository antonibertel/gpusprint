package export

import (
	"context"

	"github.com/antonibertel/gpusprint/internal/enrichment"
)

type Exporter interface {
	// Start performs any setup (start HTTP server, open gRPC conn, etc.)
	Start(ctx context.Context) error
	// Export pushes/updates a batch of hardware metrics and allocation info
	Export(ctx context.Context, snapshot enrichment.Snapshot) error
	// Close tears down resources
	Close() error
}

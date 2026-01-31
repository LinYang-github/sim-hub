package search

import (
	"log/slog"
	"time"

	"sim-hub/internal/data"
)

// SearchClient wraps NATS interactions for search
type SearchClient struct {
	nats *data.NATSClient
}

func NewSearchClient(nats *data.NATSClient) *SearchClient {
	return &SearchClient{nats: nats}
}

// Search executes a search query via NATS RPC
func (s *SearchClient) Search(query string) ([]SearchResult, error) {
	if s.nats == nil || !s.nats.Config.Enabled {
		return nil, data.ErrNATSDisabled
	}

	var result []SearchResult

	// Request with short timeout (e.g., 200ms) for fast downgrade
	err := s.nats.Encoded.Request("simhub.search.query", query, &result, 200*time.Millisecond)
	if err != nil {
		slog.Warn("Search RPC failed or timed out", "error", err)
		return nil, err
	}

	return result, nil
}

package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"sim-hub/internal/data"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
)

// EventPayload matches core.LifecycleEvent structure
type EventPayload struct {
	Type       string         `json:"type"`
	ResourceID string         `json:"resource_id"`
	VersionID  string         `json:"version_id,omitempty"`
	TypeKey    string         `json:"type_key"`
	Timestamp  time.Time      `json:"timestamp"`
	Data       map[string]any `json:"data,omitempty"`
}

// Mimic core constants
const (
	EventResourceCreated = "resource.created"
	EventResourceDeleted = "resource.deleted"
	EventResourceUpdated = "resource.updated"
)

type ESWorker struct {
	es         *elasticsearch.Client
	nats       *data.NATSClient
	apiBaseURL string
	indexName  string
}

func NewESWorker(es *elasticsearch.Client, nats *data.NATSClient, apiBaseURL, indexName string) *ESWorker {
	return &ESWorker{
		es:         es,
		nats:       nats,
		apiBaseURL: apiBaseURL,
		indexName:  indexName,
	}
}

func (w *ESWorker) Start() {
	// 1. Ensure Index Exists
	w.ensureIndex()

	// 2. Subscribe to Resource Events (Async Sync)
	// simhub.events.resource
	w.nats.Encoded.Subscribe("simhub.events.resource", w.handleResourceEvent)

	// 3. Subscribe to Search Requests (RPC)
	// simhub.search.query
	w.nats.Encoded.Subscribe("simhub.search.query", w.handleSearchRequest)

	slog.Info("ES Search Worker started", "index", w.indexName)
}

func (w *ESWorker) ensureIndex() {
	res, err := w.es.Indices.Exists([]string{w.indexName})
	if err != nil {
		slog.Error("Failed to check index existence", "error", err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		// Create index with basic settings
		// We can add custom analyzers later (e.g. smartcn)
		settings := `{
			"settings": {
				"number_of_shards": 1,
				"number_of_replicas": 0
			},
			"mappings": {
				"properties": {
					"id": { "type": "keyword" },
					"name": { "type": "text", "analyzer": "standard" },
					"tags": { "type": "keyword" },
					"type_name": { "type": "keyword" },
					"created_at": { "type": "date" }
				}
			}
		}`
		res, err := w.es.Indices.Create(w.indexName, w.es.Indices.Create.WithBody(strings.NewReader(settings)))
		if err != nil {
			slog.Error("Failed to create index", "error", err)
			return
		}
		defer res.Body.Close()
		slog.Info("Created Elasticsearch index", "index", w.indexName)
	}
}

func (w *ESWorker) handleResourceEvent(event *EventPayload) {
	slog.Info("Received resource event for search sync", "type", event.Type, "id", event.ResourceID)

	ctx := context.Background()

	switch event.Type {
	case EventResourceDeleted:
		w.deleteFromIndex(ctx, event.ResourceID)
	case EventResourceCreated, EventResourceUpdated:
		w.syncToIndex(ctx, event.ResourceID)
	}
}

func (w *ESWorker) deleteFromIndex(ctx context.Context, id string) {
	req := esapi.DeleteRequest{
		Index:      w.indexName,
		DocumentID: id,
	}
	res, err := req.Do(ctx, w.es)
	if err != nil {
		slog.Error("ES delete failed", "id", id, "error", err)
		return
	}
	defer res.Body.Close()
}

func (w *ESWorker) syncToIndex(ctx context.Context, id string) {
	// Fetch full resource from API
	// TODO: Auth? Usually worker is trusted or needs an internal token.
	// For now assuming internal API is accessible directly or we skip auth.
	// Actually the API might require auth.
	// But let's assume the sidecar pattern where worker is in trusted network / localhost.
	// Or we can add a system token. For simplicty V1, try direct access.
	// NOTE: The `ListResources` in API might be public or protected. `GetResource` is usually protected?
	// `GetResource` allows PUBLIC access.

	url := fmt.Sprintf("%s/api/v1/resources/%s", w.apiBaseURL, id)
	resp, err := http.Get(url)
	if err != nil {
		slog.Error("Failed to fetch resource from API", "url", url, "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		slog.Warn("API returned non-200 for resource sync", "status", resp.StatusCode)
		return
	}

	body, _ := io.ReadAll(resp.Body)

	// Index document
	req := esapi.IndexRequest{
		Index:      w.indexName,
		DocumentID: id,
		Body:       bytes.NewReader(body),
		Refresh:    "true",
	}

	res, err := req.Do(ctx, w.es)
	if err != nil {
		slog.Error("ES index failed", "id", id, "error", err)
		return
	}
	defer res.Body.Close()

	if res.IsError() {
		slog.Error("ES index error response", "status", res.Status())
	} else {
		slog.Info("Synced resource to ES", "id", id)
	}
}

func (w *ESWorker) handleSearchRequest(subject, reply string, query string) {
	// NATS RPC handler
	// Query ES
	slog.Info("Handling search request", "query", query)

	var buf bytes.Buffer
	queryBody := map[string]interface{}{
		"query": map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":     query,
				"fields":    []string{"name^3", "tags", "type_name"},
				"fuzziness": "AUTO",
			},
		},
		"_source": []string{"id"}, // Only need IDs
		"size":    50,
	}
	if err := json.NewEncoder(&buf).Encode(queryBody); err != nil {
		return
	}

	res, err := w.es.Search(
		w.es.Search.WithContext(context.Background()),
		w.es.Search.WithIndex(w.indexName),
		w.es.Search.WithBody(&buf),
		w.es.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		slog.Error("ES search failed", "error", err)
		return
	}
	defer res.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return
	}

	var ids []string
	if hits, ok := result["hits"].(map[string]interface{}); ok {
		if hitList, ok := hits["hits"].([]interface{}); ok {
			for _, hit := range hitList {
				if h, ok := hit.(map[string]interface{}); ok {
					if id, ok := h["_id"].(string); ok {
						ids = append(ids, id)
					}
				}
			}
		}
	}

	// Respond via NATS
	w.nats.Encoded.Publish(reply, ids)
}

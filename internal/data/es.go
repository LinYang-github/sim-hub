package data

import (
	"fmt"

	"sim-hub/internal/conf"

	"github.com/elastic/go-elasticsearch/v7"
)

// NewElasticsearch initializes the ES client
func NewElasticsearch(c *conf.Elasticsearch) (*elasticsearch.Client, error) {
	if len(c.Addresses) == 0 {
		return nil, fmt.Errorf("elasticsearch addresses cannot be empty")
	}

	cfg := elasticsearch.Config{
		Addresses: c.Addresses,
		Username:  c.Username,
		Password:  c.Password,
	}

	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("error creating the elasticsearch client: %w", err)
	}

	// Ping to verify connection
	res, err := es.Info()
	if err != nil {
		return nil, fmt.Errorf("error getting elasticsearch info: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("elasticsearch info request failed: %s", res.String())
	}

	return es, nil
}

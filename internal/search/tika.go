package search

import (
	"fmt"
	"io"
	"net/http"
)

// TikaClient for extracting text from documents
type TikaClient struct {
	BaseURL string
	Client  *http.Client
}

func NewTikaClient(baseURL string) *TikaClient {
	return &TikaClient{
		BaseURL: baseURL,
		Client:  &http.Client{},
	}
}

// Extract text content from a file stream
func (t *TikaClient) Extract(reader io.Reader) (string, error) {
	url := fmt.Sprintf("%s/tika", t.BaseURL)
	req, err := http.NewRequest("PUT", url, reader)
	if err != nil {
		return "", err
	}

	// Tika accepts header to hint content type, but usually detects automatically.
	// Just sending raw bytes to /tika endpoint returns plain text by default.
	req.Header.Set("Accept", "text/plain")

	resp, err := t.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("tika returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

package simhub

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListResources(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/resources" {
			t.Errorf("expected path /api/v1/resources, got %s", r.URL.Path)
		}
		res := ResourceListResponse{
			Items: []Resource{
				{ID: "1", Name: "test"},
			},
			Total: 1,
		}
		json.NewEncoder(w).Encode(res)
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	resp, err := client.ListResources(context.Background(), "scenario", 1, 10)
	if err != nil {
		t.Fatal(err)
	}

	if resp.Total != 1 {
		t.Errorf("expected total 1, got %d", resp.Total)
	}
	if resp.Items[0].Name != "test" {
		t.Errorf("expected name test, got %s", resp.Items[0].Name)
	}
}

func TestGetResource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res := Resource{ID: "123", Name: "details"}
		json.NewEncoder(w).Encode(res)
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	res, err := client.GetResource(context.Background(), "123")
	if err != nil {
		t.Fatal(err)
	}

	if res.ID != "123" {
		t.Errorf("expected ID 123, got %s", res.ID)
	}
}

func TestErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	_, err := client.GetResource(context.Background(), "123")
	if err == nil {
		t.Error("expected error, got nil")
	}
}

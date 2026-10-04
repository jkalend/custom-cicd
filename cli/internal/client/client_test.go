package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientWrappedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": []map[string]any{
				{"id": "p-1", "name": "Pipeline 1", "status": "success"},
			},
		})
	}))
	defer server.Close()

	c := NewClient(server.URL)
	pipelines, err := c.ListPipelines()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pipelines) != 1 || pipelines[0].Name != "Pipeline 1" {
		t.Errorf("unexpected pipelines result: %+v", pipelines)
	}
}

func TestClientErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"error":   "name and steps are required",
		})
	}))
	defer server.Close()

	c := NewClient(server.URL)
	_, err := c.CreatePipeline(&Pipeline{})
	if err == nil {
		t.Fatalf("expected error from failed request")
	}
}

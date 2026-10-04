package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"laya-cicd-backend/internal/engine"
	"laya-cicd-backend/internal/laya"
)

func newTestServer(t *testing.T) *Server {
	tmpDir := t.TempDir()
	dataFile := filepath.Join(tmpDir, "api_test.json")
	client := &laya.Client{
		GatewayURL: "http://127.0.0.1:9999/v1/evaluate",
		Model:      "convaiinnovations/laya",
		RecordFile: filepath.Join(tmpDir, "decisions.jsonl"),
	}
	analyzer := laya.NewAnalyzer(client)
	eng, err := engine.New(dataFile, analyzer)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	return &Server{Engine: eng, Laya: client, Analyzer: analyzer}
}

func doRequest(handler http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	var bodyReader *bytes.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(raw)
	} else {
		bodyReader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func parseResponse(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse json response %q: %v", rec.Body.String(), err)
	}
	return resp
}

func TestHealthEndpoints(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()

	for _, path := range []string{"/health", "/api/health"} {
		rec := doRequest(h, "GET", path, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s returned status %d", path, rec.Code)
		}
		resp := parseResponse(t, rec)
		if resp["success"] != true {
			t.Errorf("expected success true")
		}
		data, ok := resp["data"].(map[string]any)
		if !ok || data["status"] != "healthy" {
			t.Errorf("expected status healthy, got: %+v", data)
		}
		if data["timestamp"] == "" {
			t.Errorf("expected timestamp in health data")
		}
	}
}

func TestPipelineLifecycleAPI(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()

	// 1. Create pipeline with invalid input (empty)
	rec := doRequest(h, "POST", "/api/pipelines", map[string]any{})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty pipeline, got %d", rec.Code)
	}

	// 2. Create valid pipeline
	createPayload := map[string]any{
		"name":        "API Test Pipeline",
		"description": "Integration test",
		"steps": []map[string]any{
			{"name": "step1", "command": "echo step1", "timeout": 10},
		},
	}
	rec = doRequest(h, "POST", "/api/pipelines", createPayload)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	data := resp["data"].(map[string]any)
	pipeID, ok := data["pipeline_id"].(string)
	if !ok || pipeID == "" {
		t.Fatalf("expected pipeline_id, got %+v", data)
	}

	// 3. Get pipeline
	rec = doRequest(h, "GET", "/api/pipelines/"+pipeID, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	pipeData := parseResponse(t, rec)["data"].(map[string]any)
	if pipeData["name"] != "API Test Pipeline" {
		t.Errorf("expected name 'API Test Pipeline', got %v", pipeData["name"])
	}

	// 4. List pipelines
	rec = doRequest(h, "GET", "/api/pipelines", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	listData := parseResponse(t, rec)["data"].([]any)
	if len(listData) != 1 {
		t.Errorf("expected 1 pipeline, got %d", len(listData))
	}

	// 5. Run pipeline synchronously
	rec = doRequest(h, "POST", "/api/pipelines/"+pipeID+"/run?background=false", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	runData := parseResponse(t, rec)["data"].(map[string]any)
	runID := runData["run_id"].(string)

	// 6. Get run
	rec = doRequest(h, "GET", "/api/runs/"+runID, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	runDetails := parseResponse(t, rec)["data"].(map[string]any)
	if runDetails["status"] != "success" {
		t.Errorf("expected status 'success', got %v", runDetails["status"])
	}

	// 7. Delete run
	rec = doRequest(h, "DELETE", "/api/runs/"+runID, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on delete run, got %d", rec.Code)
	}

	// 8. Delete pipeline
	rec = doRequest(h, "DELETE", "/api/pipelines/"+pipeID, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on delete pipeline, got %d", rec.Code)
	}

	// 9. Verify 404 after deletion
	rec = doRequest(h, "GET", "/api/pipelines/"+pipeID, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for deleted pipeline, got %d", rec.Code)
	}
}

func TestCreateAndRunAPI(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()

	payload := map[string]any{
		"name": "Create And Run",
		"steps": []map[string]any{
			{"name": "step1", "command": "echo test", "timeout": 5},
		},
	}
	rec := doRequest(h, "POST", "/api/pipelines/run", payload)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	data := parseResponse(t, rec)["data"].(map[string]any)
	if data["pipeline_id"] == "" || data["run_id"] == "" {
		t.Errorf("expected pipeline_id and run_id, got: %+v", data)
	}
}

func TestAILayerAPI(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()

	// 1. Status
	rec := doRequest(h, "GET", "/api/ai/status", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for AI status, got %d", rec.Code)
	}
	statusData := parseResponse(t, rec)["data"].(map[string]any)
	if statusData["gateway"] == "" {
		t.Errorf("expected gateway field in AI status")
	}

	// 2. Triage Logs
	rec = doRequest(h, "POST", "/api/ai/triage", map[string]any{
		"logs":   "fatal error: runtime: out of memory",
		"source": "test",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for triage, got %d: %s", rec.Code, rec.Body.String())
	}
	triageData := parseResponse(t, rec)["data"].(map[string]any)
	if triageData["action"] == "" {
		t.Errorf("expected action field in triage data")
	}

	// 3. Route Issue
	rec = doRequest(h, "POST", "/api/ai/issue", map[string]any{
		"title":  "Crash on click",
		"body":   "App panics when opening settings page",
		"labels": []string{"ui"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for issue routing, got %d: %s", rec.Code, rec.Body.String())
	}
	issueData := parseResponse(t, rec)["data"].(map[string]any)
	if issueData["component"] == "" {
		t.Errorf("expected component field in issue data")
	}

	// 4. Decisions list
	rec = doRequest(h, "GET", "/api/ai/decisions?limit=10", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for decisions, got %d", rec.Code)
	}

	// 5. Notifications
	rec = doRequest(h, "GET", "/api/ai/notifications", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for notifications, got %d", rec.Code)
	}
}

func TestRunLogsAndArtifactsAPI(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()

	pID := srv.Engine.CreatePipeline(engine.PipelineDefinition{
		Name: "log-art-pipeline",
		Steps: []engine.Step{
			{Name: "step-1", Command: "echo hello logs"},
		},
	})
	runID, err := srv.Engine.StartRun(pID, false)
	if err != nil {
		t.Fatalf("failed to start run: %v", err)
	}

	// 1. Logs JSON endpoint
	req := httptest.NewRequest("GET", "/api/runs/"+runID+"/logs", nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for logs json, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	if resp["success"] != true {
		t.Errorf("expected success true in logs response")
	}

	// 2. Artifacts endpoint
	rec = doRequest(h, "GET", "/api/runs/"+runID+"/artifacts", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for artifacts, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAIFixAndFeedbackAPI(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()

	// 1. Suggest Fix
	fixReq := map[string]any{
		"pipeline": "build-pipe",
		"step": map[string]any{
			"name":    "npm-build",
			"command": "npm run build",
			"error":   "Error: Cannot find module 'react'",
			"output":  "npm ERR! missing dependency",
		},
	}
	rec := doRequest(h, "POST", "/api/ai/fix", fixReq)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for ai fix, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	data, ok := resp["data"].(map[string]any)
	if !ok || data["hypothesis"] == "" {
		t.Fatalf("expected hypothesis in fix suggestion: %+v", data)
	}

	// 2. Feedback Save & List
	feedbackReq := map[string]any{
		"decision_id": "12345",
		"rating":      "positive",
		"comment":     "Accurate diagnostic!",
	}
	rec = doRequest(h, "POST", "/api/ai/feedback", feedbackReq)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for feedback save, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = doRequest(h, "GET", "/api/ai/feedback?limit=5", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for feedback list, got %d: %s", rec.Code, rec.Body.String())
	}
	feedResp := parseResponse(t, rec)
	feedList, ok := feedResp["data"].([]any)
	if !ok || len(feedList) != 1 {
		t.Fatalf("expected 1 feedback entry, got: %+v", feedResp)
	}
}

func TestGitHubWebhookAPI(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()

	srv.Engine.CreatePipeline(engine.PipelineDefinition{
		Name: "my-repo-ci",
		Steps: []engine.Step{
			{Name: "build", Command: "echo build"},
		},
	})

	// 1. Ping event
	req := httptest.NewRequest("POST", "/webhooks/github", bytes.NewReader([]byte("{}")))
	req.Header.Set("X-GitHub-Event", "ping")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for ping, got %d", rec.Code)
	}

	// 2. Push event triggering pipeline
	payload := []byte(`{"ref":"refs/heads/main","repository":{"name":"my-repo"}}`)
	req = httptest.NewRequest("POST", "/webhooks/github", bytes.NewReader(payload))
	req.Header.Set("X-GitHub-Event", "push")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for webhook push, got %d: %s", rec.Code, rec.Body.String())
	}
	resp := parseResponse(t, rec)
	data := resp["data"].(map[string]any)
	if data["triggered"] != true || data["run_id"] == "" {
		t.Fatalf("expected webhook to trigger pipeline run, got: %+v", data)
	}
}

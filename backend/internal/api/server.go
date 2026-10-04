// Package api serves the REST API: the original pipeline/run endpoints
// (wire-compatible with the Flask backend the CLI and frontend expect) plus
// the AI layer endpoints (status, decisions, playgrounds).
package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/rs/cors"

	"laya-cicd-backend/internal/engine"
	"laya-cicd-backend/internal/laya"
)

// Server wires the engine and Laya analyzer to HTTP.
type Server struct {
	Engine   *engine.Engine
	Laya     *laya.Client
	Analyzer *laya.Analyzer
}

// NewServer builds the server and its dependencies.
func NewServer(dataFile string) (*Server, error) {
	client := laya.DefaultClient()
	analyzer := laya.NewAnalyzer(client)
	eng, err := engine.New(dataFile, analyzer)
	if err != nil {
		return nil, err
	}
	return &Server{Engine: eng, Laya: client, Analyzer: analyzer}, nil
}

// Handler returns the fully-wired HTTP handler with CORS.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /api/health", s.handleHealth)

	// Pipelines. Routes are registered twice: the frontend/nginx talks to
	// the root paths, the Go CLI (and external API users) use /api/* — the
	// backend serves both so it works with or without the reverse proxy.
	mux.HandleFunc("GET /pipelines", s.handleListPipelines)
	mux.HandleFunc("POST /pipelines", s.handleCreatePipeline)
	mux.HandleFunc("GET /pipelines/{id}", s.handleGetPipeline)
	mux.HandleFunc("DELETE /pipelines/{id}", s.handleDeletePipeline)
	mux.HandleFunc("POST /pipelines/{id}/run", s.handleRunPipeline)
	mux.HandleFunc("POST /pipelines/{id}/cancel", s.handleCancelPipeline)
	mux.HandleFunc("POST /pipelines/run", s.handleCreateAndRun)

	mux.HandleFunc("GET /api/pipelines", s.handleListPipelines)
	mux.HandleFunc("POST /api/pipelines", s.handleCreatePipeline)
	mux.HandleFunc("GET /api/pipelines/{id}", s.handleGetPipeline)
	mux.HandleFunc("DELETE /api/pipelines/{id}", s.handleDeletePipeline)
	mux.HandleFunc("POST /api/pipelines/{id}/run", s.handleRunPipeline)
	mux.HandleFunc("POST /api/pipelines/{id}/cancel", s.handleCancelPipeline)
	mux.HandleFunc("POST /api/pipelines/run", s.handleCreateAndRun)

	// Runs
	mux.HandleFunc("GET /runs", s.handleListRuns)
	mux.HandleFunc("GET /runs/{id}", s.handleGetRun)
	mux.HandleFunc("DELETE /runs/{id}", s.handleDeleteRun)
	mux.HandleFunc("POST /runs/{id}/cancel", s.handleCancelRun)
	mux.HandleFunc("GET /runs/{id}/logs", s.handleRunLogs)
	mux.HandleFunc("GET /runs/{id}/artifacts", s.handleListArtifacts)
	mux.HandleFunc("GET /runs/{id}/artifacts/{step}/{name}", s.handleGetArtifactFile)

	mux.HandleFunc("GET /api/runs", s.handleListRuns)
	mux.HandleFunc("GET /api/runs/{id}", s.handleGetRun)
	mux.HandleFunc("DELETE /api/runs/{id}", s.handleDeleteRun)
	mux.HandleFunc("POST /api/runs/{id}/cancel", s.handleCancelRun)
	mux.HandleFunc("GET /api/runs/{id}/logs", s.handleRunLogs)
	mux.HandleFunc("GET /api/runs/{id}/artifacts", s.handleListArtifacts)
	mux.HandleFunc("GET /api/runs/{id}/artifacts/{step}/{name}", s.handleGetArtifactFile)

	// AI layer
	mux.HandleFunc("GET /ai/status", s.handleAIStatus)
	mux.HandleFunc("GET /ai/decisions", s.handleListDecisions)
	mux.HandleFunc("POST /ai/triage", s.handleTriageLogs)
	mux.HandleFunc("POST /ai/issue", s.handleRouteIssue)
	mux.HandleFunc("GET /ai/notifications", s.handleListNotifications)
	mux.HandleFunc("POST /ai/fix", s.handleSuggestFix)
	mux.HandleFunc("POST /ai/feedback", s.handleSaveFeedback)
	mux.HandleFunc("GET /ai/feedback", s.handleListFeedback)

	mux.HandleFunc("GET /api/ai/status", s.handleAIStatus)
	mux.HandleFunc("GET /api/ai/decisions", s.handleListDecisions)
	mux.HandleFunc("POST /api/ai/triage", s.handleTriageLogs)
	mux.HandleFunc("POST /api/ai/issue", s.handleRouteIssue)
	mux.HandleFunc("GET /api/ai/notifications", s.handleListNotifications)
	mux.HandleFunc("POST /api/ai/fix", s.handleSuggestFix)
	mux.HandleFunc("POST /api/ai/feedback", s.handleSaveFeedback)
	mux.HandleFunc("GET /api/ai/feedback", s.handleListFeedback)

	// Webhooks
	mux.HandleFunc("POST /webhooks/github", s.handleGitHubWebhook)
	mux.HandleFunc("POST /api/webhooks/github", s.handleGitHubWebhook)

	return cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	}).Handler(mux)
}

// --- response helpers --------------------------------------------------------

// The CLI unwraps {data, success, error}; the original Flask bridge used the
// same shape via agent_interface.py. The Next.js API routes strip nothing —
// they forward the backend JSON directly — so both consumers work with the
// wrapped shape.
func writeData(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"data":    data,
	})
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": false,
		"error":   message,
	})
}

func readJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return errEmptyBody
	}
	return json.Unmarshal(body, target)
}

var errEmptyBody = &HTTPError{http.StatusBadRequest, "request body required"}

// HTTPError is an error carrying a status code.
type HTTPError struct {
	Status  int
	Message string
}

func (e *HTTPError) Error() string { return e.Message }

// --- health ---------------------------------------------------------------------

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeData(w, map[string]any{
		"status":       "healthy",
		"timestamp":    nowISO(),
		"agent_status": "running",
	})
}

// --- pipelines ------------------------------------------------------------------

func (s *Server) handleListPipelines(w http.ResponseWriter, r *http.Request) {
	writeData(w, s.Engine.ListPipelines())
}

func (s *Server) handleCreatePipeline(w http.ResponseWriter, r *http.Request) {
	var def engine.PipelineDefinition
	if err := readJSON(r, &def); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(def.Name) == "" || len(def.Steps) == 0 {
		writeError(w, http.StatusBadRequest, "name and steps are required")
		return
	}
	for i, st := range def.Steps {
		if strings.TrimSpace(st.Name) == "" || strings.TrimSpace(st.Command) == "" {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("step %d requires non-empty name and command", i+1))
			return
		}
	}
	id := s.Engine.CreatePipeline(def)
	writeData(w, map[string]any{
		"pipeline_id": id,
		"status":      "created",
	})
}

func (s *Server) handleGetPipeline(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, ok := s.Engine.GetPipeline(id)
	if !ok {
		writeError(w, http.StatusNotFound, "Pipeline not found")
		return
	}
	writeData(w, p)
}

func (s *Server) handleDeletePipeline(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if s.Engine.DeletePipeline(id) {
		writeData(w, map[string]any{"status": "deleted"})
	} else {
		writeError(w, http.StatusNotFound, "Pipeline not found or cannot be deleted")
	}
}

func (s *Server) handleRunPipeline(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	background := r.URL.Query().Get("background") != "false"
	runID, err := s.Engine.StartRun(id, background)
	if err != nil {
		writeError(w, http.StatusNotFound, "Failed to start pipeline: "+err.Error())
		return
	}
	status := "running"
	if !background {
		status = "completed"
	}
	writeData(w, map[string]any{
		"run_id": runID,
		"status": status,
	})
}

func (s *Server) handleCancelPipeline(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if s.Engine.CancelPipeline(id) {
		writeData(w, map[string]any{"status": "cancelled"})
	} else {
		writeError(w, http.StatusNotFound, "Pipeline not found or not running")
	}
}

func (s *Server) handleCreateAndRun(w http.ResponseWriter, r *http.Request) {
	var def engine.PipelineDefinition
	if err := readJSON(r, &def); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(def.Name) == "" || len(def.Steps) == 0 {
		writeError(w, http.StatusBadRequest, "name and steps are required")
		return
	}
	for i, st := range def.Steps {
		if strings.TrimSpace(st.Name) == "" || strings.TrimSpace(st.Command) == "" {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("step %d requires non-empty name and command", i+1))
			return
		}
	}
	id := s.Engine.CreatePipeline(def)
	runID, err := s.Engine.StartRun(id, true)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to start run: "+err.Error())
		return
	}
	writeData(w, map[string]any{
		"pipeline_id": id,
		"run_id":      runID,
		"status":      "running",
	})
}

// --- runs ------------------------------------------------------------------------

func (s *Server) handleListRuns(w http.ResponseWriter, r *http.Request) {
	writeData(w, s.Engine.ListRuns(r.URL.Query().Get("pipeline_id")))
}

func (s *Server) handleGetRun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	run, ok := s.Engine.GetRun(id)
	if !ok {
		writeError(w, http.StatusNotFound, "Run not found")
		return
	}
	writeData(w, run)
}

func (s *Server) handleDeleteRun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if s.Engine.DeleteRun(id) {
		writeData(w, map[string]any{"status": "deleted"})
	} else {
		writeError(w, http.StatusNotFound, "Run not found or currently running")
	}
}

func (s *Server) handleCancelRun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if s.Engine.CancelRun(id) {
		writeData(w, map[string]any{"status": "cancelled"})
	} else {
		writeError(w, http.StatusNotFound, "Run not found or not running")
	}
}

// --- AI layer --------------------------------------------------------------------

func (s *Server) handleAIStatus(w http.ResponseWriter, r *http.Request) {
	writeData(w, s.Laya.Status())
}

func (s *Server) handleListDecisions(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	writeData(w, s.Laya.ListDecisions(limit))
}

func (s *Server) handleTriageLogs(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Logs   string `json:"logs"`
		Source string `json:"source"`
	}
	if err := readJSON(r, &req); err != nil || strings.TrimSpace(req.Logs) == "" {
		writeError(w, http.StatusBadRequest, "logs required")
		return
	}
	writeData(w, s.Analyzer.TriageLogs(req.Logs, req.Source))
}

func (s *Server) handleRouteIssue(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title  string   `json:"title"`
		Body   string   `json:"body"`
		Labels []string `json:"labels"`
	}
	if err := readJSON(r, &req); err != nil || strings.TrimSpace(req.Title) == "" {
		writeError(w, http.StatusBadRequest, "title required")
		return
	}
	writeData(w, s.Analyzer.RouteIssue(req.Title, req.Body, req.Labels))
}

func (s *Server) handleListNotifications(w http.ResponseWriter, r *http.Request) {
	writeData(w, s.Engine.ListNotifications())
}

func (s *Server) handleRunLogs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing run id")
		return
	}
	_, found := s.Engine.GetRun(id)
	if !found {
		writeError(w, http.StatusNotFound, fmt.Sprintf("run %s not found", id))
		return
	}

	// JSON response when requested
	if strings.Contains(r.Header.Get("Accept"), "application/json") && !strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
		history, _, unsub := s.Engine.Logs().Subscribe(id)
		unsub()
		writeData(w, history)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	history, ch, unsubscribe := s.Engine.Logs().Subscribe(id)
	defer unsubscribe()

	for _, ev := range history {
		data, err := json.Marshal(ev)
		if err == nil {
			fmt.Fprintf(w, "data: %s\n\n", data)
		}
	}
	flusher.Flush()

	// If the run is already completed and no buffered events remain, close stream
	run, _ := s.Engine.GetRun(id)
	status, _ := run["status"].(string)
	if status != string(engine.StatusRunning) && status != string(engine.StatusPending) && len(ch) == 0 {
		return
	}

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		case ev, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(ev)
			if err == nil {
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			}
			run, found := s.Engine.GetRun(id)
			if found {
				st, _ := run["status"].(string)
				if st != string(engine.StatusRunning) && st != string(engine.StatusPending) && len(ch) == 0 {
					return
				}
			}
		}
	}
}

func (s *Server) handleListArtifacts(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing run id")
		return
	}
	artifacts, err := engine.ListRunArtifacts(s.Engine.ArtifactsDir(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, artifacts)
}

func (s *Server) handleGetArtifactFile(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	step := r.PathValue("step")
	name := r.PathValue("name")

	if strings.Contains(runID, "..") || strings.Contains(step, "..") || strings.Contains(name, "..") ||
		strings.ContainsAny(name, "/\\") {
		writeError(w, http.StatusBadRequest, "invalid artifact path parameter")
		return
	}

	targetDir := filepath.Join(s.Engine.ArtifactsDir(), runID, step)
	filePath := filepath.Join(targetDir, name)

	rel, err := filepath.Rel(s.Engine.ArtifactsDir(), filePath)
	if err != nil || strings.HasPrefix(rel, "..") {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		writeError(w, http.StatusNotFound, "artifact not found")
		return
	}

	http.ServeFile(w, r, filePath)
}

func (s *Server) handleSuggestFix(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Pipeline string      `json:"pipeline"`
		Step     engine.Step `json:"step"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid json: %s", err))
		return
	}
	writeData(w, s.Analyzer.SuggestFix(req.Step, req.Pipeline))
}

func (s *Server) handleSaveFeedback(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid json: %s", err))
		return
	}
	if err := s.Laya.SaveFeedback(body); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, map[string]any{"saved": true})
}

func (s *Server) handleListFeedback(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	writeData(w, s.Laya.ListFeedback(limit))
}

func (s *Server) handleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}

	secret := os.Getenv("GITHUB_WEBHOOK_SECRET")
	if secret != "" {
		sig := r.Header.Get("X-Hub-Signature-256")
		if sig == "" {
			writeError(w, http.StatusUnauthorized, "missing signature")
			return
		}
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		expectedMAC := "sha256=" + hex.EncodeToString(mac.Sum(nil))
		if !hmac.Equal([]byte(sig), []byte(expectedMAC)) {
			writeError(w, http.StatusUnauthorized, "invalid signature")
			return
		}
	}

	event := r.Header.Get("X-GitHub-Event")
	if event == "ping" {
		writeData(w, map[string]any{"pong": true})
		return
	}

	var payload struct {
		Ref        string `json:"ref"`
		Repository struct {
			Name string `json:"name"`
		} `json:"repository"`
	}
	_ = json.Unmarshal(body, &payload)

	pipelines := s.Engine.ListPipelines()
	var targetPipelineID string
	repoName := strings.ToLower(payload.Repository.Name)

	for _, p := range pipelines {
		name := strings.ToLower(fmt.Sprintf("%v", p["name"]))
		if repoName != "" && strings.Contains(name, repoName) {
			targetPipelineID = fmt.Sprintf("%v", p["id"])
			break
		}
	}
	if targetPipelineID == "" && len(pipelines) > 0 {
		targetPipelineID = fmt.Sprintf("%v", pipelines[0]["id"])
	}

	if targetPipelineID == "" {
		writeError(w, http.StatusNotFound, "no pipeline available to trigger")
		return
	}

	runID, err := s.Engine.StartRun(targetPipelineID, true)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeData(w, map[string]any{
		"event":       event,
		"pipeline_id": targetPipelineID,
		"run_id":      runID,
		"triggered":   true,
	})
}

// --- main plumbing ----------------------------------------------------------------
func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// Run starts the HTTP server on PORT (default 8000) with graceful shutdown.
func Run() {
	dataFile := os.Getenv("DATA_FILE")
	if dataFile == "" {
		dataFile = "data/agent_data.json"
	}
	srv, err := NewServer(dataFile)
	if err != nil {
		log.Fatalf("failed to init server: %v", err)
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	addr := ":" + port
	httpServer := &http.Server{
		Addr:    addr,
		Handler: srv.Handler(),
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("laya-cicd backend listening on %s (data: %s)", addr, dataFile)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down server gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("server forced to shutdown: %v", err)
	}
	log.Println("Server stopped")
}

// Package api serves the REST API: the original pipeline/run endpoints
// (wire-compatible with the Flask backend the CLI and frontend expect) plus
// the AI layer endpoints (status, decisions, playgrounds).
package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rs/cors"

	"jev-cicd-backend/internal/engine"
	"jev-cicd-backend/internal/jev"
)

// Server wires the engine and Jev analyzer to HTTP.
type Server struct {
	Engine   *engine.Engine
	Jev      *jev.Client
	Analyzer *jev.Analyzer
}

// NewServer builds the server and its dependencies.
func NewServer(dataFile string) (*Server, error) {
	client := jev.DefaultClient()
	analyzer := jev.NewAnalyzer(client)
	eng, err := engine.New(dataFile, analyzer)
	if err != nil {
		return nil, err
	}
	return &Server{Engine: eng, Jev: client, Analyzer: analyzer}, nil
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

	mux.HandleFunc("GET /api/runs", s.handleListRuns)
	mux.HandleFunc("GET /api/runs/{id}", s.handleGetRun)
	mux.HandleFunc("DELETE /api/runs/{id}", s.handleDeleteRun)
	mux.HandleFunc("POST /api/runs/{id}/cancel", s.handleCancelRun)

	// AI layer
	mux.HandleFunc("GET /ai/status", s.handleAIStatus)
	mux.HandleFunc("GET /ai/decisions", s.handleListDecisions)
	mux.HandleFunc("POST /ai/triage", s.handleTriageLogs)
	mux.HandleFunc("POST /ai/issue", s.handleRouteIssue)
	mux.HandleFunc("GET /ai/notifications", s.handleListNotifications)

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
	if def.Name == "" || len(def.Steps) == 0 {
		writeError(w, http.StatusBadRequest, "name and steps are required")
		return
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
	if def.Name == "" || len(def.Steps) == 0 {
		writeError(w, http.StatusBadRequest, "name and steps are required")
		return
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
	writeData(w, s.Jev.Status())
}

func (s *Server) handleListDecisions(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	writeData(w, s.Jev.ListDecisions(limit))
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

// --- main plumbing ----------------------------------------------------------------
func nowISO() string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}

// Run starts the HTTP server on PORT (default 8000).
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
	log.Printf("jev-cicd backend listening on %s (data: %s)", addr, dataFile)
	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

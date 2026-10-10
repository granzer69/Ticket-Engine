package api

import (
	"context"
	"encoding/json"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ticketengine/verification-lab/internal/failurelab"
	"ticketengine/verification-lab/internal/metrics"
	"ticketengine/verification-lab/internal/runner"
)

//go:embed static_fallback/*
var fallbackFS embed.FS

type Server struct {
	Addr   string
	Engine *runner.Engine
	Hub    *metrics.Hub
}

type runRequest struct {
	Preset          string   `json:"preset"`
	Claims          []string `json:"claims"`
	Gates           []string `json:"gates"`
	Scenarios       []string `json:"scenarios"`
	ConfirmHeavy    bool     `json:"confirm_heavy"`
	ConfirmFault    bool     `json:"confirm_fault"`
}

func (s *Server) ListenAndServe() error {
	if s.Hub != nil {
		s.Hub.Start()
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/snapshot", s.handleSnapshot)
	mux.HandleFunc("/api/stream", s.handleSSE)
	mux.HandleFunc("/api/runs", s.handleRuns)
	mux.HandleFunc("/api/runs/", s.handleRunByID)
	mux.HandleFunc("/api/failure-lab/scenarios", s.handleFailureLabCatalog)

	staticHandler := s.staticHandler()
	mux.Handle("/", staticHandler)

	srv := &http.Server{Addr: s.Addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("verilab listening on %s (target %s)", s.Addr, s.Engine.Collector().TargetBase)
	return srv.ListenAndServe()
}

func (s *Server) staticHandler() http.Handler {
	if dir := os.Getenv("VERILAB_WEB_DIST"); dir != "" {
		if st, err := os.Stat(dir); err == nil && st.IsDir() {
			return http.FileServer(http.Dir(dir))
		}
	}
	candidate := filepath.Join("verification-lab", "web", "dist")
	if st, err := os.Stat(candidate); err == nil && st.IsDir() {
		return http.FileServer(http.Dir(candidate))
	}
	sub, err := fs.Sub(fallbackFS, "static_fallback")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "ui unavailable", 500)
		})
	}
	return http.FileServer(http.FS(sub))
}

func (s *Server) latestSnapshot(ctx context.Context) metrics.Snapshot {
	if s.Hub != nil {
		snap := s.Hub.Latest()
		if snap.Sequence > 0 {
			snap.StaleMs = time.Since(snap.At).Milliseconds()
			return snap
		}
	}
	snap, _ := s.Engine.Collector().Collect(ctx)
	return snap
}

func (s *Server) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, s.latestSnapshot(r.Context()))
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", 500)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ctx := r.Context()
	if s.Hub != nil {
		ch := s.Hub.Subscribe(ctx)
		for snap := range ch {
			snap.StaleMs = time.Since(snap.At).Milliseconds()
			b, _ := json.Marshal(snap)
			_, _ = w.Write([]byte("event: snapshot\n"))
			_, _ = w.Write([]byte("data: "))
			_, _ = w.Write(b)
			_, _ = w.Write([]byte("\n\n"))
			flusher.Flush()
		}
		return
	}

	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			snap := s.latestSnapshot(ctx)
			b, _ := json.Marshal(snap)
			_, _ = w.Write([]byte("event: snapshot\n"))
			_, _ = w.Write([]byte("data: "))
			_, _ = w.Write(b)
			_, _ = w.Write([]byte("\n\n"))
			flusher.Flush()
		}
	}
}

func (s *Server) handleRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req runRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", 400)
		return
	}
	ctx := context.Background()
	var id string
	var err error
	switch strings.ToLower(req.Preset) {
	case "smoke", "":
		id, err = s.Engine.StartSmoke(ctx)
	case "claims":
		id, err = s.Engine.StartClaims(ctx, req.Claims)
	case "gates", "phase-gates":
		id, err = s.Engine.StartGates(ctx, req.Gates)
	case "failure-lab":
		if !req.ConfirmFault {
			http.Error(w, "confirm_fault required for failure-lab preset", 400)
			return
		}
		id, err = s.Engine.StartFailureLab(ctx, req.Scenarios)
	case "1k", "load-1k":
		id, err = s.Engine.StartBoundedLoad(ctx, "1k", 1_000)
	case "10k", "load-10k":
		id, err = s.Engine.StartBoundedLoad(ctx, "10k", 10_000)
	case "25k", "load-25k":
		id, err = s.Engine.StartBoundedLoad(ctx, "25k", 25_000)
	case "50k", "load-50k":
		id, err = s.Engine.StartBoundedLoad(ctx, "50k", 50_000)
	case "heavy-100k", "100k":
		if !req.ConfirmHeavy {
			http.Error(w, "confirm_heavy required for 100k preset", 400)
			return
		}
		id, err = s.Engine.StartHeavy(ctx)
	case "benchmark-smoke", "benchmark-exhaustion-mini", "benchmark-saturated-15k":
		id, err = s.Engine.StartBenchmark(ctx, strings.ToLower(req.Preset))
	case "benchmark-exhaustion-15k":
		if !req.ConfirmHeavy {
			http.Error(w, "confirm_heavy required for benchmark-exhaustion-15k preset", 400)
			return
		}
		id, err = s.Engine.StartBenchmark(ctx, "benchmark-exhaustion-15k")
	default:
		http.Error(w, "unknown preset", 400)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, map[string]string{"run_id": id})
}

func (s *Server) handleRunByID(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/runs/")
	if rest == "" {
		http.NotFound(w, r)
		return
	}
	parts := strings.Split(rest, "/")
	id := parts[0]
	export := len(parts) == 2 && parts[1] == "export"
	rep, ok := s.Engine.Get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if export {
		w.Header().Set("Content-Disposition", "attachment; filename="+id+".json")
	}
	writeJSON(w, rep)
}

func (s *Server) handleFailureLabCatalog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, failurelab.Catalog())
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

package server

import (
	"encoding/json"
	"github.com/stockyard-dev/stockyard-seismograph/internal/store"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

type Server struct {
	db      *store.DB
	mux     *http.ServeMux
	limits  Limits
	dataDir string
	pCfg    map[string]json.RawMessage
}

func New(db *store.DB, limits Limits, dataDir string) *Server {
	s := &Server{db: db, mux: http.NewServeMux(), limits: limits, dataDir: dataDir}

	s.mux.HandleFunc("GET /api/errors", s.listErrors)
	s.mux.HandleFunc("POST /api/errors", s.ingestError)
	s.mux.HandleFunc("GET /api/errors/{id}", s.getError)
	s.mux.HandleFunc("PATCH /api/errors/{id}/status", s.setStatus)
	s.mux.HandleFunc("DELETE /api/errors/{id}", s.deleteError)
	s.mux.HandleFunc("GET /api/errors/{id}/occurrences", s.listOccurrences)
	s.mux.HandleFunc("GET /api/sources", s.listSources)
	s.mux.HandleFunc("GET /api/stats", s.stats)
	s.mux.HandleFunc("GET /api/health", s.health)
	s.mux.HandleFunc("GET /api/tier", func(w http.ResponseWriter, r *http.Request) {
		wj(w, 200, map[string]any{"tier": s.limits.Tier, "upgrade_url": "https://stockyard.dev/seismograph/"})
	})
	s.mux.HandleFunc("GET /ui", s.dashboard)
	s.mux.HandleFunc("GET /ui/", s.dashboard)
	s.mux.HandleFunc("GET /", s.root)

	s.loadPersonalConfig()
	s.mux.HandleFunc("GET /api/config", s.configHandler)
	s.mux.HandleFunc("GET /api/extras/{resource}", s.listExtras)
	s.mux.HandleFunc("GET /api/extras/{resource}/{id}", s.getExtras)
	s.mux.HandleFunc("PUT /api/extras/{resource}/{id}", s.putExtras)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }
func wj(w http.ResponseWriter, c int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(c)
	json.NewEncoder(w).Encode(v)
}
func we(w http.ResponseWriter, c int, m string) { wj(w, c, map[string]string{"error": m}) }
func (s *Server) root(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/ui", 302)
}

func (s *Server) listErrors(w http.ResponseWriter, r *http.Request) {
	level := r.URL.Query().Get("level")
	status := r.URL.Query().Get("status")
	source := r.URL.Query().Get("source")
	wj(w, 200, map[string]any{"errors": s.db.List(level, status, source)})
}

func (s *Server) ingestError(w http.ResponseWriter, r *http.Request) {
	if s.limits.MaxItems > 0 {
		all := s.db.List("", "", "")
		if len(all) >= s.limits.MaxItems {
			we(w, 402, "Free tier limit reached. Upgrade at https://stockyard.dev/seismograph/")
			return
		}
	}
	var body struct {
		Title    string `json:"title"`
		Message  string `json:"message"`
		Level    string `json:"level"`
		Source   string `json:"source"`
		Stack    string `json:"stack"`
		Metadata string `json:"metadata"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.Title == "" {
		we(w, 400, "title required")
		return
	}
	evt, err := s.db.Ingest(body.Title, body.Message, body.Level, body.Source, body.Stack, body.Metadata)
	if err != nil {
		we(w, 500, err.Error())
		return
	}
	wj(w, 201, evt)
}

func (s *Server) getError(w http.ResponseWriter, r *http.Request) {
	e := s.db.Get(r.PathValue("id"))
	if e == nil {
		we(w, 404, "not found")
		return
	}
	wj(w, 200, e)
}

func (s *Server) setStatus(w http.ResponseWriter, r *http.Request) {
	e := s.db.Get(r.PathValue("id"))
	if e == nil {
		we(w, 404, "not found")
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	valid := map[string]bool{"open": true, "acknowledged": true, "resolved": true, "ignored": true}
	if !valid[body.Status] {
		we(w, 400, "status must be: open, acknowledged, resolved, ignored")
		return
	}
	s.db.SetStatus(e.ID, body.Status)
	wj(w, 200, s.db.Get(e.ID))
}

func (s *Server) deleteError(w http.ResponseWriter, r *http.Request) {
	if s.db.Get(r.PathValue("id")) == nil {
		we(w, 404, "not found")
		return
	}
	s.db.Delete(r.PathValue("id"))
	wj(w, 200, map[string]string{"status": "deleted"})
}

func (s *Server) listOccurrences(w http.ResponseWriter, r *http.Request) {
	e := s.db.Get(r.PathValue("id"))
	if e == nil {
		we(w, 404, "not found")
		return
	}
	wj(w, 200, map[string]any{"occurrences": s.db.Occurrences(e.Fingerprint)})
}

func (s *Server) listSources(w http.ResponseWriter, r *http.Request) {
	wj(w, 200, map[string]any{"sources": s.db.Sources()})
}

func (s *Server) stats(w http.ResponseWriter, r *http.Request) { wj(w, 200, s.db.Stats()) }
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	stats := s.db.Stats()
	wj(w, 200, map[string]any{"service": "seismograph", "status": "ok", "errors": stats["total"], "open": stats["open"]})
}

// ─── personalization (auto-added) ──────────────────────────────────

func (s *Server) loadPersonalConfig() {
	path := filepath.Join(s.dataDir, "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var cfg map[string]json.RawMessage
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("%s: warning: could not parse config.json: %v", "seismograph", err)
		return
	}
	s.pCfg = cfg
	log.Printf("%s: loaded personalization from %s", "seismograph", path)
}

func (s *Server) configHandler(w http.ResponseWriter, r *http.Request) {
	if s.pCfg == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{}"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.pCfg)
}

func (s *Server) listExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	all := s.db.AllExtras(resource)
	out := make(map[string]json.RawMessage, len(all))
	for id, data := range all {
		out[id] = json.RawMessage(data)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (s *Server) getExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	id := r.PathValue("id")
	data := s.db.GetExtras(resource, id)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(data))
}

func (s *Server) putExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	id := r.PathValue("id")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":"read body"}`, 400)
		return
	}
	var probe map[string]any
	if err := json.Unmarshal(body, &probe); err != nil {
		http.Error(w, `{"error":"invalid json"}`, 400)
		return
	}
	if err := s.db.SetExtras(resource, id, string(body)); err != nil {
		http.Error(w, `{"error":"save failed"}`, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":"saved"}`))
}

// Package server provides the web interface for the DNS scanner
package server

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Server represents the web server
type Server struct {
	Router      *mux.Router
	Jobs        map[string]*ScanJob
	Templates   *template.Template
	Logger      *log.Logger
	mu          sync.Mutex
	StaticDir   string
	TemplateDir string
	WordlistDir string
}

// NewServer creates a new web server instance
func NewServer(staticDir, templateDir, wordlistDir string) *Server {
	logger := log.New(os.Stdout, "[SERVER] ", log.LstdFlags)
	
	// Create server instance
	s := &Server{
		Router:      mux.NewRouter(),
		Jobs:        make(map[string]*ScanJob),
		Logger:      logger,
		StaticDir:   staticDir,
		TemplateDir: templateDir,
		WordlistDir: wordlistDir,
	}
	
	// Parse templates
	s.parseTemplates()
	
	// Set up routes
	s.setupRoutes()
	
	return s
}

// parseTemplates loads and parses HTML templates
func (s *Server) parseTemplates() {
	s.Templates = template.New("").Funcs(template.FuncMap{
		"join": func(strs []string, sep string) string {
			result := ""
			for i, str := range strs {
				if i > 0 {
					result += sep
				}
				result += str
			}
			return result
		},
	})
	
	templateFiles, err := filepath.Glob(filepath.Join(s.TemplateDir, "*.html"))
	if err != nil {
		s.Logger.Fatalf("Failed to find templates: %v", err)
	}
	
	s.Templates = template.Must(s.Templates.ParseFiles(templateFiles...))
}

// setupRoutes configures the HTTP routes
func (s *Server) setupRoutes() {
	// Static files
	fileServer := http.FileServer(http.Dir(s.StaticDir))
	s.Router.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/", fileServer),
	)
	
	// API routes
	s.Router.HandleFunc("/", s.homeHandler).Methods("GET")
	s.Router.HandleFunc("/scan", s.scanHandler).Methods("POST")
	s.Router.HandleFunc("/jobs/{id}", s.jobHandler).Methods("GET")
	s.Router.HandleFunc("/jobs/{id}/csv", s.csvHandler).Methods("GET")
	
	// Middleware for logging
	s.Router.Use(s.loggingMiddleware)
}

// loggingMiddleware logs all HTTP requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.Logger.Printf("%s %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}

// Start starts the web server
func (s *Server) Start(addr string) error {
	s.Logger.Printf("Server starting on %s", addr)
	return http.ListenAndServe(addr, s.Router)
}

// GenerateJobID generates a unique job ID
func (s *Server) GenerateJobID() string {
	return uuid.New().String()
}

// GetJob retrieves a job by ID
func (s *Server) GetJob(id string) (*ScanJob, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, exists := s.Jobs[id]
	return job, exists
}

// AddJob adds a new job to the server
func (s *Server) AddJob(job *ScanJob) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Jobs[job.ID] = job
}

// GetRecentJobs returns the most recent jobs
func (s *Server) GetRecentJobs(limit int) []*ScanJob {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Convert map to slice
	var jobs []*ScanJob
	for _, job := range s.Jobs {
		jobs = append(jobs, job)
	}
	
	// Sort by start time (newest first)
	sortJobsByStartTime(jobs)
	
	// Limit the number of jobs
	if len(jobs) > limit {
		jobs = jobs[:limit]
	}
	
	return jobs
}

// sortJobsByStartTime sorts jobs by start time (newest first)
func sortJobsByStartTime(jobs []*ScanJob) {
	// Sort jobs by start time (newest first)
	for i := 0; i < len(jobs); i++ {
		for j := i + 1; j < len(jobs); j++ {
			if jobs[i].StartTime.Before(jobs[j].StartTime) {
				jobs[i], jobs[j] = jobs[j], jobs[i]
			}
		}
	}
}
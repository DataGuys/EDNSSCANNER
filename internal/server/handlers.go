package server

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// homeHandler handles the home page
func (s *Server) homeHandler(w http.ResponseWriter, r *http.Request) {
	// Get available wordlists
	wordlists, err := getAvailableWordlists(s.WordlistDir)
	if err != nil {
		s.Logger.Printf("Error getting wordlists: %v", err)
		wordlists = []string{}
	}
	
	// Get recent jobs
	recentJobs := s.GetRecentJobs(10)
	
	// Prepare template data
	data := struct {
		Jobs      []*ScanJob
		Wordlists []string
	}{
		Jobs:      recentJobs,
		Wordlists: wordlists,
	}
	
	// Render template
	err = s.Templates.ExecuteTemplate(w, "home.html", data)
	if err != nil {
		s.Logger.Printf("Error rendering template: %v", err)
		http.Error(w, "Error rendering page", http.StatusInternalServerError)
	}
}

// scanHandler handles scan form submission
func (s *Server) scanHandler(w http.ResponseWriter, r *http.Request) {
	// Parse form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}
	
	// Get and validate domain
	domain := r.FormValue("domain")
	if domain == "" {
		http.Error(w, "Domain is required", http.StatusBadRequest)
		return
	}
	
	// Clean domain (remove http://, https://, etc.)
	domain = cleanDomain(domain)
	
	// Get other parameters
	wordlist := r.FormValue("wordlist")
	
	// Validate wordlist if provided
	if wordlist != "" {
		if !isValidWordlist(s.WordlistDir, wordlist) {
			http.Error(w, "Invalid wordlist", http.StatusBadRequest)
			return
		}
	}
	
	// Get threads parameter
	threads, err := strconv.Atoi(r.FormValue("threads"))
	if err != nil || threads < 1 {
		threads = 10 // Default value
	}
	
	// Cap threads to a reasonable limit
	if threads > 50 {
		threads = 50
	}
	
	// Get timeout parameter
	timeout, err := strconv.Atoi(r.FormValue("timeout"))
	if err != nil || timeout < 1 {
		timeout = 5 // Default value in seconds
	}
	
	// Cap timeout to a reasonable limit
	if timeout > 30 {
		timeout = 30
	}
	
	// Generate unique job ID
	jobID := s.GenerateJobID()
	
	// Create new job
	job := NewScanJob(
		jobID,
		domain,
		wordlist,
		threads,
		time.Duration(timeout)*time.Second,
	)
	
	// Add job to server
	s.AddJob(job)
	
	// Start job in a goroutine
	go job.Run(s.WordlistDir)
	
	// Redirect to job page
	http.Redirect(w, r, "/jobs/"+jobID, http.StatusSeeOther)
}

// jobHandler handles displaying job details
func (s *Server) jobHandler(w http.ResponseWriter, r *http.Request) {
	// Get job ID from URL
	vars := mux.Vars(r)
	jobID := vars["id"]
	
	// Get job
	job, exists := s.GetJob(jobID)
	if !exists {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}
	
	// Prepare template data
	data := struct {
		Job *ScanJob
	}{
		Job: job,
	}
	
	// Render template
	err := s.Templates.ExecuteTemplate(w, "job.html", data)
	if err != nil {
		s.Logger.Printf("Error rendering template: %v", err)
		http.Error(w, "Error rendering page", http.StatusInternalServerError)
	}
}

// csvHandler generates and serves a CSV file with job results
func (s *Server) csvHandler(w http.ResponseWriter, r *http.Request) {
	// Get job ID from URL
	vars := mux.Vars(r)
	jobID := vars["id"]
	
	// Get job
	job, exists := s.GetJob(jobID)
	if !exists {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}
	
	// Check if job has completed
	if job.Status != "Completed" {
		http.Error(w, "Job not completed", http.StatusBadRequest)
		return
	}
	
	// Set headers for CSV download
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-subdomains.csv", job.Domain))
	
	// Create CSV writer
	csvWriter := csv.NewWriter(w)
	
	// Write header
	csvWriter.Write([]string{
		"Subdomain", "IP Addresses", "Creation Date", "A Records", 
		"AAAA Records", "CNAME Records", "MX Records", 
		"TXT Records", "NS Records", "SOA Records",
	})
	
	// Write data rows
	for _, result := range job.Results {
		dnsRecords := result.DNSRecords
		
		csvWriter.Write([]string{
			result.Subdomain,
			strings.Join(result.IPAddresses, ", "),
			result.CreationDate,
			strings.Join(dnsRecords["A"], ", "),
			strings.Join(dnsRecords["AAAA"], ", "),
			strings.Join(dnsRecords["CNAME"], ", "),
			strings.Join(dnsRecords["MX"], ", "),
			strings.Join(dnsRecords["TXT"], ", "),
			strings.Join(dnsRecords["NS"], ", "),
			strings.Join(dnsRecords["SOA"], ", "),
		})
	}
	
	// Flush the writer
	csvWriter.Flush()
}

// Helper functions

// cleanDomain removes protocol prefixes and trailing slashes from domain
func cleanDomain(domain string) string {
	// Remove protocol prefixes
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "www.")
	
	// Remove path and query parts
	if i := strings.IndexAny(domain, "/?#"); i >= 0 {
		domain = domain[:i]
	}
	
	// Remove trailing slash
	domain = strings.TrimSuffix(domain, "/")
	
	return domain
}

// getAvailableWordlists returns a list of available wordlists
func getAvailableWordlists(wordlistDir string) ([]string, error) {
	// Create wordlist directory if it doesn't exist
	if _, err := os.Stat(wordlistDir); os.IsNotExist(err) {
		if err := os.MkdirAll(wordlistDir, 0755); err != nil {
			return nil, err
		}
	}
	
	// Read wordlist directory
	files, err := os.ReadDir(wordlistDir)
	if err != nil {
		return nil, err
	}
	
	// Filter for .txt files
	var wordlists []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".txt") {
			wordlists = append(wordlists, file.Name())
		}
	}
	
	return wordlists, nil
}

// isValidWordlist checks if a wordlist is valid
func isValidWordlist(wordlistDir, wordlist string) bool {
	// Check if the wordlist exists
	wordlistPath := filepath.Join(wordlistDir, wordlist)
	_, err := os.Stat(wordlistPath)
	return err == nil
}
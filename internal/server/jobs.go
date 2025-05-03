package server

import (
	"log"
	"path/filepath"
	"time"

	"github.com/username/dns-scanner/internal/scanner"
)

// ScanJob represents a DNS scanning job
type ScanJob struct {
	ID          string
	Domain      string
	Status      string // "Running", "Completed", "Failed"
	StartTime   time.Time
	EndTime     time.Time
	Results     []scanner.SubdomainResult
	WordlistPath string
	Threads     int
	Timeout     time.Duration
	Logger      *log.Logger
}

// NewScanJob creates a new scan job
func NewScanJob(id, domain, wordlist string, threads int, timeout time.Duration) *ScanJob {
	return &ScanJob{
		ID:          id,
		Domain:      domain,
		Status:      "Pending",
		StartTime:   time.Now(),
		WordlistPath: wordlist,
		Threads:     threads,
		Timeout:     timeout,
		Logger:      log.New(log.Writer(), "[JOB-"+id+"] ", log.LstdFlags),
	}
}

// Run executes the scanning job
func (j *ScanJob) Run(wordlistDir string) {
	j.Status = "Running"
	j.Logger.Printf("Starting scan for domain: %s", j.Domain)

	// Create full path to wordlist if provided
	var wordlistPath string
	if j.WordlistPath != "" {
		wordlistPath = filepath.Join(wordlistDir, j.WordlistPath)
	}

	// Create scanner instance
	scannerInst := scanner.NewScanner(
		j.Domain,
		j.Threads,
		j.Timeout,
		wordlistPath,
		j.Logger,
	)

	// Run the scan
	results, err := scannerInst.Scan()
	
	// Update job status and results
	if err != nil {
		j.Status = "Failed"
		j.Logger.Printf("Scan failed: %v", err)
	} else {
		j.Status = "Completed"
		j.Results = results
		j.Logger.Printf("Scan completed with %d results", len(results))
	}
	
	j.EndTime = time.Now()
}

// Duration returns the duration of the job
func (j *ScanJob) Duration() time.Duration {
	if j.Status == "Running" || j.Status == "Pending" {
		return time.Since(j.StartTime)
	}
	return j.EndTime.Sub(j.StartTime)
}

// HasResults returns true if the job has results
func (j *ScanJob) HasResults() bool {
	return j.Status == "Completed" && len(j.Results) > 0
}
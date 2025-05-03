// Package models contains the shared data structures used across the application
package models

import (
	"time"

	"github.com/google/uuid"
)

// ScanJob represents a DNS scanning job
type ScanJob struct {
	ID           uuid.UUID      `json:"id"`
	Domain       string         `json:"domain"`
	Status       string         `json:"status"` // "pending", "running", "completed", "failed"
	StartTime    time.Time      `json:"startTime"`
	EndTime      time.Time      `json:"endTime"`
	WordlistID   uuid.UUID      `json:"wordlistId,omitempty"`
	Threads      int            `json:"threads"`
	Timeout      time.Duration  `json:"timeout"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	ErrorMessage string         `json:"errorMessage,omitempty"`
	ResultCount  int            `json:"resultCount"`
	Results      []*SubdomainResult `json:"results,omitempty"`
}

// SubdomainResult represents a discovered subdomain and its information
type SubdomainResult struct {
	ID              uuid.UUID      `json:"id"`
	ScanJobID       uuid.UUID      `json:"scanJobId"`
	Subdomain       string         `json:"subdomain"`
	IPAddresses     []string       `json:"ipAddresses"`
	CreationDate    string         `json:"creationDate"`
	DiscoveryMethod string         `json:"discoveryMethod"` // "passive", "brute_force", "certificate", "virustotal"
	CreatedAt       time.Time      `json:"createdAt"`
	DNSRecords      []*DNSRecord   `json:"dnsRecords,omitempty"`
}

// DNSRecord represents a DNS record for a subdomain
type DNSRecord struct {
	ID                uuid.UUID    `json:"id"`
	SubdomainResultID uuid.UUID    `json:"subdomainResultId"`
	RecordType        string       `json:"recordType"` // "A", "AAAA", "CNAME", "MX", "TXT", "NS", "SOA"
	RecordValue       string       `json:"recordValue"`
	CreatedAt         time.Time    `json:"createdAt"`
}

// Wordlist represents a wordlist for subdomain scanning
type Wordlist struct {
	ID          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	Filename    string          `json:"filename"`
	Description string          `json:"description"`
	EntryCount  int             `json:"entryCount"`
	FileSize    int64           `json:"fileSize"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
	Source      string          `json:"source"` // "upload", "ai", "default"
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ScanStats represents statistics about a scan
type ScanStats struct {
	TotalSubdomains     int `json:"totalSubdomains"`
	PassiveSubdomains   int `json:"passiveSubdomains"`
	BruteForceSubdomains int `json:"bruteForceSubdomains"`
	WithIPv4            int `json:"withIPv4"`
	WithIPv6            int `json:"withIPv6"`
	WithCreationDate    int `json:"withCreationDate"`
	RecordTypeCounts    map[string]int `json:"recordTypeCounts"`
}

// ScanFilter represents filters for querying scan results
type ScanFilter struct {
	Domain          string    `json:"domain,omitempty"`
	SubdomainPattern string   `json:"subdomainPattern,omitempty"`
	HasIPAddress    *bool     `json:"hasIPAddress,omitempty"`
	RecordTypes     []string  `json:"recordTypes,omitempty"`
	DiscoveryMethod string    `json:"discoveryMethod,omitempty"`
	DateFrom        time.Time `json:"dateFrom,omitempty"`
	DateTo          time.Time `json:"dateTo,omitempty"`
	Limit           int       `json:"limit,omitempty"`
	Offset          int       `json:"offset,omitempty"`
}

// AIGenerationRequest represents a request to generate a wordlist using AI
type AIGenerationRequest struct {
	ID                uuid.UUID `json:"id"`
	WordlistID        uuid.UUID `json:"wordlistId"`
	CompanyName       string    `json:"companyName"`
	Industry          string    `json:"industry"`
	Products          string    `json:"products"`
	Technologies      string    `json:"technologies"`
	TargetDomain      string    `json:"targetDomain"`
	AdditionalContext string    `json:"additionalContext"`
	PromptUsed        string    `json:"promptUsed"`
	CreatedAt         time.Time `json:"createdAt"`
}

// Duration returns a human-readable duration between start and end times
func (j *ScanJob) Duration() string {
	var duration time.Duration
	
	if j.Status == "pending" || j.Status == "running" {
		duration = time.Since(j.StartTime)
	} else {
		duration = j.EndTime.Sub(j.StartTime)
	}
	
	// Format duration for human readability
	seconds := int(duration.Seconds())
	minutes := seconds / 60
	hours := minutes / 60
	seconds = seconds % 60
	minutes = minutes % 60
	
	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

// HasResults returns true if the job has completed with results
func (j *ScanJob) HasResults() bool {
	return j.Status == "completed" && j.ResultCount > 0
}

// SizeFormatted returns the file size in a human-readable format
func (w *Wordlist) SizeFormatted() string {
	size := float64(w.FileSize)
	
	if size < 1024 {
		return fmt.Sprintf("%.0f B", size)
	}
	
	size /= 1024
	if size < 1024 {
		return fmt.Sprintf("%.1f KB", size)
	}
	
	size /= 1024
	return fmt.Sprintf("%.1f MB", size)
}
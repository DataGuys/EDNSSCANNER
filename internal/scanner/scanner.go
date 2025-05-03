// Package scanner provides DNS subdomain scanning functionality
package scanner

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// SubdomainResult stores information about a discovered subdomain
type SubdomainResult struct {
	Subdomain    string
	IPAddresses  []string
	DNSRecords   map[string][]string
	CreationDate string
}

// Scanner represents a subdomain scanner
type Scanner struct {
	Domain       string
	Threads      int
	Timeout      time.Duration
	WordlistPath string
	Subdomains   map[string]bool
	Results      []SubdomainResult
	mu           sync.Mutex
	Logger       *log.Logger
}

// NewScanner creates a new subdomain scanner
func NewScanner(domain string, threads int, timeout time.Duration, wordlist string, logger *log.Logger) *Scanner {
	return &Scanner{
		Domain:       domain,
		Threads:      threads,
		Timeout:      timeout,
		WordlistPath: wordlist,
		Subdomains:   make(map[string]bool),
		Results:      []SubdomainResult{},
		Logger:       logger,
	}
}

// Scan performs the complete scanning process
func (s *Scanner) Scan() ([]SubdomainResult, error) {
	s.Logger.Printf("[+] Starting subdomain scan for %s\n", s.Domain)

	// Run passive techniques first (faster and stealthier)
	s.Logger.Println("[*] Running passive enumeration techniques...")
	if err := s.PassiveTechniques(); err != nil {
		s.Logger.Printf("[!] Error in passive techniques: %v\n", err)
	}

	// Then run active techniques if wordlist is provided
	if s.WordlistPath != "" {
		s.Logger.Printf("[*] Starting brute force using wordlist: %s\n", s.WordlistPath)
		if err := s.BruteForce(); err != nil {
			s.Logger.Printf("[!] Error in brute force: %v\n", err)
		}
	}

	// Get DNS information for discovered subdomains
	s.Logger.Printf("[*] Retrieving DNS information for %d subdomains...\n", len(s.Subdomains))
	s.GetDNSInfo()

	s.Logger.Printf("[+] Total unique subdomains discovered: %d\n", len(s.Results))
	return s.Results, nil
}

// PassiveTechniques performs passive subdomain enumeration
func (s *Scanner) PassiveTechniques() error {
	var wg sync.WaitGroup
	var errs []error

	// Search certificate transparency logs
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.Logger.Println("[*] Searching certificate transparency logs...")
		if err := s.CertSearch(); err != nil {
			s.mu.Lock()
			errs = append(errs, err)
			s.mu.Unlock()
		}
	}()

	// Search VirusTotal passive DNS
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.Logger.Println("[*] Searching VirusTotal passive DNS...")
		if err := s.VirusTotalSearch(); err != nil {
			s.mu.Lock()
			errs = append(errs, err)
			s.mu.Unlock()
		}
	}()

	wg.Wait()

	if len(errs) > 0 {
		return fmt.Errorf("errors occurred during passive techniques: %v", errs)
	}

	s.Logger.Printf("[+] Discovered %d subdomains through passive techniques\n", len(s.Subdomains))
	return nil
}

// GetDNSInfo gets DNS information for all discovered subdomains
func (s *Scanner) GetDNSInfo() {
	var wg sync.WaitGroup
	resultsChan := make(chan SubdomainResult)
	semaphore := make(chan struct{}, s.Threads)
	
	// Collect results in a separate goroutine
	go func() {
		for result := range resultsChan {
			s.mu.Lock()
			s.Results = append(s.Results, result)
			s.mu.Unlock()
		}
	}()

	// Process each subdomain
	for subdomain := range s.Subdomains {
		wg.Add(1)
		semaphore <- struct{}{}
		
		go func(sub string) {
			defer wg.Done()
			defer func() { <-semaphore }()
			
			fullDomain := fmt.Sprintf("%s.%s", sub, s.Domain)
			dnsRecords, err := s.GetDNSRecords(fullDomain)
			if err != nil {
				s.Logger.Printf("[!] Error getting DNS records for %s: %v\n", fullDomain, err)
				return
			}
			
			// Extract IP addresses from A records
			var ipAddresses []string
			if aRecords, ok := dnsRecords["A"]; ok {
				ipAddresses = aRecords
			}
			
			// Try to get creation date
			creationDate := s.GetCreationInfo(fullDomain)
			
			// Add to results
			resultsChan <- SubdomainResult{
				Subdomain:    fullDomain,
				IPAddresses:  ipAddresses,
				DNSRecords:   dnsRecords,
				CreationDate: creationDate,
			}
		}(subdomain)
	}
	
	wg.Wait()
	close(resultsChan)
}
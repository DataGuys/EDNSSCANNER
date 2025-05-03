package scanner

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// CertSearch searches certificate transparency logs for subdomains
func (s *Scanner) CertSearch() error {
	url := fmt.Sprintf("https://crt.sh/?q=%%.%s&output=json", s.Domain)
	
	client := &http.Client{Timeout: s.Timeout}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("error searching certificate transparency: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-200 response: %d", resp.StatusCode)
	}
	
	var entries []struct {
		NameValue string `json:"name_value"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return fmt.Errorf("error decoding JSON response: %w", err)
	}
	
	for _, entry := range entries {
		name := strings.ToLower(entry.NameValue)
		if strings.HasSuffix(name, "."+s.Domain) && !strings.Contains(name, "*") {
			subdomain := strings.TrimSuffix(name, "."+s.Domain)
			if subdomain != "" {
				s.mu.Lock()
				s.Subdomains[subdomain] = true
				s.mu.Unlock()
			}
		}
	}
	
	return nil
}

// VirusTotalSearch queries VirusTotal passive DNS database
func (s *Scanner) VirusTotalSearch() error {
	url := fmt.Sprintf("https://www.virustotal.com/ui/domains/%s/subdomains?limit=40", s.Domain)
	
	client := &http.Client{Timeout: s.Timeout}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("error creating VirusTotal request: %w", err)
	}
	
	// Add a User-Agent header to avoid being blocked
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error querying VirusTotal: %w", err)
	}
	defer resp.Body.Close()
	
	// If we get rate limited or another error, just log and continue
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-200 response from VirusTotal: %d", resp.StatusCode)
	}
	
	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("error decoding VirusTotal JSON response: %w", err)
	}
	
	for _, item := range result.Data {
		subdomain := strings.ToLower(item.ID)
		if strings.HasSuffix(subdomain, "."+s.Domain) {
			subName := strings.TrimSuffix(subdomain, "."+s.Domain)
			if subName != "" {
				s.mu.Lock()
				s.Subdomains[subName] = true
				s.mu.Unlock()
			}
		}
	}
	
	return nil
}
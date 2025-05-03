package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/username/dns-scanner/internal/repository"
)

// AIService handles AI-powered wordlist generation
type AIService struct {
	WordlistRepo   *repository.WordlistRepository
	WordlistDir    string
	APIKey         string
	APIEndpoint    string
	DefaultPrompt  string
	RequestTimeout time.Duration
	Model          string
}

// GenerationRequest represents a request to generate a wordlist
type GenerationRequest struct {
	CompanyName       string `json:"companyName"`
	Industry          string `json:"industry"`
	Products          string `json:"products"`
	Technologies      string `json:"technologies"`
	TargetDomain      string `json:"targetDomain"`
	AdditionalContext string `json:"additionalContext"`
	WordlistName      string `json:"wordlistName"`
}

// NewAIService creates a new AI service
func NewAIService(wordlistRepo *repository.WordlistRepository, wordlistDir string) *AIService {
	// Get API key from environment
	apiKey := os.Getenv("CLAUDE_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("AI_API_KEY")
	}

	// Default Claude API endpoint
	apiEndpoint := os.Getenv("CLAUDE_API_ENDPOINT")
	if apiEndpoint == "" {
		apiEndpoint = "https://api.anthropic.com/v1/messages"
	}

	// Default model
	model := os.Getenv("CLAUDE_MODEL")
	if model == "" {
		model = "claude-3-7-sonnet-20250219" // Latest Claude model as of May 2025
	}

	// Default prompt template
	defaultPrompt := `
You are a cybersecurity expert focusing on subdomain enumeration for ethical hacking and penetration testing.

Generate a list of likely subdomains for the target company based on the following information:
- Company Name: {{.CompanyName}}
- Industry/Sector: {{.Industry}}
- Products/Services: {{.Products}}
- Technologies Used: {{.Technologies}}
- Target Domain: {{.TargetDomain}}
- Additional Context: {{.AdditionalContext}}

Consider the following when generating subdomains:
1. Common subdomains (www, mail, api, etc.)
2. Product and service-related subdomains
3. Development and testing environments (dev, test, staging)
4. Internal tools and systems (admin, intranet, vpn)
5. Geographic locations (if the company has regional presence)
6. Acquired companies or brands
7. Technology stack-specific subdomains
8. Industry-specific systems and nomenclature

Provide ONLY the list of subdomains, one per line, without the domain suffix.
For example: "api" not "api.example.com"
Do not include any explanations or other text.
Be extremely thorough and creative with your generation.
`

	return &AIService{
		WordlistRepo:   wordlistRepo,
		WordlistDir:    wordlistDir,
		APIKey:         apiKey,
		APIEndpoint:    apiEndpoint,
		Model:          model,
		DefaultPrompt:  defaultPrompt,
		RequestTimeout: 60 * time.Second,
	}
}

// GenerateWordlist generates a wordlist using AI
func (s *AIService) GenerateWordlist(ctx context.Context, req GenerationRequest) (*repository.Wordlist, error) {
	// Check if API key is configured
	if s.APIKey == "" {
		return nil, fmt.Errorf("Claude API key not configured, set CLAUDE_API_KEY environment variable")
	}

	// Prepare the prompt by replacing placeholders
	prompt := s.DefaultPrompt
	prompt = strings.ReplaceAll(prompt, "{{.CompanyName}}", req.CompanyName)
	prompt = strings.ReplaceAll(prompt, "{{.Industry}}", req.Industry)
	prompt = strings.ReplaceAll(prompt, "{{.Products}}", req.Products)
	prompt = strings.ReplaceAll(prompt, "{{.Technologies}}", req.Technologies)
	prompt = strings.ReplaceAll(prompt, "{{.TargetDomain}}", req.TargetDomain)
	prompt = strings.ReplaceAll(prompt, "{{.AdditionalContext}}", req.AdditionalContext)

	// Build the request to Claude API
	type Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	type ClaudeRequest struct {
		Model       string  `json:"model"`
		Messages    []Message `json:"messages"`
		MaxTokens   int     `json:"max_tokens"`
		Temperature float64 `json:"temperature"`
		System      string  `json:"system,omitempty"`
	}

	claudeReq := ClaudeRequest{
		Model: s.Model,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   2000,
		Temperature: 0.7,
		System:      "You are a cybersecurity expert that helps generate subdomain wordlists for ethical hackers and security researchers.",
	}

	reqBody, err := json.Marshal(claudeReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Claude request: %w", err)
	}

	// Make the HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.APIEndpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", s.APIKey)
	httpReq.Header.Set("Anthropic-Version", "2023-06-01")

	client := &http.Client{
		Timeout: s.RequestTimeout,
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make Claude API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Claude API error (status %d): %s", resp.StatusCode, body)
	}

	// Parse the response
	type ClaudeResponse struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}

	var claudeResp ClaudeResponse
	err = json.NewDecoder(resp.Body).Decode(&claudeResp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode Claude response: %w", err)
	}

	if len(claudeResp.Content) == 0 {
		return nil, fmt.Errorf("Claude API returned no content")
	}

	// Extract and clean the generated wordlist
	wordlistContent := claudeResp.Content[0].Text
	lines := strings.Split(wordlistContent, "\n")
	var cleanedLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Remove domain suffix if present
		if strings.Contains(line, ".") {
			parts := strings.Split(line, ".")
			if len(parts) > 1 && strings.Contains(req.TargetDomain, parts[len(parts)-1]) {
				line = strings.Join(parts[:len(parts)-1], ".")
			}
		}
		cleanedLines = append(cleanedLines, line)
	}

	// Create metadata for the wordlist
	metadata := map[string]interface{}{
		"companyName":       req.CompanyName,
		"industry":          req.Industry,
		"products":          req.Products,
		"technologies":      req.Technologies,
		"targetDomain":      req.TargetDomain,
		"additionalContext": req.AdditionalContext,
		"promptUsed":        prompt,
		"generatedAt":       time.Now().Format(time.RFC3339),
		"model":             s.Model,
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Create a file with the generated wordlist
	wordlistID := uuid.New()
	filename := fmt.Sprintf("ai_%s.txt", wordlistID.String())
	filePath := filepath.Join(s.WordlistDir, filename)

	// Add header with generation info
	header := fmt.Sprintf(`# AI-generated wordlist for %s (%s)
# Generated: %s
# Target Domain: %s
# Entries: %d
# AI Model: %s
#
# This wordlist was automatically generated using Claude AI based on:
# - Company: %s
# - Industry: %s
# - Products: %s
# - Technologies: %s
#
# For ethical hacking and security research purposes only.
#
`,
		req.CompanyName, 
		req.TargetDomain,
		time.Now().Format(time.RFC3339),
		req.TargetDomain,
		len(cleanedLines),
		s.Model,
		req.CompanyName,
		req.Industry,
		req.Products,
		req.Technologies,
	)

	fileContent := header + strings.Join(cleanedLines, "\n")
	err = ioutil.WriteFile(filePath, []byte(fileContent), 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write wordlist file: %w", err)
	}

	// Create wordlist record in database
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	wordlist := &repository.Wordlist{
		ID:          wordlistID,
		Name:        req.WordlistName,
		Filename:    filename,
		Description: fmt.Sprintf("AI-generated wordlist for %s using Claude", req.CompanyName),
		EntryCount:  len(cleanedLines),
		FileSize:    fileInfo.Size(),
		Source:      "ai",
		Metadata:    metadataJSON,
	}

	err = s.WordlistRepo.Create(ctx, wordlist)
	if err != nil {
		// Cleanup file on failure
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to create wordlist in database: %w", err)
	}

	return wordlist, nil
}

// IsConfigured returns true if the AI service is properly configured
func (s *AIService) IsConfigured() bool {
	return s.APIKey != ""
}
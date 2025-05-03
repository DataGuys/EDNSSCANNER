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
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("AI_API_KEY")
	}

	// Default OpenAI API endpoint
	apiEndpoint := os.Getenv("AI_API_ENDPOINT")
	if apiEndpoint == "" {
		apiEndpoint = "https://api.openai.com/v1/chat/completions"
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
		DefaultPrompt:  defaultPrompt,
		RequestTimeout: 60 * time.Second,
	}
}

// GenerateWordlist generates a wordlist using AI
func (s *AIService) GenerateWordlist(ctx context.Context, req GenerationRequest) (*repository.Wordlist, error) {
	// Check if API key is configured
	if s.APIKey == "" {
		return nil, fmt.Errorf("AI API key not configured, set OPENAI_API_KEY environment variable")
	}

	// Prepare the prompt by replacing placeholders
	prompt := s.DefaultPrompt
	prompt = strings.ReplaceAll(prompt, "{{.CompanyName}}", req.CompanyName)
	prompt = strings.ReplaceAll(prompt, "{{.Industry}}", req.Industry)
	prompt = strings.ReplaceAll(prompt, "{{.Products}}", req.Products)
	prompt = strings.ReplaceAll(prompt, "{{.Technologies}}", req.Technologies)
	prompt = strings.ReplaceAll(prompt, "{{.TargetDomain}}", req.TargetDomain)
	prompt = strings.ReplaceAll(prompt, "{{.AdditionalContext}}", req.AdditionalContext)

	// Build the request to OpenAI API
	type Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	type OpenAIRequest struct {
		Model       string    `json:"model"`
		Messages    []Message `json:"messages"`
		Temperature float64   `json:"temperature"`
		MaxTokens   int       `json:"max_tokens"`
	}

	openAIReq := OpenAIRequest{
		Model: "gpt-4", // Or use gpt-3.5-turbo for a more affordable option
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are a cybersecurity expert that helps generate subdomain wordlists for ethical hackers and security researchers.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.7,
		MaxTokens:   2000,
	}

	reqBody, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OpenAI request: %w", err)
	}

	// Make the HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.APIEndpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.APIKey)

	client := &http.Client{
		Timeout: s.RequestTimeout,
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make OpenAI API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error (status %d): %s", resp.StatusCode, body)
	}

	// Parse the response
	type OpenAIResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	var openAIResp OpenAIResponse
	err = json.NewDecoder(resp.Body).Decode(&openAIResp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode OpenAI response: %w", err)
	}

	if len(openAIResp.Choices) == 0 {
		return nil, fmt.Errorf("OpenAI API returned no choices")
	}

	// Extract and clean the generated wordlist
	wordlistContent := openAIResp.Choices[0].Message.Content
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
		"model":             "gpt-4", // Or whatever model was used
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
#
# This wordlist was automatically generated using AI based on:
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
		Description: fmt.Sprintf("AI-generated wordlist for %s", req.CompanyName),
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
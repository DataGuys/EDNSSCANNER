package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// wordlistsHandler handles GET requests to the wordlists page
func (s *Server) wordlistsHandler(w http.ResponseWriter, r *http.Request) {
	// Get all wordlists
	ctx := r.Context()
	wordlists, err := s.Dependencies.WordlistRepo.GetAll(ctx)
	if err != nil {
		s.Logger.Printf("Error getting wordlists: %v", err)
		http.Error(w, "Error getting wordlists", http.StatusInternalServerError)
		return
	}

	// Prepare template data
	data := struct {
		Wordlists []*repository.Wordlist
		AIEnabled bool
	}{
		Wordlists: wordlists,
		AIEnabled: s.Dependencies.AIService.IsConfigured(),
	}

	// Render template
	err = s.Templates.ExecuteTemplate(w, "wordlists.html", data)
	if err != nil {
		s.Logger.Printf("Error rendering template: %v", err)
		http.Error(w, "Error rendering page", http.StatusInternalServerError)
	}
}

// wordlistUploadHandler handles POST requests to upload a wordlist
func (s *Server) wordlistUploadHandler(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form
	err := r.ParseMultipartForm(10 << 20) // 10 MB max
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get form values
	wordlistName := r.FormValue("wordlistName")
	if wordlistName == "" {
		http.Error(w, "Wordlist name is required", http.StatusBadRequest)
		return
	}

	// Get uploaded file
	file, header, err := r.FormFile("wordlistFile")
	if err != nil {
		http.Error(w, "Failed to get uploaded file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".txt") {
		http.Error(w, "Only .txt files are allowed", http.StatusBadRequest)
		return
	}

	// Save to temporary file
	tempFile, err := os.CreateTemp("", "upload-*.txt")
	if err != nil {
		s.Logger.Printf("Failed to create temp file: %v", err)
		http.Error(w, "Failed to save uploaded file", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	_, err = io.Copy(tempFile, file)
	if err != nil {
		s.Logger.Printf("Failed to copy uploaded file: %v", err)
		http.Error(w, "Failed to save uploaded file", http.StatusInternalServerError)
		return
	}

	// Create wordlist in database
	ctx := r.Context()
	metadata := map[string]interface{}{
		"originalFilename": header.Filename,
		"uploadedBy":       "web",
		"contentType":      header.Header.Get("Content-Type"),
	}
	metadataJSON, _ := json.Marshal(metadata)

	wordlist, err := s.Dependencies.WordlistRepo.CreateFromFile(
		ctx,
		wordlistName,
		tempFile.Name(),
		"upload",
		json.RawMessage(metadataJSON),
	)
	if err != nil {
		s.Logger.Printf("Failed to create wordlist: %v", err)
		http.Error(w, "Failed to create wordlist", http.StatusInternalServerError)
		return
	}

	// Redirect to wordlists page
	http.Redirect(w, r, "/wordlists", http.StatusSeeOther)
}

// wordlistGenerateHandler handles POST requests to generate a wordlist with AI
func (s *Server) wordlistGenerateHandler(w http.ResponseWriter, r *http.Request) {
	// Check if AI service is configured
	if !s.Dependencies.AIService.IsConfigured() {
		http.Error(w, "AI service is not configured", http.StatusServiceUnavailable)
		return
	}

	// Parse form
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get form values
	req := ai.GenerationRequest{
		CompanyName:       r.FormValue("companyName"),
		Industry:          r.FormValue("industry"),
		Products:          r.FormValue("products"),
		Technologies:      r.FormValue("technologies"),
		TargetDomain:      r.FormValue("targetDomain"),
		AdditionalContext: r.FormValue("additionalContext"),
		WordlistName:      r.FormValue("wordlistName"),
	}

	// Validate required fields
	if req.CompanyName == "" || req.TargetDomain == "" || req.WordlistName == "" {
		http.Error(w, "Company name, target domain, and wordlist name are required", http.StatusBadRequest)
		return
	}

	// Generate wordlist
	ctx := r.Context()
	_, err = s.Dependencies.AIService.GenerateWordlist(ctx, req)
	if err != nil {
		s.Logger.Printf("Failed to generate wordlist: %v", err)
		http.Error(w, "Failed to generate wordlist: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Redirect to wordlists page
	http.Redirect(w, r, "/wordlists", http.StatusSeeOther)
}

// wordlistViewHandler handles GET requests to view a wordlist
func (s *Server) wordlistViewHandler(w http.ResponseWriter, r *http.Request) {
	// Get wordlist ID from URL
	vars := mux.Vars(r)
	idStr := vars["id"]
	
	// Parse UUID
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid wordlist ID", http.StatusBadRequest)
		return
	}

	// Get wordlist
	ctx := r.Context()
	wordlist, err := s.Dependencies.WordlistRepo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "Wordlist not found", http.StatusNotFound)
		return
	}

	// Read wordlist file
	filePath := filepath.Join(s.WordlistDir, wordlist.Filename)
	content, err := os.ReadFile(filePath)
	if err != nil {
		s.Logger.Printf("Failed to read wordlist file: %v", err)
		http.Error(w, "Failed to read wordlist", http.StatusInternalServerError)
		return
	}

	// Prepare template data
	data := struct {
		Wordlist *repository.Wordlist
		Content  string
	}{
		Wordlist: wordlist,
		Content:  string(content),
	}

	// Render template
	err = s.Templates.ExecuteTemplate(w, "wordlist_view.html", data)
	if err != nil {
		s.Logger.Printf("Error rendering template: %v", err)
		http.Error(w, "Error rendering page", http.StatusInternalServerError)
	}
}

// wordlistDownloadHandler handles GET requests to download a wordlist
func (s *Server) wordlistDownloadHandler(w http.ResponseWriter, r *http.Request) {
	// Get wordlist ID from URL
	vars := mux.Vars(r)
	idStr := vars["id"]
	
	// Parse UUID
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid wordlist ID", http.StatusBadRequest)
		return
	}

	// Get wordlist
	ctx := r.Context()
	wordlist, err := s.Dependencies.WordlistRepo.GetByID(ctx, id)
	if err != nil {
		http.Error(w, "Wordlist not found", http.StatusNotFound)
		return
	}

	// Read wordlist file
	filePath := filepath.Join(s.WordlistDir, wordlist.Filename)
	content, err := os.ReadFile(filePath)
	if err != nil {
		s.Logger.Printf("Failed to read wordlist file: %v", err)
		http.Error(w, "Failed to read wordlist", http.StatusInternalServerError)
		return
	}

	// Set content disposition and type
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.txt", wordlist.Name))
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))

	// Write content
	_, err = w.Write(content)
	if err != nil {
		s.Logger.Printf("Failed to write content: %v", err)
	}
}

// wordlistDeleteHandler handles POST requests to delete a wordlist
func (s *Server) wordlistDeleteHandler(w http.ResponseWriter, r *http.Request) {
	// Parse form
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get wordlist ID
	idStr := r.FormValue("id")
	if idStr == "" {
		http.Error(w, "Wordlist ID is required", http.StatusBadRequest)
		return
	}

	// Parse UUID
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid wordlist ID", http.StatusBadRequest)
		return
	}

	// Delete wordlist
	ctx := r.Context()
	err = s.Dependencies.WordlistRepo.Delete(ctx, id)
	if err != nil {
		s.Logger.Printf("Failed to delete wordlist: %v", err)
		http.Error(w, "Failed to delete wordlist", http.StatusInternalServerError)
		return
	}

	// Redirect to wordlists page
	http.Redirect(w, r, "/wordlists", http.StatusSeeOther)
}
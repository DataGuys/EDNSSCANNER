package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	"github.com/username/dns-scanner/internal/database"
)

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
	Metadata    json.RawMessage `json:"metadata"`
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

// WordlistRepository handles wordlist database operations
type WordlistRepository struct {
	DB          *database.Database
	WordlistDir string
}

// NewWordlistRepository creates a new wordlist repository
func NewWordlistRepository(db *database.Database, wordlistDir string) *WordlistRepository {
	return &WordlistRepository{
		DB:          db,
		WordlistDir: wordlistDir,
	}
}

// Create adds a new wordlist to the database
func (r *WordlistRepository) Create(ctx context.Context, wordlist *Wordlist) error {
	query := `
	INSERT INTO wordlists 
	    (id, name, filename, description, entry_count, file_size, source, metadata) 
	VALUES 
	    ($1, $2, $3, $4, $5, $6, $7, $8)
	RETURNING id, created_at, updated_at
	`

	// Generate UUID if not provided
	if wordlist.ID == uuid.Nil {
		wordlist.ID = uuid.New()
	}

	row := r.DB.Pool.QueryRow(
		ctx, 
		query,
		wordlist.ID,
		wordlist.Name,
		wordlist.Filename,
		wordlist.Description,
		wordlist.EntryCount,
		wordlist.FileSize,
		wordlist.Source,
		wordlist.Metadata,
	)

	err := row.Scan(&wordlist.ID, &wordlist.CreatedAt, &wordlist.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create wordlist: %w", err)
	}

	return nil
}

// GetByID retrieves a wordlist by ID
func (r *WordlistRepository) GetByID(ctx context.Context, id uuid.UUID) (*Wordlist, error) {
	query := `
	SELECT 
		id, name, filename, description, entry_count, file_size, 
		created_at, updated_at, source, metadata
	FROM wordlists
	WHERE id = $1
	`

	row := r.DB.Pool.QueryRow(ctx, query, id)
	
	var wordlist Wordlist
	err := row.Scan(
		&wordlist.ID,
		&wordlist.Name,
		&wordlist.Filename,
		&wordlist.Description,
		&wordlist.EntryCount,
		&wordlist.FileSize,
		&wordlist.CreatedAt,
		&wordlist.UpdatedAt,
		&wordlist.Source,
		&wordlist.Metadata,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("wordlist not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get wordlist: %w", err)
	}

	return &wordlist, nil
}

// GetAll retrieves all wordlists
func (r *WordlistRepository) GetAll(ctx context.Context) ([]*Wordlist, error) {
	query := `
	SELECT 
		id, name, filename, description, entry_count, file_size, 
		created_at, updated_at, source, metadata
	FROM wordlists
	ORDER BY created_at DESC
	`

	rows, err := r.DB.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query wordlists: %w", err)
	}
	defer rows.Close()

	var wordlists []*Wordlist
	for rows.Next() {
		var wordlist Wordlist
		err := rows.Scan(
			&wordlist.ID,
			&wordlist.Name,
			&wordlist.Filename,
			&wordlist.Description,
			&wordlist.EntryCount,
			&wordlist.FileSize,
			&wordlist.CreatedAt,
			&wordlist.UpdatedAt,
			&wordlist.Source,
			&wordlist.Metadata,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan wordlist row: %w", err)
		}
		wordlists = append(wordlists, &wordlist)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("error iterating wordlist rows: %w", rows.Err())
	}

	return wordlists, nil
}

// Delete removes a wordlist by ID
func (r *WordlistRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// First, get the wordlist to find the filename
	wordlist, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Begin transaction
	tx, err := r.DB.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Delete from database
	query := `DELETE FROM wordlists WHERE id = $1`
	_, err = tx.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete wordlist from database: %w", err)
	}

	// Delete the file
	filePath := filepath.Join(r.WordlistDir, wordlist.Filename)
	err = os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete wordlist file: %w", err)
	}

	// Commit transaction
	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// CreateFromFile creates a new wordlist from a file
func (r *WordlistRepository) CreateFromFile(ctx context.Context, name, sourcePath, source string, metadata json.RawMessage) (*Wordlist, error) {
	// Generate a unique filename
	filename := fmt.Sprintf("%s.txt", uuid.New().String())
	destPath := filepath.Join(r.WordlistDir, filename)

	// Copy the file
	content, err := ioutil.ReadFile(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read source file: %w", err)
	}

	err = ioutil.WriteFile(destPath, content, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write destination file: %w", err)
	}

	// Count entries (non-empty, non-comment lines)
	var entryCount int
	lines := 0
	for _, line := range content {
		if line == '\n' {
			lines++
		}
	}
	entryCount = lines + 1 // Add 1 for the last line if it doesn't end with newline

	// Create wordlist record
	fileInfo, err := os.Stat(destPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	wordlist := &Wordlist{
		ID:          uuid.New(),
		Name:        name,
		Filename:    filename,
		Description: "",
		EntryCount:  entryCount,
		FileSize:    fileInfo.Size(),
		Source:      source,
		Metadata:    metadata,
	}

	err = r.Create(ctx, wordlist)
	if err != nil {
		// Cleanup file on failure
		os.Remove(destPath)
		return nil, err
	}

	return wordlist, nil
}

// GetContent retrieves the content of a wordlist
func (r *WordlistRepository) GetContent(ctx context.Context, id uuid.UUID) ([]string, error) {
	wordlist, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	filePath := filepath.Join(r.WordlistDir, wordlist.Filename)
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read wordlist file: %w", err)
	}

	// Split content into lines and filter empty lines and comments
	var lines []string
	for _, line := range content {
		line := string(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
	}

	return lines, nil
}
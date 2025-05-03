package server

import (
	"log"
	"os"

	"github.com/username/dns-scanner/internal/ai"
	"github.com/username/dns-scanner/internal/database"
	"github.com/username/dns-scanner/internal/repository"
)

// Config represents the server configuration
type Config struct {
	StaticDir   string
	TemplateDir string
	WordlistDir string
	Port        int
	DB          database.Config
}

// Dependencies contains all the service dependencies
type Dependencies struct {
	DB              *database.Database
	WordlistRepo    *repository.WordlistRepository
	ScanJobRepo     *repository.ScanJobRepository // We'll implement this later
	AIService       *ai.AIService
	Logger          *log.Logger
}

// DefaultConfig returns a default configuration
func DefaultConfig() Config {
	return Config{
		StaticDir:   "./static",
		TemplateDir: "./templates",
		WordlistDir: "./wordlists",
		Port:        8080,
		DB: database.Config{
			Host:     getEnvWithDefault("DB_HOST", "localhost"),
			Port:     getEnvAsIntWithDefault("DB_PORT", 5432),
			User:     getEnvWithDefault("DB_USER", "postgres"),
			Password: getEnvWithDefault("DB_PASSWORD", "postgres"),
			DBName:   getEnvWithDefault("DB_NAME", "dnsscanner"),
			SSLMode:  getEnvWithDefault("DB_SSLMODE", "disable"),
		},
	}
}

// InitializeDependencies initializes all dependencies
func InitializeDependencies(config Config) (*Dependencies, error) {
	logger := log.New(os.Stdout, "[SERVER] ", log.LstdFlags)

	// Initialize database
	db, err := database.New(config.DB)
	if err != nil {
		return nil, err
	}

	// Run migrations
	if err := db.RunMigrations(); err != nil {
		return nil, err
	}

	// Initialize repositories
	wordlistRepo := repository.NewWordlistRepository(db, config.WordlistDir)
	scanJobRepo := repository.NewScanJobRepository(db) // We'll implement this later

	// Initialize AI service
	aiService := ai.NewAIService(wordlistRepo, config.WordlistDir)

	return &Dependencies{
		DB:              db,
		WordlistRepo:    wordlistRepo,
		ScanJobRepo:     scanJobRepo,
		AIService:       aiService,
		Logger:          logger,
	}, nil
}

// Helper functions

// getEnvWithDefault gets an environment variable with a default value
func getEnvWithDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvAsIntWithDefault gets an environment variable as an integer with a default value
func getEnvAsIntWithDefault(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return intValue
}
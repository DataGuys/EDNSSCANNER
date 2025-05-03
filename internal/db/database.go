package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

// Database represents a connection to the database
type Database struct {
	Pool *pgxpool.Pool
}

// Config contains the database configuration
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// New creates a new database connection
func New(config Config) (*Database, error) {
	// Build connection string
	connString := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		config.User, config.Password, config.Host, config.Port, config.DBName, config.SSLMode,
	)

	// Create connection pool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.Connect(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Database{Pool: pool}, nil
}

// Close closes the database connection
func (db *Database) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}

// RunMigrations runs database migrations
func (db *Database) RunMigrations() error {
	// Here we would normally use a migration library
	// For simplicity, we'll just execute our schema.sql manually
	// In a production app, consider using golang-migrate or similar

	// Read schema.sql file (implementation not shown)
	schema := `
	-- Enable UUID extension
	CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

	-- Wordlists table
	CREATE TABLE IF NOT EXISTS wordlists (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		name VARCHAR(255) NOT NULL,
		filename VARCHAR(255) NOT NULL,
		description TEXT,
		entry_count INTEGER NOT NULL DEFAULT 0,
		file_size INTEGER NOT NULL DEFAULT 0,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		source VARCHAR(50) NOT NULL DEFAULT 'upload',
		metadata JSONB
	);

	-- Scan jobs table
	CREATE TABLE IF NOT EXISTS scan_jobs (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		domain VARCHAR(255) NOT NULL,
		status VARCHAR(50) NOT NULL DEFAULT 'pending',
		start_time TIMESTAMP WITH TIME ZONE,
		end_time TIMESTAMP WITH TIME ZONE,
		wordlist_id UUID REFERENCES wordlists(id) ON DELETE SET NULL,
		threads INTEGER NOT NULL DEFAULT 10,
		timeout INTEGER NOT NULL DEFAULT 5,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		error_message TEXT,
		result_count INTEGER NOT NULL DEFAULT 0,
		configuration JSONB
	);

	-- Subdomain results table
	CREATE TABLE IF NOT EXISTS subdomain_results (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		scan_job_id UUID NOT NULL REFERENCES scan_jobs(id) ON DELETE CASCADE,
		subdomain VARCHAR(255) NOT NULL,
		ip_addresses TEXT[],
		creation_date VARCHAR(255),
		discovery_method VARCHAR(50) NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		UNIQUE(scan_job_id, subdomain)
	);

	-- DNS records table
	CREATE TABLE IF NOT EXISTS dns_records (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		subdomain_result_id UUID NOT NULL REFERENCES subdomain_results(id) ON DELETE CASCADE,
		record_type VARCHAR(10) NOT NULL,
		record_value TEXT NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
	);

	-- AI generation requests table
	CREATE TABLE IF NOT EXISTS ai_generation_requests (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		wordlist_id UUID NOT NULL REFERENCES wordlists(id) ON DELETE CASCADE,
		company_name VARCHAR(255) NOT NULL,
		industry VARCHAR(255),
		products TEXT,
		technologies TEXT,
		target_domain VARCHAR(255) NOT NULL,
		additional_context TEXT,
		prompt_used TEXT,
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
	);

	-- Create indexes
	CREATE INDEX IF NOT EXISTS idx_scan_jobs_domain ON scan_jobs(domain);
	CREATE INDEX IF NOT EXISTS idx_scan_jobs_status ON scan_jobs(status);
	CREATE INDEX IF NOT EXISTS idx_subdomain_results_scan_job_id ON subdomain_results(scan_job_id);
	CREATE INDEX IF NOT EXISTS idx_dns_records_subdomain_result_id ON dns_records(subdomain_result_id);
	CREATE INDEX IF NOT EXISTS idx_dns_records_type ON dns_records(record_type);
	CREATE INDEX IF NOT EXISTS idx_wordlists_source ON wordlists(source);
	`

	// Execute the schema
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := db.Pool.Exec(ctx, schema)
	return err
}
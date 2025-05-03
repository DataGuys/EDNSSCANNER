-- DNS Scanner Database Schema

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Wordlists table
CREATE TABLE wordlists (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    filename VARCHAR(255) NOT NULL,
    description TEXT,
    entry_count INTEGER NOT NULL DEFAULT 0,
    file_size INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    source VARCHAR(50) NOT NULL DEFAULT 'upload', -- 'upload', 'ai', 'default'
    metadata JSONB -- Store additional metadata about the wordlist
);

-- Scan jobs table
CREATE TABLE scan_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    domain VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- 'pending', 'running', 'completed', 'failed'
    start_time TIMESTAMP WITH TIME ZONE,
    end_time TIMESTAMP WITH TIME ZONE,
    wordlist_id UUID REFERENCES wordlists(id) ON DELETE SET NULL,
    threads INTEGER NOT NULL DEFAULT 10,
    timeout INTEGER NOT NULL DEFAULT 5,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    error_message TEXT,
    result_count INTEGER NOT NULL DEFAULT 0,
    configuration JSONB -- Store additional configuration
);

-- Subdomain results table
CREATE TABLE subdomain_results (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scan_job_id UUID NOT NULL REFERENCES scan_jobs(id) ON DELETE CASCADE,
    subdomain VARCHAR(255) NOT NULL,
    ip_addresses TEXT[], -- Array of IP addresses
    creation_date VARCHAR(255),
    discovery_method VARCHAR(50) NOT NULL, -- 'passive', 'brute_force', 'certificate', 'virustotal'
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(scan_job_id, subdomain)
);

-- DNS records table
CREATE TABLE dns_records (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    subdomain_result_id UUID NOT NULL REFERENCES subdomain_results(id) ON DELETE CASCADE,
    record_type VARCHAR(10) NOT NULL, -- 'A', 'AAAA', 'CNAME', 'MX', 'TXT', 'NS', 'SOA'
    record_value TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- AI generation requests table
CREATE TABLE ai_generation_requests (
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
CREATE INDEX idx_scan_jobs_domain ON scan_jobs(domain);
CREATE INDEX idx_scan_jobs_status ON scan_jobs(status);
CREATE INDEX idx_subdomain_results_scan_job_id ON subdomain_results(scan_job_id);
CREATE INDEX idx_dns_records_subdomain_result_id ON dns_records(subdomain_result_id);
CREATE INDEX idx_dns_records_type ON dns_records(record_type);
CREATE INDEX idx_wordlists_source ON wordlists(source);
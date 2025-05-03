package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/username/dns-scanner/internal/server"
)

// Default paths
const (
	defaultStaticDir   = "./static"
	defaultTemplateDir = "./templates"
	defaultWordlistDir = "./wordlists"
)

func main() {
	// Parse command line flags
	port := flag.Int("port", 8080, "Port to listen on")
	staticDir := flag.String("static", defaultStaticDir, "Path to static files directory")
	templateDir := flag.String("templates", defaultTemplateDir, "Path to template files directory")
	wordlistDir := flag.String("wordlists", defaultWordlistDir, "Path to wordlist files directory")
	flag.Parse()

	// Ensure directories exist
	ensureDir(*staticDir)
	ensureDir(*templateDir)
	ensureDir(*wordlistDir)

	// Create server
	srv := server.NewServer(*staticDir, *templateDir, *wordlistDir)

	// Start server
	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Starting DNS Scanner Web Interface on http://localhost%s", addr)
	log.Fatal(srv.Start(addr))
}

// ensureDir makes sure a directory exists
func ensureDir(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		log.Printf("Creating directory: %s", dir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create example wordlist if wordlist directory is empty
	if filepath.Base(dir) == "wordlists" {
		files, err := os.ReadDir(dir)
		if err != nil {
			log.Fatalf("Failed to read wordlist directory: %v", err)
		}

		if len(files) == 0 {
			createExampleWordlist(dir)
		}
	}
}

// createExampleWordlist creates a simple wordlist in the wordlist directory
func createExampleWordlist(dir string) {
	commonWordlist := filepath.Join(dir, "common.txt")
	log.Printf("Creating example wordlist: %s", commonWordlist)

	// Common subdomains
	commonSubdomains := []string{
		"# Common subdomains",
		"www",
		"mail",
		"ftp",
		"admin",
		"blog",
		"test",
		"dev",
		"api",
		"secure",
		"shop",
		"store",
		"webmail",
		"portal",
		"support",
		"vpn",
		"m",
		"mobile",
		"app",
		"staging",
		"media",
		"images",
		"files",
		"docs",
		"beta",
		"demo",
	}

	// Write to file
	file, err := os.Create(commonWordlist)
	if err != nil {
		log.Printf("Failed to create example wordlist: %v", err)
		return
	}
	defer file.Close()

	for _, line := range commonSubdomains {
		fmt.Fprintln(file, line)
	}
}
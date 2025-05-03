# DNS Subdomain Scanner

A powerful DNS subdomain scanner with a web interface, database integration, and AI-powered wordlist generation, packaged in a Docker container. This Go-based solution combines both passive and active enumeration techniques to discover subdomains, retrieve DNS information, and identify creation dates.

![DNS Subdomain Scanner](https://your-screenshot-url-here.png)

## Features

- **Multiple Discovery Techniques**:
  - **Passive**: Certificate Transparency logs, VirusTotal passive DNS
  - **Active**: DNS brute force with customizable wordlists
- **Comprehensive Information Retrieval**:
  - DNS Records (A, AAAA, CNAME, MX, TXT, NS, SOA)
  - IP Addresses
  - Creation Dates (via WHOIS)
- **Advanced Wordlist Management**:
  - Upload custom wordlists
  - AI-powered wordlist generation based on company info
  - Wordlist browsing and searching
- **Persistent Storage**:
  - PostgreSQL database for scan results
  - Secure data storage and retrieval
  - Comprehensive reporting
- **Modern Web Interface**:
  - Clean, responsive UI
  - Real-time scan status updates
  - Sortable and filterable results
  - CSV export for further analysis
- **Efficient Performance**:
  - Written in Go for high performance
  - Concurrent scanning
  - Configurable threads and timeouts
- **Easy Deployment**:
  - Docker Compose for easy setup
  - Simple one-liner deployment

## Quick Deployment

Deploy the DNS Subdomain Scanner with this one-liner:

```bash
curl -s https://raw.githubusercontent.com/DataGuys/EDNSSCANNER/refs/heads/main/deploy.sh | bash
```

This script will:
1. Clone the repository
2. Build the Docker containers (web app and PostgreSQL)
3. Start the services
4. Open the web interface in your default browser

## Requirements

- Docker
- Docker Compose
- Git (for cloning the repository)
- Bash (for the deployment script)
- OpenAI API key (optional, for AI wordlist generation)

## Manual Installation

If you prefer to install manually:

```bash
# Clone the repository
git clone https://github.com/username/dns-scanner.git

# Navigate to the project directory
cd dns-scanner

# Build and start the Docker containers
docker-compose up -d

# Access the web interface
echo "DNS Scanner available at: http://localhost:8080"
```

## Usage

### Basic Scanning

1. Access the web interface at http://localhost:8080
2. Enter a domain to scan (e.g., example.com)
3. Optionally select a wordlist for brute force scanning
4. Configure threads and timeout settings
5. Start the scan
6. View results in the web interface or download as CSV

### Managing Wordlists

1. Go to the Wordlists page
2. Upload custom wordlists or use the provided default wordlists
3. Browse and search existing wordlists
4. Download wordlists for use in other tools

### AI-Powered Wordlist Generation

1. Go to the Wordlists page
2. Scroll to the "AI-Powered Wordlist Generator" section
3. Enter company information:
   - Company name
   - Industry/sector
   - Products/services
   - Technologies used
   - Target domain
   - Additional context
4. Generate a custom wordlist based on this information
5. Use the generated wordlist for scanning

## Configuration

### Environment Variables

The application uses the following environment variables:

```
# Database configuration
DB_HOST=postgres
DB_PORT=5432
DB_USER=dnsscanner
DB_PASSWORD=securepassword
DB_NAME=dnsscanner
DB_SSLMODE=disable

# AI integration (optional)
OPENAI_API_KEY=your_api_key_here
```

### Docker Compose

Modify `docker-compose.yml` to:
- Change the exposed port
- Set different database credentials
- Add your OpenAI API key
- Mount custom directories

### OpenAI API Key

To use the AI-powered wordlist generation feature, you need to provide an OpenAI API key:

1. Get an API key from [OpenAI](https://platform.openai.com/)
2. Add it to the `docker-compose.yml` file:
   ```yaml
   environment:
     - OPENAI_API_KEY=your_api_key_here
   ```
   Or set it as an environment variable before running the container.

## Advanced Configuration

### Custom PostgreSQL Configuration

You can customize the PostgreSQL configuration by adding a custom `postgresql.conf` file:

```yaml
volumes:
  - ./postgresql.conf:/etc/postgresql/postgresql.conf
command: ["postgres", "-c", "config_file=/etc/postgresql/postgresql.conf"]
```

### Security Considerations

- Only scan domains you own or have explicit permission to scan
- Be aware of rate limits from DNS servers and services
- Aggressive scanning may trigger security alerts
- Protect your database credentials and API keys

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Deployment Script

Here's the content of the `deploy.sh` script that powers the one-liner:

```bash
#!/bin/bash
# DNS Subdomain Scanner Deployment Script

echo "== DNS Subdomain Scanner Deployment =="
echo "Installing DNS Subdomain Scanner..."

# Check requirements
if ! command -v docker &> /dev/null; then
    echo "Error: Docker is not installed. Please install Docker first."
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    echo "Error: Docker Compose is not installed. Please install Docker Compose first."
    exit 1
fi

# Clone repository
echo "Cloning repository..."
git clone https://github.com/username/dns-scanner.git
cd dns-scanner

# Optional: Set OpenAI API key if available
if [ -n "$OPENAI_API_KEY" ]; then
    echo "OPENAI_API_KEY detected, adding to environment..."
    grep -q "OPENAI_API_KEY" docker-compose.yml || sed -i 's/# - OPENAI_API_KEY=your_api_key_here/- OPENAI_API_KEY='$OPENAI_API_KEY'/g' docker-compose.yml
fi

# Build and start Docker containers
echo "Building and starting Docker containers..."
docker-compose up -d

# Check if deployment was successful
if [ $? -eq 0 ]; then
    echo "DNS Subdomain Scanner deployed successfully!"
    echo "Access the web interface at: http://localhost:8080"
    
    # Try to open in browser if available
    if command -v xdg-open &> /dev/null; then
        xdg-open http://localhost:8080
    elif command -v open &> /dev/null; then
        open http://localhost:8080
    fi
else
    echo "Error: Deployment failed."
    echo "Please check the logs with: docker-compose logs"
    exit 1
fi
```

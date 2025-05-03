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

# Build and start Docker container
echo "Building and starting Docker container..."
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
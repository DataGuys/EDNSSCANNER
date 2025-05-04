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
# DNS Subdomain Scanner: Azure Deployment Guide

This guide walks through the complete process of deploying the DNS Subdomain Scanner to Azure with secure access through Azure AD Application Proxy.

## Prerequisites

- [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli) installed
- [Docker](https://docs.docker.com/get-docker/) installed
- An Azure subscription with adequate permissions
- An Azure AD tenant with Global Administrator or Application Administrator permissions

## Step 1: Prepare Your Environment

### Login to Azure

```bash
az login
```

### Set Default Subscription (if you have multiple)

```bash
az account set --subscription "Your Subscription Name"
```

### Create a Resource Group

```bash
az group create --name dns-scanner-rg --location eastus
```

## Step 2: Deploy Azure Container Registry (ACR)

### Create ACR

```bash
az acr create \
  --resource-group dns-scanner-rg \
  --name dnsscannerprod \
  --sku Standard \
  --admin-enabled true
```

### Login to ACR

```bash
az acr login --name dnsscannerprod
```

## Step 3: Build and Push Docker Image

### Navigate to App Directory

```bash
cd /path/to/dns-scanner
```

### Build Docker Image

```bash
docker build -t dnsscannerprod.azurecr.io/dns-scanner:v1.0 .
```

### Push Image to ACR

```bash
docker push dnsscannerprod.azurecr.io/dns-scanner:v1.0
```

## Step 4: Create Azure Database for PostgreSQL

### Create PostgreSQL Server

```bash
az postgres flexible-server create \
  --resource-group dns-scanner-rg \
  --name dns-scanner-db \
  --location eastus \
  --admin-user dnsscanner \
  --admin-password "SecurePassword123!" \
  --sku-name Standard_B1ms \
  --tier Burstable \
  --version 14

# Allow Azure services to access the PostgreSQL server
az postgres flexible-server firewall-rule create \
  --resource-group dns-scanner-rg \
  --name dns-scanner-db \
  --rule-name AllowAzureServices \
  --start-ip-address 0.0.0.0 \
  --end-ip-address 0.0.0.0
```

### Create Database

```bash
az postgres flexible-server db create \
  --resource-group dns-scanner-rg \
  --server-name dns-scanner-db \
  --database-name dnsscanner
```

## Step 5: Deploy App Service

### Create App Service Plan

```bash
az appservice plan create \
  --resource-group dns-scanner-rg \
  --name dns-scanner-plan \
  --is-linux \
  --sku P1V2
```

### Create Web App

```bash
# Get ACR credentials
ACR_USERNAME=$(az acr credential show --name dnsscannerprod --query "username" -o tsv)
ACR_PASSWORD=$(az acr credential show --name dnsscannerprod --query "passwords[0].value" -o tsv)

# Create Web App
az webapp create \
  --resource-group dns-scanner-rg \
  --plan dns-scanner-plan \
  --name dns-scanner-app \
  --deployment-container-image-name dnsscannerprod.azurecr.io/dns-scanner:v1.0

# Configure container settings
az webapp config container set \
  --name dns-scanner-app \
  --resource-group dns-scanner-rg \
  --docker-custom-image-name dnsscannerprod.azurecr.io/dns-scanner:v1.0 \
  --docker-registry-server-url https://dnsscannerprod.azurecr.io \
  --docker-registry-server-user $ACR_USERNAME \
  --docker-registry-server-password $ACR_PASSWORD
```

### Configure App Settings

```bash
az webapp config appsettings set \
  --resource-group dns-scanner-rg \
  --name dns-scanner-app \
  --settings \
    DB_HOST="dns-scanner-db.postgres.database.azure.com" \
    DB_PORT=5432 \
    DB_USER="dnsscanner@dns-scanner-db" \
    DB_PASSWORD="SecurePassword123!" \
    DB_NAME="dnsscanner" \
    DB_SSLMODE="require"
```

## Step 6: Register Application in Azure AD

### Create App Registration

```bash
# Get your tenant ID
TENANT_ID=$(az account show --query tenantId -o tsv)

# Create App Registration
APP_ID=$(az ad app create \
  --display-name "DNS Scanner App" \
  --sign-in-audience AzureADMyOrg \
  --web-redirect-uris "https://dns-scanner-app.azurewebsites.net/.auth/login/aad/callback" \
  --query appId -o tsv)

# Create Service Principal for the App
az ad sp create --id $APP_ID
```

## Step 7: Configure Azure AD Authentication in App Service

```bash
# Get your tenant ID
TENANT_ID=$(az account show --query tenantId -o tsv)

# Configure authentication settings
az webapp auth update \
  --resource-group dns-scanner-rg \
  --name dns-scanner-app \
  --enabled true \
  --action LoginWithAzureActiveDirectory \
  --aad-token-issuer-url "https://sts.windows.net/$TENANT_ID/" \
  --aad-client-id $APP_ID
```

## Step 8: Configure Azure App Proxy for the Application

This step needs to be done in the Azure Portal:

1. Navigate to Azure Active Directory > Enterprise applications
2. Create a new application > On-premises application
3. Fill in the details:
   - Name: DNS Scanner App
   - Internal URL: https://dns-scanner-app.azurewebsites.net
   - External URL: Auto-generated or custom domain
   - Pre-authentication method: Azure Active Directory
   - Connector Group: Default

## Step 9: Configure Conditional Access Policy

1. Navigate to Azure Active Directory > Security > Conditional Access
2. Create a new policy targeting the DNS Scanner App
3. Configure conditions:
   - Users and groups: Select appropriate groups
   - Cloud apps or actions: Select the DNS Scanner App
   - Conditions: Configure as needed (e.g., device platforms, locations)
4. Access controls:
   - Grant: Require multi-factor authentication
   - Session: Configure session controls as needed

## Step 10: Test the Deployment

1. Navigate to the external URL of your application (from App Proxy)
2. You should be redirected to Azure AD login
3. After authentication and MFA, you should be able to access the DNS Scanner application

## Step 11: Optional - Configure Persistent Storage

For production use, you may want to configure persistent storage for wordlists and scan results:

```bash
# Create an Azure Storage Account
az storage account create \
  --name dnsscannerstorage \
  --resource-group dns-scanner-rg \
  --location eastus \
  --sku Standard_LRS

# Create a file share
az storage share create \
  --name wordlists \
  --account-name dnsscannerstorage

# Get storage account key
STORAGE_KEY=$(az storage account keys list \
  --resource-group dns-scanner-rg \
  --account-name dnsscannerstorage \
  --query "[0].value" -o tsv)

# Configure the Web App to use the file share
az webapp config storage-account add \
  --resource-group dns-scanner-rg \
  --name dns-scanner-app \
  --custom-id wordlists \
  --storage-type AzureFiles \
  --share-name wordlists \
  --account-name dnsscannerstorage \
  --access-key "$STORAGE_KEY" \
  --mount-path /app/wordlists
```

## Troubleshooting

### Check Container Logs

```bash
az webapp log tail --name dns-scanner-app --resource-group dns-scanner-rg
```

### Check Deployment Status

```bash
az webapp deployment container show-cd-url --name dns-scanner-app --resource-group dns-scanner-rg
```

### Restart Web App

```bash
az webapp restart --name dns-scanner-app --resource-group dns-scanner-rg
```

## Security Considerations

1. **Database Security**:
   - Rotate database passwords regularly
   - Use Azure Private Link for database connectivity if possible

2. **Container Registry**:
   - Implement vulnerability scanning for container images
   - Use Azure Container Registry tasks for automated builds

3. **Application Security**:
   - Regularly review and update Azure AD application settings
   - Configure proper RBAC for the application

4. **Network Security**:
   - Consider implementing a Virtual Network for the App Service
   - Use Azure Front Door or Application Gateway for additional protection

## Bicep Deployment Alternative

If you prefer an Infrastructure-as-Code approach, use the provided Bicep template:

1. Save the template to a file named `dns-scanner-deploy.bicep`
2. Deploy using:

```bash
az deployment group create \
  --resource-group dns-scanner-rg \
  --template-file dns-scanner-deploy.bicep \
  --parameters dbAdminPassword="SecurePassword123!"
```

## Cleanup Resources

When you no longer need the resources, you can delete the entire resource group:

```bash
az group delete --name dns-scanner-rg --yes
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

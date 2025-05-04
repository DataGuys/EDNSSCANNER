# DNS Subdomain Scanner - Azure Deployment

A powerful DNS subdomain scanner with a web interface, database integration, and AI-powered wordlist generation. This solution combines passive and active enumeration techniques to discover subdomains, retrieve DNS information, and identify creation dates - all securely deployed on Azure.

## Features

- **Multiple Discovery Techniques**: Certificate Transparency logs, VirusTotal passive DNS, DNS brute force
- **Comprehensive DNS Information**: A, AAAA, CNAME, MX, TXT, NS, SOA records, IP addresses, creation dates
- **AI-Powered Wordlist Generation**: Create custom wordlists based on company information
- **Secure Azure Deployment**: Azure AD authentication, MFA, and enterprise-grade security
- **Modern Web Interface**: Real-time updates, CSV export, responsive design

## Azure Deployment Requirements

- Azure Subscription
- Azure Cloud Shell access
- Contributor role permissions
- Azure AD Premium P1 license (for Application Proxy & Conditional Access)
- Global Administrator or Application Administrator role in Azure AD

## Quick Deployment (One-Liner)

Deploy the entire solution with a single command in Azure Cloud Shell:

```bash
read -s -p "Enter your secure password: " SECUREPASSWORD
echo
curl -L https://raw.githubusercontent.com/DataGuys/EDNSSCANNER/refs/heads/main/deploy-azure.sh | bash -s -- -g "dns-scanner-rg" -l "eastus" -n "dnsscanner" -p "$SECUREPASSWORD"
```

Parameters:
- `-g`: Resource group name (default: dns-scanner-rg)
- `-l`: Azure region location (default: eastus)
- `-n`: Name prefix for resources (default: dnsscanner)
- `-p`: Database admin password (required)

## What Gets Deployed

1. **Azure Resource Group** (if it doesn't exist)
2. **Azure Container Registry** (Standard tier)
3. **Azure Database for PostgreSQL** (Flexible Server)
4. **App Service Plan** (Premium V2 tier)
5. **Web App** with container deployment
6. **Azure AD Application** registration and configuration
7. **Application Proxy** setup with pre-authentication

## Manual Deployment Steps

If you prefer more control, follow these steps:

### 1. Prepare Your Environment

```bash
# Login to Azure (if not using Cloud Shell)
az login

# Set default subscription
az account set --subscription "Your Subscription Name"

# Create resource group
az group create --name dns-scanner-rg --location eastus
```

### 2. Deploy Azure Infrastructure

```bash
# Deploy infrastructure using Bicep template
az deployment group create \
  --resource-group dns-scanner-rg \
  --template-file main.bicep \
  --parameters namePrefix=dnsscanner \
  --parameters dbAdminPassword="YourSecurePassword123!"
```

### 3. Configure Azure AD Application Proxy

```bash
# Run configuration script
./setup-app-registration.sh -g "dns-scanner-rg" -n "dns-scanner-app"
```

## Accessing Your Deployment

After deployment completes, you'll receive:

1. **Web App URL**: The direct URL to your application
2. **External URL**: The URL for accessing through Application Proxy
3. **Admin Credentials**: Initially set up with your Azure AD credentials

## Configuration

### Environment Variables

Key environment variables you may want to customize:

- `DB_HOST`: PostgreSQL server hostname
- `DB_USER`: Database username
- `DB_PASSWORD`: Database password
- `OPENAI_API_KEY`: For AI-powered wordlist generation (optional)

### Security Settings

Configure these in the Azure Portal:

1. **Conditional Access Policy**: To enforce MFA
2. **App Service Authentication**: For Azure AD integration
3. **Application Proxy**: For secure remote access

## Usage

1. Access the application through the external URL
2. Authenticate with Azure AD credentials
3. Enter a domain to scan (e.g., example.com)
4. Configure scan parameters and start the scan
5. View results in the web interface or download as CSV

## Troubleshooting

### Common Issues

- **Deployment Fails**: Check Azure activity logs for specific errors
- **Authentication Issues**: Verify Azure AD App Registration settings
- **Application Not Loading**: Check Web App logs with `az webapp log tail`

### Logs and Diagnostics

```bash
# View Web App logs
az webapp log tail --name <your-app-name> --resource-group dns-scanner-rg

# Check deployment status
az webapp deployment container show-cd-url --name <your-app-name> --resource-group dns-scanner-rg

# Restart Web App
az webapp restart --name <your-app-name> --resource-group dns-scanner-rg
```

## Cleanup

When you're done, remove all resources:

```bash
az group delete --name dns-scanner-rg --yes
```

## Security Considerations

- Only scan domains you own or have explicit permission to scan
- Rotate database passwords regularly
- Configure proper RBAC for application access
- Use Azure Private Link for enhanced database security
- Implement vulnerability scanning for container images

## License

This project is licensed under the MIT License - see the LICENSE file for details.

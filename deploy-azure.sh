#!/bin/bash
# DNS Subdomain Scanner - One-click Azure Deployment Script
# Copyright (c) 2025 - MIT License

set -e

# Default values
RESOURCE_GROUP="dns-scanner-rg"
LOCATION="eastus"
NAME_PREFIX="dnsscanner"
DB_PASSWORD=""

# Parse command line arguments
while getopts "g:l:n:p:" opt; do
  case $opt in
    g) RESOURCE_GROUP="$OPTARG" ;;
    l) LOCATION="$OPTARG" ;;
    n) NAME_PREFIX="$OPTARG" ;;
    p) DB_PASSWORD="$OPTARG" ;;
    *) echo "Usage: $0 [-g resource_group] [-l location] [-n name_prefix] [-p db_password]" >&2
       exit 1 ;;
  esac
done

# Verify required parameters
if [ -z "$DB_PASSWORD" ]; then
  echo "Error: Database password (-p) is required"
  echo "Usage: $0 [-g resource_group] [-l location] [-n name_prefix] [-p db_password]"
  exit 1
fi

# Check if logged in to Azure
echo "Verifying Azure CLI login..."
SUBSCRIPTION_ID=$(az account show --query id -o tsv 2>/dev/null || echo "")
if [ -z "$SUBSCRIPTION_ID" ]; then
  echo "You are not logged in to Azure. Please run 'az login' first."
  exit 1
fi

echo "=== DNS Subdomain Scanner - Azure Deployment ==="
echo "Subscription: $(az account show --query name -o tsv)"
echo "Resource Group: $RESOURCE_GROUP"
echo "Location: $LOCATION"
echo "Name Prefix: $NAME_PREFIX"
echo ""

# Create temporary directory
TEMP_DIR=$(mktemp -d)
cd $TEMP_DIR
echo "Working in temporary directory: $TEMP_DIR"

# Clone the repository
echo "Cloning repository..."
git clone --depth 1 https://github.com/DataGuys/EDNSSCANNER.git .

# Create resource group if it doesn't exist
echo "Ensuring resource group exists..."
az group create --name $RESOURCE_GROUP --location $LOCATION --tags "Application=DNSScanner" "DeployedBy=DeploymentScript" "DeploymentDate=$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

# Deploy infrastructure using Bicep
echo "Deploying Azure infrastructure..."
DEPLOYMENT_OUTPUT=$(az deployment group create \
  --resource-group $RESOURCE_GROUP \
  --template-file main.bicep \
  --parameters location=$LOCATION \
  --parameters namePrefix=$NAME_PREFIX \
  --parameters dbAdminPassword=$DB_PASSWORD \
  --query properties.outputs)

# Extract values from deployment output
ACR_LOGIN_SERVER=$(echo $DEPLOYMENT_OUTPUT | jq -r '.acrLoginServer.value')
WEBAPP_URL=$(echo $DEPLOYMENT_OUTPUT | jq -r '.webAppUrl.value')
POSTGRES_FQDN=$(echo $DEPLOYMENT_OUTPUT | jq -r '.postgresServerFqdn.value')
APP_ID=$(echo $DEPLOYMENT_OUTPUT | jq -r '.appRegistrationId.value')

echo "Infrastructure deployment completed!"
echo "ACR Login Server: $ACR_LOGIN_SERVER"
echo "Web App URL: $WEBAPP_URL"
echo "PostgreSQL Server: $POSTGRES_FQDN"
echo "App Registration ID: $APP_ID"

# Get ACR credentials
echo "Getting ACR credentials..."
ACR_NAME="${NAME_PREFIX}acr"
ACR_USERNAME=$(az acr credential show --name $ACR_NAME --query username -o tsv)
ACR_PASSWORD=$(az acr credential show --name $ACR_NAME --query "passwords[0].value" -o tsv)

# Build and push Docker image
echo "Building and pushing Docker image..."
REGISTRY_URL="$ACR_LOGIN_SERVER/$NAME_PREFIX:v1.0"

az acr login --name $ACR_NAME

# Build image
echo "Building Docker image..."
az acr build --registry $ACR_NAME --image $NAME_PREFIX:v1.0 --file Dockerfile .

# Configure Web App
echo "Configuring Web App..."
WEBAPP_NAME="${NAME_PREFIX}-app"

# Set container configuration
az webapp config container set \
  --resource-group $RESOURCE_GROUP \
  --name $WEBAPP_NAME \
  --docker-custom-image-name "$REGISTRY_URL" \
  --docker-registry-server-url "https://$ACR_LOGIN_SERVER" \
  --docker-registry-server-user $ACR_USERNAME \
  --docker-registry-server-password $ACR_PASSWORD

# Setup Azure AD App Proxy
echo "Setting up Azure AD Application Proxy..."
TENANT_ID=$(az account show --query tenantId -o tsv)

# Generate the script for App Proxy setup
cat > setup-app-proxy.sh << EOF
#!/bin/bash
# This part needs to be run manually due to Azure AD permissions
echo "===== Manual Steps for Azure AD Application Proxy Configuration ====="
echo "1. Go to Azure Portal > Azure Active Directory > Enterprise applications"
echo "2. Create a new application > On-premises application"
echo "3. Fill in the details:"
echo "   - Name: DNS Scanner App"
echo "   - Internal URL: $WEBAPP_URL"
echo "   - External URL: Auto-generated or custom domain"
echo "   - Pre-authentication method: Azure Active Directory"
echo "   - Connector Group: Default"
echo ""
echo "4. Navigate to Azure AD > Security > Conditional Access"
echo "5. Create a new policy targeting the DNS Scanner App"
echo "6. Configure MFA requirements as needed"
echo ""
echo "Your Web App URL: $WEBAPP_URL"
echo "Your Azure AD Tenant ID: $TENANT_ID"
echo "Your App Registration ID: $APP_ID"
EOF

chmod +x setup-app-proxy.sh

# Configure App Registration
echo "Configuring App Registration..."
REPLY_URL="${WEBAPP_URL}/.auth/login/aad/callback"

# Update App Registration with redirect URI
az ad app update \
  --id $APP_ID \
  --web-redirect-uris $REPLY_URL

# Configure Web App Authentication
echo "Configuring Web App Authentication..."
az webapp auth update \
  --resource-group $RESOURCE_GROUP \
  --name $WEBAPP_NAME \
  --enabled true \
  --action LoginWithAzureActiveDirectory \
  --aad-token-issuer-url "https://sts.windows.net/$TENANT_ID/" \
  --aad-client-id $APP_ID

# Add storage for wordlists (optional)
echo "Setting up persistent storage for wordlists..."
STORAGE_ACCOUNT="${NAME_PREFIX}storage"

# Create storage account
az storage account create \
  --resource-group $RESOURCE_GROUP \
  --name $STORAGE_ACCOUNT \
  --location $LOCATION \
  --sku Standard_LRS \
  --kind StorageV2

# Create file share
az storage share create \
  --name wordlists \
  --account-name $STORAGE_ACCOUNT

# Get storage key
STORAGE_KEY=$(az storage account keys list \
  --resource-group $RESOURCE_GROUP \
  --account-name $STORAGE_ACCOUNT \
  --query "[0].value" -o tsv)

# Mount storage to Web App
az webapp config storage-account add \
  --resource-group $RESOURCE_GROUP \
  --name $WEBAPP_NAME \
  --custom-id wordlists \
  --storage-type AzureFiles \
  --share-name wordlists \
  --account-name $STORAGE_ACCOUNT \
  --access-key "$STORAGE_KEY" \
  --mount-path /app/wordlists

# Create basic wordlist
echo "Setting up default wordlist..."
echo "www
mail
admin
blog
test
dev
api
secure
shop
store
webmail
portal
support
vpn
mobile" > common.txt

# Upload wordlist to file share
az storage file upload \
  --account-name $STORAGE_ACCOUNT \
  --account-key "$STORAGE_KEY" \
  --share-name wordlists \
  --source common.txt \
  --path common.txt

# Restart Web App to apply changes
echo "Restarting Web App..."
az webapp restart --name $WEBAPP_NAME --resource-group $RESOURCE_GROUP

# Cleanup temporary directory
cd ~
rm -rf $TEMP_DIR

echo ""
echo "=== Deployment Summary ==="
echo "DNS Scanner deployed successfully!"
echo ""
echo "Web App URL: $WEBAPP_URL"
echo "Resource Group: $RESOURCE_GROUP"
echo "Database Server: $POSTGRES_FQDN"
echo ""
echo "Important: Complete the Azure AD Application Proxy setup using the generated script:"
echo "$ ./setup-app-proxy.sh"
echo ""
echo "Troubleshooting:"
echo "- To view logs: az webapp log tail --name $WEBAPP_NAME --resource-group $RESOURCE_GROUP"
echo "- To restart app: az webapp restart --name $WEBAPP_NAME --resource-group $RESOURCE_GROUP"
echo ""
echo "Thank you for using DNS Subdomain Scanner!"

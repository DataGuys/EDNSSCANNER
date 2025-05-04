#!/bin/bash
# DNS Subdomain Scanner Deployment Script
# Setup script for Azure AD App Registration and App Proxy

# Variables - update these for your environment
RESOURCE_GROUP="dns-scanner-rg"
WEBAPP_NAME="dns-scanner-app"
APP_DISPLAY_NAME="DNS Scanner App"
REPLY_URL=""  # Will be auto-populated

# Get tenant ID
TENANT_ID=$(az account show --query tenantId -o tsv)
echo "Using Tenant ID: $TENANT_ID"

# Get web app URL
WEBAPP_URL=$(az webapp show \
  --resource-group $RESOURCE_GROUP \
  --name $WEBAPP_NAME \
  --query defaultHostName \
  --output tsv)
echo "Web App URL: $WEBAPP_URL"

# Set reply URL
REPLY_URL="https://$WEBAPP_URL/.auth/login/aad/callback"
echo "Reply URL: $REPLY_URL"

# Check if app registration already exists
APP_ID=$(az ad app list \
  --display-name "$APP_DISPLAY_NAME" \
  --query "[0].appId" \
  --output tsv)

if [ -z "$APP_ID" ]; then
  echo "Creating new app registration..."
  
  # Create app registration
  APP_ID=$(az ad app create \
    --display-name "$APP_DISPLAY_NAME" \
    --sign-in-audience AzureADMyOrg \
    --web-redirect-uris "$REPLY_URL" \
    --query "appId" \
    --output tsv)
  
  echo "Created app with ID: $APP_ID"
  
  # Create service principal for the app
  az ad sp create --id $APP_ID
  echo "Created service principal for app"
else
  echo "App registration already exists with ID: $APP_ID"
  
  # Update existing app registration with correct reply URL
  az ad app update \
    --id $APP_ID \
    --web-redirect-uris "$REPLY_URL"
  
  echo "Updated app registration with current reply URL"
fi

# Configure required API permissions
echo "Configuring API permissions..."

# Microsoft Graph - User.Read permission (for basic profile)
az ad app permission add \
  --id $APP_ID \
  --api 00000003-0000-0000-c000-000000000000 \
  --api-permissions e1fe6dd8-ba31-4d61-89e7-88639da4683d=Scope

# Grant admin consent for the API permissions
echo "Granting admin consent for API permissions..."
az ad app permission admin-consent --id $APP_ID

# Configure web app authentication
echo "Configuring Web App Authentication..."
az webapp auth update \
  --resource-group $RESOURCE_GROUP \
  --name $WEBAPP_NAME \
  --enabled true \
  --action LoginWithAzureActiveDirectory \
  --aad-token-issuer-url "https://sts.windows.net/$TENANT_ID/" \
  --aad-client-id $APP_ID

# Output information for App Proxy setup
echo ""
echo "==== NEXT STEPS ===="
echo "To configure Azure AD Application Proxy:"
echo "1. Go to Azure Portal > Azure Active Directory > Enterprise applications"
echo "2. Create a new application > On-premises application"
echo "3. Fill in the details:"
echo "   - Name: $APP_DISPLAY_NAME"
echo "   - Internal URL: https://$WEBAPP_URL"
echo "   - Pre-authentication method: Azure Active Directory"
echo "   - Connector Group: Default"
echo ""
echo "For Conditional Access policy:"
echo "1. Go to Azure Active Directory > Security > Conditional Access"
echo "2. Create a new policy targeting the application"
echo "3. Configure MFA requirements as needed"
echo ""
echo "Your App Registration ID: $APP_ID"
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

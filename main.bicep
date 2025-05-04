// DNS Subdomain Scanner Azure Deployment
// This template deploys:
// - Azure Container Registry
// - Azure Database for PostgreSQL
// - App Service Plan
// - Web App with Container
// - Azure App Proxy Configuration

@description('Location for all resources.')
param location string = resourceGroup().location

@description('Name prefix for all resources')
param namePrefix string = 'dnsscanner'

@description('Admin username for PostgreSQL')
param dbAdminUsername string = 'dnsscanner'

@description('Admin password for PostgreSQL')
@secure()
param dbAdminPassword string

@description('The Azure AD Tenant ID')
param tenantId string = subscription().tenantId

// Resource names
var acrName = '${namePrefix}acr'
var appServicePlanName = '${namePrefix}-plan'
var webAppName = '${namePrefix}-app'
var postgresServerName = '${namePrefix}-db'
var dbName = 'dnsscanner'

// Container Registry
resource acr 'Microsoft.ContainerRegistry/registries@2023-01-01-preview' = {
  name: acrName
  location: location
  sku: {
    name: 'Standard'
  }
  properties: {
    adminUserEnabled: true
  }
}

// Azure Database for PostgreSQL
resource postgresServer 'Microsoft.DBforPostgreSQL/flexibleServers@2022-12-01' = {
  name: postgresServerName
  location: location
  sku: {
    name: 'Standard_B1ms'
    tier: 'Burstable'
  }
  properties: {
    version: '14'
    administratorLogin: dbAdminUsername
    administratorLoginPassword: dbAdminPassword
    storage: {
      storageSizeGB: 32
    }
    backup: {
      backupRetentionDays: 7
      geoRedundantBackup: 'Disabled'
    }
    highAvailability: {
      mode: 'Disabled'
    }
  }
}

// PostgreSQL Database
resource postgresDatabase 'Microsoft.DBforPostgreSQL/flexibleServers/databases@2022-12-01' = {
  name: dbName
  parent: postgresServer
  properties: {
    charset: 'UTF8'
    collation: 'en_US.utf8'
  }
}

// PostgreSQL Firewall Rule - Allow Azure Services
resource postgresFirewallRule 'Microsoft.DBforPostgreSQL/flexibleServers/firewallRules@2022-12-01' = {
  name: 'AllowAllAzureIPs'
  parent: postgresServer
  properties: {
    startIpAddress: '0.0.0.0'
    endIpAddress: '0.0.0.0'
  }
}

// App Service Plan
resource appServicePlan 'Microsoft.Web/serverfarms@2022-03-01' = {
  name: appServicePlanName
  location: location
  sku: {
    name: 'P1v2'
    tier: 'PremiumV2'
  }
  kind: 'linux'
  properties: {
    reserved: true
  }
}

// Web App
resource webApp 'Microsoft.Web/sites@2022-03-01' = {
  name: webAppName
  location: location
  properties: {
    serverFarmId: appServicePlan.id
    httpsOnly: true
    siteConfig: {
      linuxFxVersion: 'DOCKER|${acrName}.azurecr.io/${namePrefix}:v1.0'
      appSettings: [
        {
          name: 'WEBSITES_ENABLE_APP_SERVICE_STORAGE'
          value: 'false'
        }
        {
          name: 'DOCKER_REGISTRY_SERVER_URL'
          value: 'https://${acrName}.azurecr.io'
        }
        {
          name: 'DOCKER_REGISTRY_SERVER_USERNAME'
          value: acr.listCredentials().username
        }
        {
          name: 'DOCKER_REGISTRY_SERVER_PASSWORD'
          value: acr.listCredentials().passwords[0].value
        }
        {
          name: 'DB_HOST'
          value: postgresServer.properties.fullyQualifiedDomainName
        }
        {
          name: 'DB_PORT'
          value: '5432'
        }
        {
          name: 'DB_USER'
          value: '${dbAdminUsername}@${postgresServerName}'
        }
        {
          name: 'DB_PASSWORD'
          value: dbAdminPassword
        }
        {
          name: 'DB_NAME'
          value: dbName
        }
        {
          name: 'DB_SSLMODE'
          value: 'require'
        }
      ]
    }
  }
  identity: {
    type: 'SystemAssigned'
  }
}

// Configure Authentication (for App Proxy)
resource authSettings 'Microsoft.Web/sites/config@2022-03-01' = {
  name: 'authsettingsV2'
  parent: webApp
  properties: {
    platform: {
      enabled: true
    }
    globalValidation: {
      requireAuthentication: true
      unauthenticatedClientAction: 'RedirectToLoginPage'
    }
    identityProviders: {
      azureActiveDirectory: {
        enabled: true
        registration: {
          openIdIssuer: 'https://sts.windows.net/${tenantId}/'
          clientId: appRegistration.properties.appId
        }
        login: {
          loginParameters: ['domain_hint=tenant.onmicrosoft.com']
        }
      }
    }
    login: {
      tokenStore: {
        enabled: true
      }
    }
  }
}

// App Registration for Azure AD
resource appRegistration 'Microsoft.AzureActiveDirectory/b2cDirectories/applications@2019-01-01-preview' = {
  name: webAppName
  properties: {
    displayName: 'DNS Scanner App'
    web: {
      redirectUris: [
        'https://${webApp.properties.defaultHostName}/.auth/login/aad/callback'
      ]
    }
  }
}

// Output important values
output acrLoginServer string = acr.properties.loginServer
output webAppUrl string = 'https://${webApp.properties.defaultHostName}'
output postgresServerFqdn string = postgresServer.properties.fullyQualifiedDomainName
output appRegistrationId string = appRegistration.properties.appId

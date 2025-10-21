# Kibaship CLI Configuration Examples

This directory contains example YAML configuration files for creating Kibaship clusters using the CLI.

## Usage

Use any of these configuration files with the `--configuration` flag:

```bash
kibaship clusters create --configuration cmd/cli/examples/digitalocean-cluster.yaml
```

## Configuration Structure

All configuration files follow this nested structure:

```yaml
# Terraform state configuration (required)
state:
  s3:
    bucket: "my-terraform-state"
    region: "us-east-1"
    access-key: "AKIAIOSFODNN7EXAMPLE"
    access-secret: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"

# Cluster configuration (required)
cluster:
  name: "app.kibaship.com"  # Domain format required
  email: "admin@kibaship.com"  # For credentials and Let's Encrypt
  paas-features: "none"  # TODO: Database features removed - will be reimplemented

  # Provider configuration (choose one)
  provider:
    digital-ocean:
      token: "your-token"
      nodes: "3"
      nodes-size: "s-4vcpu-8gb"
      region: "nyc3"
```

## Available Examples

### `digitalocean-cluster.yaml`
**Production DigitalOcean cluster**
- 5 high-performance nodes (s-8vcpu-16gb)
- TODO: Database features removed - will be reimplemented
- NYC3 region

### `aws-minimal-cluster.yaml`
**Minimal AWS staging cluster**
- TODO: Database features removed - will be reimplemented
- US West 2 region
- Staging environment setup

### `hetzner-robot-bare-cluster.yaml`
**Bare Kubernetes cluster on Hetzner Robot**
- No PaaS features (bare cluster)
- Dedicated server infrastructure
- Development environment

### `hetzner-cloud-cluster.yaml`
**Hetzner Cloud cluster**
- TODO: Database features removed - will be reimplemented
- EU Central region
- Cost-effective cloud setup

### `gcloud-enterprise-cluster.yaml`
**Enterprise Google Cloud cluster**
- TODO: Database features removed - will be reimplemented
- Service account authentication
- Enterprise-grade configuration

### `linode-cluster.yaml`
**Linode cluster**
- TODO: Database features removed - will be reimplemented
- Simple cluster setup

## Field Reference

### Required Fields

#### State Configuration
- `state.s3.bucket`: S3 bucket name for Terraform state
- `state.s3.region`: S3 bucket region
- `state.s3.access-key`: AWS access key for S3 access
- `state.s3.access-secret`: AWS secret key for S3 access

#### Cluster Configuration
- `cluster.name`: Cluster name in domain format (e.g., `app.kibaship.com`)
- `cluster.email`: Email for default credentials and Let's Encrypt certificates

### Optional Fields

#### PaaS Features
- `cluster.paas-features`: Comma-separated list of PaaS features
  - Options: `none` (TODO: Database features removed - will be reimplemented)
  - Default: `none`
  - Examples: `"none"`

### Provider-Specific Fields

Configure only the provider you're using under `cluster.provider`:

#### AWS (`cluster.provider.aws`)
- `access-key-id`: AWS Access Key ID
- `secret-access-key`: AWS Secret Access Key
- `region`: AWS region (e.g., `us-east-1`)

#### DigitalOcean (`cluster.provider.digital-ocean`)
- `token`: DigitalOcean API token
- `nodes`: Number of nodes in the node pool
- `nodes-size`: Droplet size (e.g., `s-2vcpu-2gb`, `s-4vcpu-8gb`)
- `region`: DigitalOcean region (e.g., `nyc1`, `sfo3`, `ams3`)

#### Hetzner Cloud (`cluster.provider.hetzner`)
- `token`: Hetzner Cloud API token

#### Hetzner Robot (`cluster.provider.hetzner-robot`)
- `username`: Hetzner Robot username
- `password`: Hetzner Robot password
- `cloud-token`: Hetzner Cloud token

#### Linode (`cluster.provider.linode`)
- `token`: Linode API token

#### Google Cloud (`cluster.provider.gcloud`)
- `service-account-key`: Path to service account key JSON file
- `project-id`: Google Cloud project ID
- `region`: Google Cloud region (e.g., `us-central1`)

## Security Best Practices

⚠️ **Important Security Notes:**

1. **Never commit real credentials** to version control
2. **Use environment variables** for sensitive values
3. **Separate configurations** for different environments
4. **Secure your S3 bucket** for Terraform state storage
5. **Use IAM roles** when possible instead of access keys

## Getting Started

1. **Choose an example** that matches your cloud provider
2. **Copy the file** to your project directory
3. **Replace example values** with your actual credentials
4. **Customize** cluster name, email, and PaaS features
5. **Run the create command**

```bash
# Copy and customize an example
cp cmd/cli/examples/digitalocean-cluster.yaml my-cluster.yaml

# Edit with your credentials and preferences
vim my-cluster.yaml

# Create the cluster
kibaship clusters create --configuration my-cluster.yaml
```

## Validation

All configuration files are validated when used:
- ✅ Required fields must be present
- ✅ Email addresses must be valid format
- ✅ Domain names must be properly formatted
- ✅ PaaS features must be valid options
- ✅ Provider-specific fields validated based on chosen provider

## Environment-Specific Configurations

Consider creating separate configuration files for different environments:

```
configs/
├── production-digitalocean.yaml
├── staging-aws.yaml
└── development-hetzner.yaml
```

This allows you to maintain different cluster configurations for different deployment stages while keeping credentials and settings organized.

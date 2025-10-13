# Kibaship Terraform Templates

This directory contains Terraform templates for provisioning Kubernetes clusters across different cloud providers.

## Structure

```
terraform/
├── providers/
│   └── digital-ocean/
│       ├── vars.tf.tpl      # Variable definitions
│       ├── provision.tf.tpl # Main cluster provisioning
│       └── bootstrap.tf.tpl # PaaS services installation
└── README.md
```

## Template Processing

Templates use Go's built-in `text/template` package and are embedded into the CLI binary using Go's `embed` directive.

### Variable Population

Variables are populated via TF_VAR environment variables when running Terraform:

```bash
export TF_VAR_cluster_name="my-cluster"
export TF_VAR_cluster_email="admin@example.com"
export TF_VAR_do_token="your-digitalocean-token"
export TF_VAR_do_region="nyc3"
export TF_VAR_do_node_count=3
export TF_VAR_do_node_size="s-4vcpu-8gb"
# ... other variables

terraform plan
terraform apply
```

### Template Files

#### `vars.tf.tpl`
- Defines all Terraform variables without defaults
- Variables populated via TF_VAR environment variables
- Includes sensitive variable marking
- Contains local values for internal use

#### `provision.tf.tpl`
- Main cluster provisioning logic
- Provider configuration
- Terraform S3 backend configuration
- VPC and networking setup
- Kubernetes cluster creation
- Output definitions

#### `bootstrap.tf.tpl`
- Currently empty (reserved for future PaaS services)
- Will be used for PaaS services installation

## Features

### Core Infrastructure
- ✅ **DigitalOcean Kubernetes cluster** with basic configuration
- ✅ **VPC networking** with proper IP ranges (10.10.0.0/16)
- ✅ **Terraform state** stored in S3 with encryption
- ✅ **Node pool** with configurable size and count
- ✅ **Cluster outputs** for integration

### Template Features
- ✅ **Go template variables** for dynamic configuration
- ✅ **S3 backend** configuration from user credentials
- ✅ **Provider-specific** variable handling
- ✅ **Sensitive variable** marking for security
- ✅ **Clean separation** between provisioning and bootstrapping

## Usage

Templates are processed during cluster creation:

1. **Load template** from embedded filesystem
2. **Parse template** with Go's text/template (minimal processing)
3. **Write output** to temporary Terraform files
4. **Set TF_VAR environment variables** from configuration
5. **Configure Terraform backend** via init flags
6. **Run terraform** commands on generated files

## Adding New Providers

To add a new cloud provider:

1. Create directory: `providers/{provider-name}/`
2. Add three template files:
   - `vars.tf.tpl` - Variable definitions
   - `provision.tf.tpl` - Main provisioning logic
   - `bootstrap.tf.tpl` - PaaS services setup
3. Update CLI to support the new provider
4. Add provider-specific configuration fields

## Security

- ✅ **Sensitive variables** marked appropriately
- ✅ **Credentials** passed via environment variables
- ✅ **State encryption** enabled in S3 backend
- ✅ **Network isolation** with VPC
- ✅ **TLS encryption** for all services

# GitOps Setup with ArgoCD

This repository is configured for GitOps deployment using ArgoCD with Kustomize for application management.

## Repository Structure

```
├── apps/                           # Application definitions
│   ├── cert-manager/               # SSL certificate management
│   │   ├── base/                   # Base configuration
│   │   └── overlays/staging/       # Environment-specific overlays
│   └── ingress-gateway/             # Ingress gateway configuration
│       ├── base/                   # Base gateway configuration
│       └── overlays/staging/       # Environment-specific overlays
├── clusters/                       # Cluster-specific configurations
│   ├── staging/                    # Staging cluster apps
│   └── production/                 # Production cluster (empty for now)
├── argocd/                         # ArgoCD application definitions
│   ├── applications/               # Individual application manifests
│   │   └── staging/
│   │       ├── infrastructure/     # Infrastructure apps (cert-manager, gateways)
│   │       └── applications/       # Business applications
│   └── app-of-apps/               # App of Apps pattern
└── terraform/                      # Infrastructure as Code
```

## Step-by-Step Setup Instructions

### Step 1: Update Repository URL

Update the `repoURL` in ArgoCD application files with your actual repository URL:
- `argocd/applications/staging/infrastructure/cert-manager.yaml`
- `argocd/applications/staging/infrastructure/ingress-gateway.yaml`
- `argocd/app-of-apps/staging.yaml`

Replace `https://github.com/kibamail/kibaship.git` with your actual repository URL.

### Step 2: Deploy App of Apps (Recommended)

Deploy the App of Apps to manage all applications for staging:

```bash
kubectl apply -f argocd/app-of-apps/staging.yaml
```

### Step 3: Alternative - Deploy Individual Applications

If you prefer to deploy applications individually:

```bash
kubectl apply -f argocd/applications/staging/infrastructure/cert-manager.yaml
kubectl apply -f argocd/applications/staging/infrastructure/ingress-gateway.yaml
```

### Step 4: Verify Deployment

1. Check ArgoCD UI to see your applications
2. Verify infrastructure components are deployed:

```bash
# Check cert-manager
kubectl get pods -n cert-manager
kubectl get crd | grep cert-manager

# Check Cilium Gateway
kubectl get pods -n ingress-gateway
kubectl get gateway -n ingress-gateway
kubectl get httproute -n ingress-gateway
```

## Adding New Applications

### Infrastructure Applications
For infrastructure components (gateways, monitoring, etc.):

1. Create application structure in `apps/your-app/`
2. Add overlay for staging in `apps/your-app/overlays/staging/`
3. Update `clusters/staging/kustomization.yaml` to include the new app
4. Create ArgoCD application manifest in `argocd/applications/staging/infrastructure/`

### Business Applications
For business applications (APIs, web apps, etc.):

1. Create application structure in `apps/your-app/`
2. Add overlay for staging in `apps/your-app/overlays/staging/`
3. Create ArgoCD application manifest in `argocd/applications/staging/applications/`
4. Update App of Apps if needed

## Current Applications

### Infrastructure
- **cert-manager**: SSL certificate management for staging environment
- **ingress-gateway**: Ingress gateway handling HTTP/HTTPS traffic on ports 30080/30443
  - Listens to: `*.staging.kibaship.app` and `argocd.staging.kibaship.com`
  - Ready for additional domains to be added

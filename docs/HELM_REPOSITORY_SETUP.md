# Helm Repository Setup Guide

This guide explains how to set up a Helm repository for the KibaShip Operator chart.

## Option 1: GitHub Pages Helm Repository

### 1. Create a separate helm-charts repository

```bash
# Create new repository: kibamail/helm-charts
mkdir helm-charts && cd helm-charts
git init
```

### 2. Set up repository structure

```
helm-charts/
├── index.yaml                 # Helm repository index
├── charts/                    # Packaged chart files
│   └── kibaship-0.1.0.tgz
└── docs/                      # GitHub Pages source
    ├── index.yaml -> ../index.yaml
    └── charts/ -> ../charts/
```

### 3. Add GitHub Action to main operator repo

```yaml
# .github/workflows/build-and-push.yml (addition)
helm-release:
  needs: release
  runs-on: ubuntu-latest
  steps:
    - name: Package Helm Chart
      run: |
        helm package deploy/helm/kibaship/ --destination ./charts/

    - name: Update Helm Repository
      env:
        HELM_REPO_TOKEN: ${{ secrets.HELM_REPO_TOKEN }}
      run: |
        # Clone helm-charts repo
        git clone https://x-access-token:${HELM_REPO_TOKEN}@github.com/kibamail/helm-charts.git

        # Copy new chart
        cp charts/*.tgz helm-charts/charts/

        # Update index
        cd helm-charts
        helm repo index . --url https://helm.kibaship.com

        # Commit and push
        git add .
        git commit -m "Add kibaship v${{ steps.version.outputs.VERSION }}"
        git push
```

### 4. User Installation

```bash
helm repo add kibaship https://helm.kibaship.com
helm repo update
helm install kibaship kibaship/kibaship
```

## Option 2: OCI Registry (Modern Approach)

### 1. Push charts to OCI registry

```bash
# Package chart
helm package deploy/helm/kibaship/

# Push to GitHub Container Registry
helm push kibaship-0.1.0.tgz oci://kibamail/charts
```

### 2. User Installation

```bash
helm install kibaship oci://kibamail/charts/kibaship --version 0.1.0
```

## Option 3: GitHub Releases (Simple)

### 1. Attach chart to GitHub releases

The current CI/CD can be extended to package and attach the chart:

```yaml
# .github/workflows/build-and-push.yml (addition)
- name: Package Helm Chart
  run: |
    helm package deploy/helm/kibaship/ --destination ./charts/

- name: Create GitHub Release
  uses: softprops/action-gh-release@v2
  with:
    files: |
      dist/install.yaml
      charts/kibaship-*.tgz  # Add chart package
```

### 2. User Installation

```bash
# Download from GitHub releases
curl -L -o chart.tgz https://github.com/kibamail/kibaship/releases/download/v0.1.0/kibaship-0.1.0.tgz
helm install kibaship chart.tgz
```

## Current Status

- ✅ **Chart exists**: Functional Helm chart in `deploy/helm/kibaship/`
- ✅ **Versioning**: Automatic version management in release script (chart version = app version)
- ✅ **CI/CD ready**: Fully integrated into GitHub Actions release workflow
- ✅ **Distribution**: Charts attached to GitHub releases automatically

## Recommendation

For production use, implement **Option 1 (GitHub Pages)** or **Option 2 (OCI Registry)** for the best user experience:

```bash
# Goal: Simple installation
helm repo add kibaship https://helm.kibaship.com
helm install kibaship kibaship/kibaship
```

This provides:

- Automatic chart updates
- Version management
- Easy discovery
- Standard Helm workflow

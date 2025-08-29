# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common Development Commands

```bash
# Development
pnpm dev             # Start development server with HMR
pnpm build           # Production build
pnpm start           # Start production server

# Testing
pnpm test            # Run all tests (unit + functional)
node ace test unit   # Run unit tests only  
node ace test functional # Run functional tests only

# Code Quality
pnpm lint            # ESLint
pnpm format          # Prettier formatting
pnpm typecheck       # TypeScript type checking

# Database
node ace migration:run    # Run pending migrations
node ace migration:rollback # Rollback last migration
node ace db:seed     # Run seeders
```

## Architecture Overview

**Kibaship.com** is a Kubernetes cluster management platform built on AdonisJS v6 with React frontend via Inertia.js.

### Core Domain Models & Relationships

```
User → Workspace → CloudProvider (Hetzner, DigitalOcean)
                → Project → Application → Deployment  
                → Cluster → ClusterNode, ClusterLoadBalancer, ClusterSshKey
```

### Multi-Stage Infrastructure Provisioning

The platform orchestrates Kubernetes cluster creation through sequential job execution:

1. **ProvisionClusterJob** → triggers **ProvisionNetworkJob**
2. **ProvisionNetworkJob** → triggers **ProvisionSshKeysJob**  
3. **ProvisionSshKeysJob** → triggers **ProvisionLoadBalancersJob**
4. **ProvisionServersJob** → **ProvisionVolumesJob** (not yet implemented)

Each stage:
- Updates cluster timestamps (`networkingStartedAt`, `networkingCompletedAt`, etc.)
- Stores error states (`networkingError`, `networkingErrorAt`)
- Uses TerraformService to generate and execute infrastructure-as-code

### Key Services Architecture

**TerraformService**: Generates cloud-provider-specific Terraform templates using Edge.js templating. Templates are stored in `resources/views/clusters/terraform/{provider}/` and rendered with cluster context (nodes, SSH keys, network config).

**Job System**: Bull Queue with Redis handles long-running infrastructure operations. Jobs follow the pattern:
```typescript
class ProvisionXJob extends Job {
  async handle(payload: { clusterId: string }) {
    // 1. Load cluster with relationships
    // 2. Generate Terraform template 
    // 3. Execute terraform via TerraformExecutor
    // 4. Update cluster state
    // 5. Dispatch next job in chain
  }
}
```

**Authentication Flow**: OAuth2 integration with GitHub via `@kibamail/auth-sdk`. Routes use `middleware.auth()` for workspace-scoped access.

### Frontend Architecture

**Inertia.js SSR**: React components receive props from AdonisJS controllers. Key pages:
- `inertia/pages/dashboard.tsx` - Workspace overview
- `inertia/pages/clusters/clusters.tsx` - Cluster management
- `inertia/pages/projects/project.tsx` - Project details

**Real-time Updates**: Socket.io for cluster provisioning status, Redis streams for log processing via `cluster_logs_handler.ts`.

### Data Layer Patterns

**Models**: Extend AdonisJS BaseModel with:
- UUID primary keys (`@beforeCreate generateId`)
- Relationship preloading patterns (`Cluster.complete()` loads all relations)
- Status tracking via enums (`ClusterStatus`, `ProvisioningStepStatus`)

**Migrations**: Numbered sequentially, cluster-related tables use extensive timestamp tracking for provisioning stages.

### File Organization

- `app/controllers/` - Grouped by domain (Auth/, Dashboard/, Projects/)
- `app/services/` - Business logic (terraform/, hetzner/, redis/)
- `app/jobs/clusters/` - Infrastructure provisioning jobs
- `resources/views/clusters/terraform/` - IaC templates per provider
- `inertia/` - React frontend components and pages

### Testing Strategy

- **Functional tests**: Full request/response testing in `tests/functional/`
- **Unit tests**: Service and validator testing in `tests/unit/`
- Test suites configured in `adonisrc.ts` with different timeouts (2s unit, 30s functional)

### Environment Dependencies

- MySQL database with Lucid ORM
- Redis for queues and streams  
- Terraform binary for infrastructure execution
- Vault for secrets management (HashiCorp Vault)
- Cloud provider APIs (Hetzner Cloud, DigitalOcean)
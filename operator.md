# KibaShip Operator - Complete Technical Documentation

## Overview

The KibaShip Operator is a Kubernetes controller that manages multi-tenant platform resources for application deployment and management. It provides a declarative API for creating isolated project environments with RBAC, resource management, and CI/CD integration through Tekton Pipelines.

## Architecture

### Custom Resource Definitions (CRDs)

The operator manages three primary custom resources:

1. **Project** - Top-level tenant isolation unit
2. **Application** - Application definitions within projects  
3. **Deployment** - CI/CD pipeline management for applications

### Controller Components

- **ProjectReconciler** - Manages project lifecycle and namespace provisioning
- **ApplicationReconciler** - Handles application resource management
- **DeploymentReconciler** - Manages deployment pipelines (TODO: implementation pending)
- **NamespaceManager** - Core component for namespace and RBAC management

## Detailed Operational Flow

### 1. Project Creation Workflow

When a new `Project` resource is created, the following sequence occurs:

#### 1.1 Initial Validation and Setup
```
Project Creation Event → ProjectReconciler.Reconcile()
```

**Step 1: Resource Retrieval**
- Controller fetches the Project resource from the Kubernetes API
- If resource not found (deleted), reconciliation ends gracefully

**Step 2: Deletion Check**
- If `DeletionTimestamp` is set, triggers deletion workflow (see section 3)
- Otherwise proceeds with creation/update workflow

**Step 3: Finalizer Management**
- Adds finalizer `platform.kibaship.com/project-finalizer` to ensure proper cleanup
- Updates resource and requeues for next reconciliation cycle

**Step 4: Label Validation**
- Validates required UUID labels:
  - `platform.kibaship.com/uuid` (required, must be valid UUID format)
  - `platform.kibaship.com/workspace-uuid` (optional, must be valid UUID if present)
- Uses regex validation: `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`
- If validation fails, updates status to "Failed" and stops reconciliation

#### 1.2 Namespace and RBAC Provisioning

**Step 5: Namespace Creation**
```
NamespaceManager.CreateProjectNamespace() called
```

The namespace creation process involves multiple sub-operations:

**5.1 Namespace Generation**
- Generates namespace name using pattern: `project-{project-name}-ns-kibaship-com`
- Creates namespace with labels:
  - `platform.kibaship.com/managed-by: kibaship-operator`
  - `platform.kibaship.com/project-name: {project-name}`
  - `platform.kibaship.com/uuid: {project-uuid}`
  - `platform.kibaship.com/workspace-uuid: {workspace-uuid}`

**5.2 Service Account Creation**
- Creates service account with name: `project-{project-name}-sa-kibaship-com`
- Applied in the project namespace
- Labeled with management metadata

**5.3 Admin Role Creation**  
- Creates Role with name: `project-{project-name}-admin-role-kibaship-com`
- Grants full permissions within the project namespace:
  ```yaml
  rules:
  - apiGroups: ["*"]
    resources: ["*"] 
    verbs: ["*"]
  ```
- Scoped only to the project namespace

**5.4 Role Binding Creation**
- Creates RoleBinding with name: `project-{project-name}-admin-binding-kibaship-com`
- Binds the project service account to the admin role
- Enables the service account to perform all operations within its namespace

#### 1.3 Tekton Integration Setup

**Step 6: Tekton Pipeline Integration**
```
NamespaceManager.createTektonIntegration() called
```

**6.1 Tekton Namespace Verification**
- Checks if `tekton-pipelines` namespace exists
- This namespace should be created by Tekton installation

**6.2 Tekton Role Management**
- Checks for existing `tekton-tasks-reader` role in `tekton-pipelines` namespace
- If not found, creates role with permissions:
  ```yaml
  rules:
  - apiGroups: ["tekton.dev"]
    resources: ["tasks"]
    verbs: ["get", "list", "watch"]
  ```
- Role is labeled as managed by the operator
- Only one role exists cluster-wide (shared across projects)

**6.3 Tekton Role Binding Creation**
- Creates RoleBinding in `tekton-pipelines` namespace
- Binding name: `project-{project-name}-tekton-binding-kibaship-com`  
- Binds project service account to `tekton-tasks-reader` role
- Enables project service accounts to read Tekton tasks for CI/CD

#### 1.4 Status Updates

**Step 7: Success Status Update**
- Updates Project status to "Ready" phase
- Sets `NamespaceName` field to created namespace name
- Records `LastReconcileTime` 
- Adds success message

**Final State After Project Creation:**
- ✅ Dedicated namespace created and labeled
- ✅ Service account with admin privileges in namespace
- ✅ Role and RoleBinding for namespace admin access
- ✅ Tekton integration for CI/CD capabilities
- ✅ Project status reflects ready state

### 2. Application Creation Workflow

When a new `Application` resource is created:

#### 2.1 Initial Processing
```
Application Creation Event → ApplicationReconciler.Reconcile()
```

**Step 1: Resource Retrieval and Validation**
- Fetches Application resource
- Validates deletion timestamp (triggers deletion if set)

**Step 2: UUID Label Management**
- Ensures application inherits UUID labels from parent project:
  - `platform.kibaship.com/project-uuid`
  - `platform.kibaship.com/workspace-uuid`
- Queries referenced project to copy UUID labels
- Updates Application resource if labels were added

**Step 3: Finalizer Management**
- Adds finalizer `platform.operator.kibaship.com/application-finalizer`
- Ensures proper cleanup during deletion

#### 2.2 Application Type Processing

**Step 4: Type-Specific Handling**
Based on `spec.type`, different configuration validation occurs:

**GitRepository Applications:**
- Validates provider (github.com, gitlab.com, bitbucket.com)
- Validates repository format: `^[a-zA-Z0-9._-]+/[a-zA-Z0-9._-]+$`
- **PublicAccess Field**: Controls access validation behavior
  - When `PublicAccess: true` - SecretRef is optional (for public repositories)
  - When `PublicAccess: false` (default) - SecretRef is required and must exist in project namespace
- Optional: branch, path, build/start commands, environment variables, SPA output directory

**DockerImage Applications:**
- Validates image reference format
- Optional: image pull secret reference
- Optional: specific tag override

**Database Applications (MySQL/Postgres):**
- Single instance: basic config with version and database name
- Cluster: includes replica count (min 1, default 3)
- Optional: secret reference for credentials

#### 2.3 Application Status Update

**Step 5: Status Update**
- Updates application status to "Ready" 
- Sets conditions for application readiness
- No automatic resource creation occurs

### 3. Project Deletion Workflow

When a `Project` resource is deleted:

#### 3.1 Deletion Detection
```
Project Deletion Event → ProjectReconciler.handleProjectDeletion()
```

**Step 1: Finalizer Check**
- Only proceeds if finalizer `platform.kibaship.com/project-finalizer` is present
- Prevents accidental cleanup of non-operator managed resources

#### 3.2 Resource Cleanup Sequence

**Step 2: Service Account Resource Cleanup**
```
NamespaceManager.deleteServiceAccountResources() called
```

**2.1 Tekton Role Binding Cleanup**
- Deletes RoleBinding `project-{project-name}-tekton-binding-kibaship-com` from `tekton-pipelines` namespace
- Removes project's access to Tekton tasks
- Note: Shared `tekton-tasks-reader` role is preserved for other projects

**2.2 Namespace RBAC Cleanup**
- Deletes RoleBinding: `project-{project-name}-admin-binding-kibaship-com`
- Deletes Role: `project-{project-name}-admin-role-kibaship-com`
- Deletes ServiceAccount: `project-{project-name}-sa-kibaship-com`

**Step 3: Namespace Deletion**
- Deletes project namespace: `project-{project-name}-ns-kibaship-com`
- Kubernetes automatically removes all resources within the namespace
- Includes any Applications, Deployments, and user workloads

**Step 4: Finalizer Removal**
- Removes finalizer from Project resource
- Allows Kubernetes to complete resource deletion
- Project resource is permanently removed from cluster

### 4. Application Deletion Workflow

When an `Application` resource is deleted:

#### 4.1 Cascade Deletion
```
Application Deletion Event → ApplicationReconciler.handleDeletion()
```

**Step 1: Dependent Resource Cleanup**
- Finds and deletes associated `Deployment` resources using label selector
- Uses label `platform.operator.kibaship.com/application: {app-name}` to find dependent deployments
- Deployments must be explicitly created by users or other systems

**Step 2: Finalizer Removal**
- Removes finalizer `platform.operator.kibaship.com/application-finalizer`
- Completes Application deletion

### 5. Deployment Management 

Deployments are separate resources that reference Applications but are **not automatically created**:

#### 5.1 Deployment Creation
- Users must explicitly create `Deployment` resources
- Each Deployment must reference an Application via `spec.applicationRef`
- Multiple Deployments can reference the same Application (one-to-many relationship)
- Examples: staging deployment, production deployment, feature branch deployments

#### 5.2 Pipeline Orchestration (Future Implementation)
- Integration with Tekton PipelineRuns
- Git webhook handling for automatic builds  
- Image building and registry management

#### 5.3 Status Tracking
- Pipeline run status monitoring
- Build artifact tracking (images, digests)
- Deployment history maintenance

#### 5.4 Phases Management
```
Initializing → Running → Succeeded/Failed → Waiting
```

## Security Model

### RBAC Structure

**Operator Level:**
- Cluster-admin permissions via ClusterRole `manager-role`
- Full access to all resources: `apiGroups: ["*"], resources: ["*"], verbs: ["*"]`
- Bound to ServiceAccount `controller-manager` in `system` namespace

**Project Level:**
- Each project gets dedicated namespace for isolation
- Service account with admin rights within project namespace only
- No cross-project access by default

**Tekton Integration:**
- Shared role `tekton-tasks-reader` with read-only access to Tekton tasks
- Individual role bindings per project for access control
- Enables CI/CD without cluster-admin privileges

### Resource Isolation

**Namespace Boundaries:**
- Hard isolation between projects via Kubernetes namespaces
- Resource quotas and limits can be applied per namespace
- Network policies can restrict inter-project communication

**UUID-Based Labeling:**
- All resources tagged with project and workspace UUIDs
- Enables tracking and grouping across namespace boundaries
- Supports multi-tenant billing and resource attribution

## Configuration and Customization

### Project Specification Options

**Resource Management:**
```yaml
spec:
  applicationTypes:
    mysql:
      enabled: true
      defaultLimits:
        cpu: "1"
        memory: "2Gi"
        storage: "20Gi"
      resourceBounds:
        min:
          cpu: "0.5"
          memory: "1Gi"
        max:
          cpu: "4"
          memory: "8Gi"
```

**Volume Configuration:**
```yaml
spec:
  volumes:
    maxStorageSize: "100Gi"
```

### Application Type Support

**Supported Types:**
- `MySQL` - Single instance database
- `MySQLCluster` - Multi-node MySQL cluster
- `Postgres` - Single instance PostgreSQL
- `PostgresCluster` - Multi-node PostgreSQL cluster  
- `DockerImage` - Container image deployment
- `GitRepository` - Git-based application with CI/CD

### Webhooks and Validation

**Admission Webhooks:**
- **Project validation webhook**: Ensures required UUID labels and validates UUID format
- **Application validation webhook**: Enforces naming conventions and application-specific rules
  - Validates application names follow pattern: `project-<project-slug>-app-<app-slug>-kibaship-com`
  - **GitRepository validation**: 
    - When `PublicAccess: false` (default), validates that `SecretRef` is provided
    - When `PublicAccess: true`, `SecretRef` validation is skipped
    - Repository format validation: `^[a-zA-Z0-9._-]+/[a-zA-Z0-9._-]+$`
- **Deployment validation webhook**: Enforces naming pattern for deployments
- All webhooks use regex patterns and prevent creation of invalid resources

## Troubleshooting

### Common Issues

**Project Creation Failures:**
1. Missing required UUID labels - check label presence and format
2. RBAC permissions - verify operator has cluster-admin access
3. Tekton namespace missing - ensure Tekton is installed

**Application Deployment Issues:**
1. Invalid project reference - ensure referenced project exists
2. **Git access configuration**:
   - For private repositories (`PublicAccess: false`), ensure SecretRef is provided and secret exists in project namespace
   - For public repositories (`PublicAccess: true`), SecretRef is optional but can be provided for authenticated access
   - Verify secret contains valid git access token with appropriate repository permissions
3. Repository format - ensure repository follows pattern `<org-name>/<repo-name>`
4. Resource limits - check project application type configurations

**Cleanup Issues:**
1. Finalizers stuck - check for dependent resources preventing deletion
2. RBAC cleanup failed - verify operator permissions for cross-namespace operations

### Monitoring and Observability

**Status Fields:**
- Project: `status.phase`, `status.namespaceName`, `status.message`
- Application: `status.phase`, `status.conditions`
- Deployment: `status.phase`, `status.currentPipelineRun`

**Resource Labels:**
All managed resources are labeled with:
- `platform.kibaship.com/managed-by: kibaship-operator`
- `platform.kibaship.com/project-name: {name}`
- UUID tracking labels

## Future Enhancements

### Planned Features

1. **Complete Deployment Controller Implementation**
   - Full Tekton PipelineRun integration
   - Git webhook support
   - Container registry management

2. **Resource Quota Management**
   - Per-project resource limits
   - Multi-tier pricing support
   - Usage monitoring and reporting

3. **Network Policy Integration**
   - Automatic network isolation between projects
   - Configurable ingress/egress rules
   - Service mesh integration

4. **Backup and Disaster Recovery**
   - Automated project backup
   - Cross-cluster project migration
   - Point-in-time recovery capabilities

## API Reference

### Project Resource

**Required Labels:**
- `platform.kibaship.com/uuid` - Project UUID (required)
- `platform.kibaship.com/workspace-uuid` - Workspace UUID (optional)

**Status Fields:**
- `phase`: "Pending" | "Ready" | "Failed"
- `namespaceName`: Generated namespace name
- `message`: Human-readable status message
- `lastReconcileTime`: Timestamp of last successful reconciliation

### Application Resource

**Naming Requirements:**
- **Format**: `project-<project-slug>-app-<app-slug>-kibaship-com`
- **Example**: `project-mystore-app-frontend-kibaship-com`
- **Validation**: Enforced via admission webhooks

**Required Fields:**
- `spec.projectRef.name`: Reference to parent project
- `spec.type`: Application type enum value

**Type-Specific Configuration:**
- Each application type has its own configuration section
- Only the section matching `spec.type` needs to be populated

**GitRepository Configuration:**
- `provider` (required): Git provider (github.com, gitlab.com, bitbucket.com)
- `repository` (required): Repository name in format `<org-name>/<repo-name>`
- `publicAccess` (optional, default: false): Controls SecretRef validation
  - `true`: Repository is publicly accessible, SecretRef is optional
  - `false`: Repository requires authentication, SecretRef is mandatory
- `secretRef` (conditional): Reference to secret containing git access token
  - Required when `publicAccess: false`
  - Optional when `publicAccess: true`
- `branch` (optional): Git branch to use (defaults to main/master)
- `path` (optional): Path within repository (defaults to root)
- `rootDirectory` (optional, default: "./"): Root directory for the application
- `buildCommand` (optional): Command to build the application
- `startCommand` (optional): Command to start the application
- `env` (optional): Reference to secret containing environment variables
- `spaOutputDirectory` (optional): Output directory for SPA builds

### Deployment Resource

**Naming Requirements:**
- **Format**: `project-<project-slug>-app-<app-slug>-deployment-<deployment-slug>-kibaship-com`
- **Example**: `project-mystore-app-frontend-deployment-staging-kibaship-com`
- **Validation**: Enforced via admission webhooks
- **Consistency**: Project and app slugs must match the referenced Application

**Required Fields:**
- `spec.applicationRef.name`: Reference to parent application (must follow Application naming format)

**Status Tracking:**
- Pipeline run history (last 5 runs)
- Current and last successful run information
- Built image metadata and digests

This documentation provides a complete technical overview of the KibaShip Operator's functionality, architecture, and operational characteristics.
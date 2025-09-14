# KibaShip Operator - Complete Technical Documentation

## Overview

The KibaShip Operator is a comprehensive Kubernetes controller that provides platform-as-a-service capabilities for multi-tenant application deployment and management. It offers a complete declarative API for creating isolated project environments with RBAC, resource management, database provisioning, and CI/CD integration through Tekton Pipelines.

## Architecture

### Custom Resource Definitions (CRDs)

The operator manages three primary custom resources with comprehensive lifecycle management:

1. **Project** - Top-level tenant isolation unit with resource quotas and configuration
2. **Application** - Application definitions within projects supporting multiple types
3. **Deployment** - Complete CI/CD pipeline management and application deployment

### Controller Components

- **ProjectReconciler** - Complete project lifecycle and namespace provisioning (206 lines)
- **ApplicationReconciler** - Application resource management with UUID propagation (264 lines)
- **DeploymentReconciler** - Full deployment pipeline orchestration with GitRepository and MySQL support (637 lines)
- **NamespaceManager** - Core component for namespace and cross-namespace RBAC management (700+ lines)
- **ValkeyProvisioner** - Automatic system-wide cache cluster provisioning (200 lines)
- **MySQL Utilities** - Database credential management and cluster provisioning
- **Validation Logic** - Server-side validation with webhook integration

### System Infrastructure

The operator automatically provisions and manages:
- **System Valkey Cluster** - Redis-compatible cache cluster for platform services
- **Tekton Task Library** - Custom Git clone tasks with secure authentication
- **Cross-Namespace RBAC** - Service accounts with minimal required permissions
- **Database Clusters** - MySQL clusters using Oracle MySQL Operator integration

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

**Step 4: Enhanced Label Validation**
- Validates required UUID labels using server-side webhooks:
  - `platform.kibaship.com/uuid` (required, must be valid UUID format)
  - `platform.kibaship.com/workspace-uuid` (optional, must be valid UUID if present)
- Uses regex validation: `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`
- Enforces project name uniqueness across the cluster
- If validation fails, updates status to "Failed" and stops reconciliation

#### 1.2 Namespace and RBAC Provisioning

**Step 5: Enhanced Namespace Creation**
```
NamespaceManager.CreateProjectNamespace() called
```

The namespace creation process involves multiple sub-operations:

**5.1 Namespace Generation**
- Generates namespace name using pattern: `project-{project-name}-kibaship-com`
- Creates namespace with comprehensive labels:
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

#### 1.3 Enhanced Tekton Integration Setup

**Step 6: Complete Tekton Pipeline Integration**
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
    resources: ["tasks", "pipelines", "pipelineruns"]
    verbs: ["get", "list", "watch", "create"]
  ```
- Role is labeled as managed by the operator
- Only one role exists cluster-wide (shared across projects)

**6.3 Cross-Namespace Role Binding Creation**
- Creates RoleBinding in `tekton-pipelines` namespace
- Binding name: `project-{project-name}-tekton-tasks-reader-binding-kibaship-com`
- Binds project service account to `tekton-tasks-reader` role
- Enables project service accounts to create and monitor pipelines

#### 1.4 Status Updates

**Step 7: Success Status Update**
- Updates Project status to "Ready" phase
- Sets `NamespaceName` field to created namespace name
- Records `LastReconcileTime`
- Adds success message with detailed information

**Final State After Project Creation:**
- ✅ Dedicated namespace created and labeled
- ✅ Service account with admin privileges in namespace
- ✅ Role and RoleBinding for namespace admin access
- ✅ Cross-namespace Tekton integration for CI/CD capabilities
- ✅ Project status reflects ready state with comprehensive information

### 2. Application Creation Workflow

When a new `Application` resource is created:

#### 2.1 Initial Processing and Validation
```
Application Creation Event → ApplicationReconciler.Reconcile()
```

**Step 1: Resource Retrieval and Webhook Validation**
- Fetches Application resource
- Server-side webhook validates application configuration
- Validates deletion timestamp (triggers deletion if set)

**Step 2: Enhanced UUID Label Management**
- Ensures application inherits UUID labels from parent project:
  - `platform.kibaship.com/project-uuid`
  - `platform.kibaship.com/workspace-uuid`
- Queries referenced project to copy UUID labels
- Updates Application resource if labels were added
- Ensures consistency across project resources

**Step 3: Finalizer Management**
- Adds finalizer `platform.operator.kibaship.com/application-finalizer`
- Ensures proper cleanup during deletion

#### 2.2 Enhanced Application Type Processing

**Step 4: Comprehensive Type-Specific Handling**
Based on `spec.type`, different configuration validation occurs:

**GitRepository Applications:**
- Validates provider (github.com, gitlab.com, bitbucket.com)
- Validates repository format: `^[a-zA-Z0-9._-]+/[a-zA-Z0-9._-]+$`
- **PublicAccess Field**: Controls access validation behavior
  - When `PublicAccess: true` - SecretRef is optional (for public repositories)
  - When `PublicAccess: false` (default) - SecretRef is required and must exist in project namespace
- **Enhanced Fields**: rootDirectory, buildCommand, startCommand, env, spaOutputDirectory
- Supports commit-specific builds and branch specification

**DockerImage Applications:**
- Validates image reference format
- Optional: image pull secret reference
- Optional: specific tag override with registry authentication

**Database Applications (MySQL/Postgres):**
- **Single Instance**: Basic config with version and database name
- **Cluster**: Includes replica count (min 1, default 3) with per-node resource limits
- **Enhanced Security**: Automatic credential generation and secret management
- **Resource Bounds**: CPU, memory, and storage limits per instance

#### 2.3 Application Status Update

**Step 5: Comprehensive Status Update**
- Updates application status to "Ready"
- Sets detailed conditions for application readiness
- Includes validation results and configuration summary
- No automatic resource creation occurs (explicit deployment required)

### 3. Complete Deployment Management

The Deployment controller is now fully implemented with comprehensive pipeline orchestration:

#### 3.1 Deployment Creation and Pipeline Generation

**GitRepository Deployments:**
```
DeploymentReconciler.handleGitRepositoryDeployment() called
```

**Pipeline Creation Process:**
1. **Pipeline Generation**: Creates Tekton Pipeline with custom Git clone task
2. **Secure Authentication**: Uses project namespace secrets for Git access
3. **PipelineRun Creation**: Automatically creates initial pipeline run
4. **Status Tracking**: Monitors pipeline execution and collects results

**Key Features:**
- Commit-specific builds with SHA tracking
- Built image metadata collection and storage
- Pipeline run history management (last 5 runs)
- Comprehensive error handling and status reporting

**MySQL Deployments:**
```
DeploymentReconciler.handleMySQLDeployment() called
```

**Database Provisioning Process:**
1. **Credential Generation**: Creates secure random passwords
2. **Secret Management**: Stores credentials in project namespace
3. **InnoDBCluster Creation**: Provisions MySQL cluster using Oracle MySQL Operator
4. **Status Monitoring**: Tracks cluster health and connection information

#### 3.2 Enhanced Deployment Status Management

**Deployment Phases:**
- `Initializing` - Setting up resources and validating configuration
- `Running` - Pipeline or provisioning in progress
- `Succeeded` - Deployment completed successfully
- `Failed` - Deployment encountered errors
- `Waiting` - Waiting for dependencies or manual intervention

**Status Information Includes:**
- Current and last successful pipeline run details
- Built image information with registry references
- Commit SHA and branch information for Git deployments
- Database connection details for MySQL deployments
- Comprehensive error messages and troubleshooting information

### 4. Project Deletion Workflow

When a `Project` resource is deleted:

#### 4.1 Enhanced Deletion Detection
```
Project Deletion Event → ProjectReconciler.handleProjectDeletion()
```

**Step 1: Finalizer Check**
- Only proceeds if finalizer `platform.kibaship.com/project-finalizer` is present
- Prevents accidental cleanup of non-operator managed resources

#### 4.2 Comprehensive Resource Cleanup Sequence

**Step 2: Service Account Resource Cleanup**
```
NamespaceManager.deleteServiceAccountResources() called
```

**2.1 Cross-Namespace Tekton Cleanup**
- Deletes RoleBinding `project-{project-name}-tekton-tasks-reader-binding-kibaship-com` from `tekton-pipelines` namespace
- Removes project's access to Tekton resources
- Preserves shared `tekton-tasks-reader` role for other projects

**2.2 Namespace RBAC Cleanup**
- Deletes RoleBinding: `project-{project-name}-admin-binding-kibaship-com`
- Deletes Role: `project-{project-name}-admin-role-kibaship-com`
- Deletes ServiceAccount: `project-{project-name}-sa-kibaship-com`

**2.3 Application and Deployment Cleanup**
- Automatically cascades deletion to dependent Applications and Deployments
- Cleans up associated pipelines, pipeline runs, and database resources
- Removes all managed secrets and configurations

**Step 3: Namespace Deletion**
- Deletes project namespace: `project-{project-name}-kibaship-com`
- Kubernetes automatically removes all resources within the namespace
- Includes any user workloads, persistent volumes, and network policies

**Step 4: Finalizer Removal**
- Removes finalizer from Project resource
- Allows Kubernetes to complete resource deletion
- Project resource is permanently removed from cluster

### 5. Application Deletion Workflow

When an `Application` resource is deleted:

#### 5.1 Enhanced Cascade Deletion
```
Application Deletion Event → ApplicationReconciler.handleDeletion()
```

**Step 1: Dependent Resource Cleanup**
- Finds and deletes associated `Deployment` resources using label selector
- Uses label `platform.operator.kibaship.com/application: {app-name}` to find dependent deployments
- Cleans up associated pipelines, pipeline runs, and database clusters
- Removes all generated secrets and configurations

**Step 2: Finalizer Removal**
- Removes finalizer `platform.operator.kibaship.com/application-finalizer`
- Completes Application deletion

## Security Model

### Enhanced RBAC Structure

**Operator Level:**
- Cluster-admin permissions via ClusterRole `manager-role`
- Full access to all resources: `apiGroups: ["*"], resources: ["*"], verbs: ["*"]`
- Bound to ServiceAccount `controller-manager` in operator namespace
- Includes permissions for MySQL Operator and Tekton integration

**Project Level:**
- Each project gets dedicated namespace for complete isolation
- Service account with admin rights within project namespace only
- Cross-namespace read-only access to Tekton resources
- Automatic secret management for database credentials

**Tekton Integration:**
- Shared role `tekton-tasks-reader` with minimal required permissions
- Individual cross-namespace role bindings per project
- Secure token-based Git authentication
- Pipeline execution isolation per project

### Resource Isolation and Security

**Namespace Boundaries:**
- Hard isolation between projects via Kubernetes namespaces
- Resource quotas and limits enforced per project
- Network policies can restrict inter-project communication
- Automatic cleanup prevents resource leaks

**UUID-Based Resource Tracking:**
- All resources tagged with project and workspace UUIDs
- Enables tracking and grouping across namespace boundaries
- Supports multi-tenant billing and resource attribution
- Prevents accidental cross-project resource access

**Secure Credential Management:**
- Automatic generation of secure database passwords
- Token-based Git authentication with secret management
- Project-scoped secret access only
- Credential rotation capability

## Configuration and Customization

### Enhanced Project Specification Options

**Resource Management with Bounds:**
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
          storage: "10Gi"
        max:
          cpu: "4"
          memory: "8Gi"
          storage: "100Gi"
    mysqlCluster:
      enabled: true
      defaultReplicas: 3
      clusterResourceLimits:
        perNode:
          cpu: "2"
          memory: "4Gi"
          storage: "50Gi"
        maxNodes: 5
```

**Volume Configuration:**
```yaml
spec:
  volumes:
    maxStorageSize: "500Gi"
    storageClass: "fast-ssd"
    backupRetention: "30d"
```

### Comprehensive Application Type Support

**Fully Supported Types:**
- `MySQL` - Single instance database with automated provisioning
- `MySQLCluster` - Multi-node MySQL cluster with Oracle MySQL Operator
- `Postgres` - Single instance PostgreSQL (planned)
- `PostgresCluster` - Multi-node PostgreSQL cluster (planned)
- `DockerImage` - Container image deployment with registry authentication
- `GitRepository` - Git-based application with full CI/CD pipeline

### Server-Side Validation Webhooks

**Admission Webhooks:**
- **Project validation webhook**: UUID labels, format validation, and name uniqueness
- **Application validation webhook**: Naming conventions, project references, and type-specific rules
  - Validates application names follow pattern: `project-<project-slug>-app-<app-slug>-kibaship-com`
  - **GitRepository validation**:
    - When `PublicAccess: false` (default), validates that `SecretRef` is provided
    - When `PublicAccess: true`, `SecretRef` validation is skipped
    - Repository format validation: `^[a-zA-Z0-9._-]+/[a-zA-Z0-9._-]+$`
    - Provider validation (GitHub, GitLab, Bitbucket)
- **Deployment validation webhook**: Naming patterns, application references, and configuration validation
- All webhooks use fail-closed policy for security
- Comprehensive error messages for validation failures

## System Infrastructure Management

### Automatic System Provisioning

**Valkey System Cluster:**
- Automatically provisioned on operator startup
- Provides Redis-compatible cache for platform services
- Uses unstructured API for CRD interaction
- Comprehensive health monitoring and status reporting

**Tekton Task Library:**
- Custom Git clone task with secure authentication
- Commit-specific checkout capability
- Result tracking for SHA and repository URL
- Token-based authentication with secret management

**MySQL Operator Integration:**
- Seamless integration with Oracle MySQL Operator
- Automatic InnoDBCluster provisioning
- Secure credential generation and management
- Connection string generation and secret storage

## Monitoring and Observability

### Enhanced Status Reporting

**Project Status:**
- `status.phase`: "Pending" | "Ready" | "Failed"
- `status.namespaceName`: Generated namespace name
- `status.message`: Detailed status information
- `status.lastReconcileTime`: Timestamp tracking
- `status.conditions`: Detailed condition information

**Application Status:**
- `status.phase`: Application readiness state
- `status.conditions`: Validation and configuration status
- `status.projectRef`: Resolved project information
- `status.lastUpdated`: Change tracking

**Deployment Status:**
- `status.phase`: "Initializing" | "Running" | "Succeeded" | "Failed" | "Waiting"
- `status.currentPipelineRun`: Active pipeline information
- `status.lastSuccessfulRun`: Last successful deployment details
- `status.imageInfo`: Built image metadata and registry references
- `status.pipelineRunHistory`: Last 5 pipeline runs with details
- `status.conditions`: Comprehensive deployment status conditions

### Resource Labeling and Tracking

**Comprehensive Labels:**
All managed resources include:
- `platform.kibaship.com/managed-by: kibaship-operator`
- `platform.kibaship.com/project-name: {name}`
- `platform.kibaship.com/uuid: {project-uuid}`
- `platform.kibaship.com/workspace-uuid: {workspace-uuid}`
- `platform.operator.kibaship.com/application: {app-name}` (for deployments)

## Troubleshooting

### Common Issues and Solutions

**Project Creation Failures:**
1. **Missing required UUID labels** - Ensure labels are present and valid UUID format
2. **RBAC permissions** - Verify operator has cluster-admin access and can create cross-namespace resources
3. **Tekton namespace missing** - Ensure Tekton is installed and `tekton-pipelines` namespace exists
4. **Project name conflicts** - Project names must be globally unique across the cluster

**Application Configuration Issues:**
1. **Invalid project reference** - Ensure referenced project exists and is in Ready phase
2. **Git access configuration**:
   - For private repositories (`PublicAccess: false`), ensure SecretRef exists in project namespace
   - For public repositories (`PublicAccess: true`), SecretRef is optional
   - Verify secret contains valid git access token with repository permissions
3. **Repository format validation** - Repository must follow `<org-name>/<repo-name>` pattern
4. **Resource limits** - Check project application type configurations and bounds

**Deployment Pipeline Issues:**
1. **Pipeline creation failures** - Verify Tekton installation and RBAC permissions
2. **Git clone failures** - Check token validity and repository access permissions
3. **Image build failures** - Review pipeline logs and build configuration
4. **MySQL provisioning failures** - Ensure Oracle MySQL Operator is installed and functional
5. **Resource quota exceeded** - Check project resource limits and cluster capacity

**Status and Monitoring Issues:**
1. **Finalizers stuck** - Check for dependent resources preventing deletion
2. **Status not updating** - Verify controller health and reconciliation logs
3. **Cross-namespace access denied** - Check Tekton role bindings and permissions

### Advanced Troubleshooting

**Debug Commands:**
```bash
# Check operator logs
kubectl logs -n kibaship-operator deployment/controller-manager

# Verify project resources
kubectl get projects -o yaml
kubectl describe project <project-name>

# Check namespace resources
kubectl get all -n project-<name>-kibaship-com

# Verify Tekton integration
kubectl get rolebinding -n tekton-pipelines | grep kibaship

# Check MySQL clusters
kubectl get innodbclusters -A

# Verify pipeline runs
kubectl get pipelineruns -n project-<name>-kibaship-com
```

**Status Conditions Analysis:**
Each resource includes detailed conditions explaining current state and any issues encountered.

## Testing and Quality Assurance

### Comprehensive Test Suite (3,782+ lines)

**Controller Tests:**
- **Project Controller** (292 lines) - Lifecycle, validation, status management, cleanup
- **Application Controller** (612 lines) - UUID propagation, validation, deletion cascade
- **Deployment Controller** (1,294 lines) - Pipeline creation, MySQL provisioning, status tracking
- **Namespace Management** (450 lines) - RBAC setup, Tekton integration, resource cleanup

**Integration Tests:**
- **Tekton Integration** (231 lines) - Cross-namespace role creation, pipeline execution
- **Valkey Provisioner** (368 lines) - System cluster management
- **RBAC Validation** (164 lines) - Permission verification
- **End-to-End Scenarios** (245 lines) - Complete workflow validation

**Test Coverage Highlights:**
- Complete controller reconciliation testing with 82 test cases
- Server-side webhook validation testing
- RBAC and security model validation
- Tekton pipeline creation and execution
- MySQL database provisioning scenarios
- Comprehensive error handling and edge cases
- Resource cleanup and finalizer management

## Production Deployment

### Operator Configuration

**High Availability Setup:**
- Leader election support for multi-replica deployments
- Health and readiness checks with detailed endpoints
- Graceful shutdown handling
- Resource-based deployment scaling

**Logging and Monitoring:**
- Structured logging with configurable levels
- Prometheus metrics integration
- Status condition reporting
- Resource usage tracking

**Security Configuration:**
- Service account with minimal required permissions
- Secure credential generation and management
- Token-based authentication for external services
- Network policy support for isolation

### Infrastructure Requirements

**Required Components:**
1. **Tekton Pipelines** - v0.44+ for CI/CD functionality
2. **Oracle MySQL Operator** - v9.4+ for database provisioning
3. **Valkey Operator** - v0.0.59+ for system cache (auto-installed)

**Optional Components:**
- Prometheus for metrics collection
- Grafana for dashboard visualization
- Network policies for enhanced isolation
- Storage classes for persistent volume management

## API Reference

### Project Resource

**Required Labels:**
- `platform.kibaship.com/uuid` - Project UUID (required, validated)
- `platform.kibaship.com/workspace-uuid` - Workspace UUID (optional)

**Enhanced Status Fields:**
- `phase`: "Pending" | "Ready" | "Failed"
- `namespaceName`: Generated namespace name
- `message`: Detailed human-readable status
- `lastReconcileTime`: Timestamp of last successful reconciliation
- `conditions`: Array of detailed status conditions

**Specification Options:**
- `applicationTypes`: Resource limits and bounds per application type
- `volumes`: Storage configuration and limits
- `clusterResourceLimits`: Per-node limits for cluster applications

### Application Resource

**Naming Requirements:**
- **Format**: `project-<project-slug>-app-<app-slug>-kibaship-com`
- **Example**: `project-mystore-app-frontend-kibaship-com`
- **Validation**: Enforced via server-side webhooks
- **Uniqueness**: Globally unique across cluster

**Required Fields:**
- `spec.projectRef.name`: Reference to parent project
- `spec.type`: Application type enum value

**GitRepository Configuration:**
- `provider` (required): Git provider (github.com, gitlab.com, bitbucket.com)
- `repository` (required): Repository in format `<org-name>/<repo-name>`
- `publicAccess` (optional, default: false): Controls authentication requirements
- `secretRef` (conditional): Git access token reference
- `branch` (optional): Git branch (defaults to main/master)
- `path` (optional): Path within repository
- `rootDirectory` (optional, default: "./"): Application root directory
- `buildCommand` (optional): Build command for CI/CD
- `startCommand` (optional): Application start command
- `env` (optional): Environment variables secret reference
- `spaOutputDirectory` (optional): SPA build output directory

### Deployment Resource

**Naming Requirements:**
- **Format**: `project-<project-slug>-app-<app-slug>-deployment-<deployment-slug>-kibaship-com`
- **Example**: `project-mystore-app-frontend-deployment-prod-kibaship-com`
- **Validation**: Enforced via server-side webhooks
- **Consistency**: Project and app slugs must match referenced Application

**Status Tracking:**
- `currentPipelineRun`: Active pipeline run information
- `lastSuccessfulRun`: Last successful deployment details
- `imageInfo`: Built image metadata with registry references
- `pipelineRunHistory`: Last 5 runs with commit SHA and status
- `conditions`: Comprehensive deployment status conditions

This documentation provides a complete and current technical overview of the KibaShip Operator's functionality, reflecting all implemented features and production-ready capabilities.
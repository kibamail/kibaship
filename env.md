# Environment CRD Implementation Plan

## Executive Summary

This document outlines the comprehensive plan to introduce the **Environment CRD** into the KibaShip operator. This is a significant architectural change that reorganizes the resource hierarchy from:

**Current:** `Project → Application → Deployment`

**New:** `Project → Environment → Application → Deployment`

## Current Architecture Analysis

### Existing CRD Hierarchy

```
Project (Cluster-scoped)
  ├── Creates Namespace (project-{slug}-kibaship-com)
  └── Contains Applications (namespace-scoped)
      ├── References Project via spec.projectRef
      ├── Has label: platform.kibaship.com/project-uuid
      └── Contains Deployments
          ├── References Application via spec.applicationRef
          ├── Has spec.environmentName (string, defaults to "production")
          └── Creates resources in Project's namespace
```

### Current Labels System

- `platform.kibaship.com/uuid` - Resource UUID
- `platform.kibaship.com/slug` - Resource slug
- `platform.kibaship.com/workspace-uuid` - Workspace UUID (Projects)
- `platform.kibaship.com/project-uuid` - Project UUID (Applications, Deployments, ApplicationDomains)
- `platform.kibaship.com/application-uuid` - Application UUID (Deployments, ApplicationDomains)
- `platform.kibaship.com/deployment-uuid` - Deployment UUID (ApplicationDomains)

### Current Controllers

1. **ProjectController** (`internal/controller/project_controller.go`)

   - Creates namespace for project
   - Creates registry credentials, CA certs, docker config
   - Sets project status to Ready

2. **ApplicationController** (`internal/controller/application_controller.go`)

   - Validates application belongs to project
   - Manages application environments (currently stored in `spec.environments[]`)
   - Creates environment secrets (`{app-name}-env-{env-name}`)
   - Creates ApplicationDomains for GitRepository apps
   - Sets finalizer for cascading deletes

3. **DeploymentController** (`internal/controller/deployment_controller.go`)

   - References `spec.environmentName` (string field)
   - Creates Tekton pipelines and pipeline runs
   - TODO: Database deployment management removed - will be reimplemented
   - Tracks deployment phase

4. **ApplicationDomainController** (`internal/controller/applicationdomain_controller.go`)
   - Manages domains for applications
   - Creates certificates and ingress resources

## Target Architecture

### New CRD Hierarchy

```
Project (Cluster-scoped)
  ├── Creates Namespace (project-{slug}-kibaship-com)
  └── Contains Environments (namespace-scoped, NEW)
      ├── Auto-created "production" environment on Project creation
      ├── References Project via spec.projectRef
      ├── Has label: platform.kibaship.com/project-uuid
      └── Contains Applications (namespace-scoped)
          ├── References Environment via spec.environmentRef
          ├── Has label: platform.kibaship.com/environment-uuid
          └── Contains Deployments
              ├── References Application via spec.applicationRef
              └── Inherits environment from Application
```

### New Labels System

- `platform.kibaship.com/uuid` - Resource UUID
- `platform.kibaship.com/slug` - Resource slug
- `platform.kibaship.com/workspace-uuid` - Workspace UUID (Projects)
- `platform.kibaship.com/project-uuid` - Project UUID (Environments, Applications, Deployments, ApplicationDomains)
- **`platform.kibaship.com/environment-uuid`** - Environment UUID (Applications, Deployments, ApplicationDomains) **[NEW]**
- `platform.kibaship.com/application-uuid` - Application UUID (Deployments, ApplicationDomains)
- `platform.kibaship.com/deployment-uuid` - Deployment UUID (ApplicationDomains)

## Detailed Implementation Plan

### Phase 1: Create Environment CRD

#### 1.1 Create Environment Types File

**File:** `api/v1alpha1/environment_types.go`

**Structure:**

```go
type EnvironmentSpec struct {
    // ProjectRef references the Project this environment belongs to
    // +kubebuilder:validation:Required
    ProjectRef corev1.LocalObjectReference `json:"projectRef"`

    // Description of the environment (optional)
    // +optional
    Description string `json:"description,omitempty"`


    // Variables contains environment-specific configuration variables
    // +optional
    Variables map[string]string `json:"variables,omitempty"`
}

type EnvironmentStatus struct {
    // Phase represents the current phase of the environment
    // +optional
    Phase string `json:"phase,omitempty"`

    // ApplicationCount tracks number of applications in this environment
    // +optional
    ApplicationCount int32 `json:"applicationCount,omitempty"`

    // SecretReady indicates if environment secret exists
    // +optional
    SecretReady bool `json:"secretReady,omitempty"`

    // Conditions represent the latest available observations
    // +optional
    Conditions []metav1.Condition `json:"conditions,omitempty"`

    // Message provides additional information
    // +optional
    Message string `json:"message,omitempty"`
}
```

**Validations:**

- Must have required labels: `platform.kibaship.com/uuid`, `platform.kibaship.com/slug`, `platform.kibaship.com/project-uuid`
- Name must follow format: `environment-{slug}-kibaship-com`
- UUID and slug validations via webhook

**Webhook:** Implement `ValidateCreate`, `ValidateUpdate`, `ValidateDelete` following existing patterns

**Files to create:**

- `api/v1alpha1/environment_types.go`

#### 1.2 Create Environment Controller

**File:** `internal/controller/environment_controller.go`

**Responsibilities:**

1. Add finalizer `platform.operator.kibaship.com/environment-finalizer`
2. Validate environment belongs to valid project
3. Create environment secret if needed
4. Update environment status (phase, application count, etc.)
5. Handle deletion: Delete all Applications when Environment is deleted (cascading delete)
6. Emit webhooks for environment status changes

**Key Functions:**

- `Reconcile()` - Main reconciliation loop
- `handleDeletion()` - Cascade delete all applications
- `deleteAssociatedApplications()` - Find and delete apps by environment label
- `ensureEnvironmentSecret()` - Create secret for environment variables
- `updateEnvironmentStatus()` - Update status fields
- `ensureUUIDLabels()` - Validate and set labels

**Owner References:**

- Environment → Project (for cleanup)
- Applications → Environment (for cascading deletes)

**Files to create:**

- `internal/controller/environment_controller.go`
- `internal/controller/environment_controller_test.go`

### Phase 2: Update Application CRD

#### 2.1 Modify Application Spec

**File:** `api/v1alpha1/application_types.go`

**Changes:**

```go
type ApplicationSpec struct {
    // REMOVE: ProjectRef corev1.LocalObjectReference

    // ADD: EnvironmentRef references the Environment this application belongs to
    // +kubebuilder:validation:Required
    EnvironmentRef corev1.LocalObjectReference `json:"environmentRef"`

    // Type defines the type of application (unchanged)
    Type ApplicationType `json:"type"`

    // REMOVE: Environments []ApplicationEnvironment
    // (This was the old way of managing environments within an application)

    // ... rest remains the same ...
}
```

**Label Changes:**

- Add `platform.kibaship.com/environment-uuid` label
- Keep `platform.kibaship.com/project-uuid` (inherited from environment)

**Validation Changes:**

- Update webhook to require `environment-uuid` label
- Validate `environmentRef` points to existing Environment

**Files to modify:**

- `api/v1alpha1/application_types.go` (remove ProjectRef, add EnvironmentRef, remove Environments array)

#### 2.2 Update Application Controller

**File:** `internal/controller/application_controller.go`

**Changes:**

1. **Remove:**

   - `ensureDefaultEnvironment()` - No longer needed
   - `reconcileEnvironmentSecrets()` - Move to Environment controller
   - `cleanupRemovedEnvironmentSecrets()` - Move to Environment controller
   - `updateEnvironmentStatus()` - Move to Environment controller
   - Environment-related status tracking

2. **Modify:**

   - `ensureUUIDLabels()` - Get environment UUID and project UUID from environment
   - `getProjectUUID()` - Change to get from environment instead of direct project ref
   - Validation logic to check environment exists

3. **Add:**
   - `getEnvironmentUUID()` - Extract environment UUID from environment resource

**Key Changes:**

```go
// OLD:
app.Spec.ProjectRef.Name

// NEW:
// Get environment first, then get project from environment
environment := &platformv1alpha1.Environment{}
r.Get(ctx, types.NamespacedName{Name: app.Spec.EnvironmentRef.Name, Namespace: app.Namespace}, environment)
projectRef := environment.Spec.ProjectRef.Name
```

**Files to modify:**

- `internal/controller/application_controller.go`
- `internal/controller/application_controller_test.go`

### Phase 3: Update Deployment CRD

#### 3.1 Modify Deployment Spec

**File:** `api/v1alpha1/deployment_types.go`

**Changes:**

```go
type DeploymentSpec struct {
    // ApplicationRef references the Application this deployment belongs to
    // +kubebuilder:validation:Required
    ApplicationRef corev1.LocalObjectReference `json:"applicationRef"`

    // REMOVE: EnvironmentName string
    // Environment is now inherited from Application

    // ... rest remains the same ...
}
```

**Label Changes:**

- Add `platform.kibaship.com/environment-uuid` label (inherited from application)

**Files to modify:**

- `api/v1alpha1/deployment_types.go` (remove environmentName field)

#### 3.2 Update Deployment Controller

**File:** `internal/controller/deployment_controller.go`

**Changes:**

1. Remove all references to `deployment.Spec.EnvironmentName`
2. Get environment UUID from application's labels
3. Update pipeline creation to use environment context if needed

**Files to modify:**

- `internal/controller/deployment_controller.go`
- `internal/controller/deployment_controller_test.go`

### Phase 4: Update Project Controller

#### 4.1 Modify Project Controller

**File:** `internal/controller/project_controller.go`

**Add Auto-Creation of Production Environment:**

```go
// After namespace creation and before status update
func (r *ProjectReconciler) ensureDefaultEnvironment(ctx context.Context, project *platformv1alpha1.Project, namespace string) error {
    // Check if production environment already exists
    productionEnvName := "environment-production-kibaship-com"
    existingEnv := &platformv1alpha1.Environment{}
    err := r.Get(ctx, types.NamespacedName{
        Name: productionEnvName,
        Namespace: namespace,
    }, existingEnv)

    if err == nil {
        // Production environment already exists
        return nil
    }

    if !errors.IsNotFound(err) {
        return fmt.Errorf("failed to check for production environment: %w", err)
    }

    // Create production environment
    productionEnv := &platformv1alpha1.Environment{
        ObjectMeta: metav1.ObjectMeta{
            Name: productionEnvName,
            Namespace: namespace,
            Labels: map[string]string{
                validation.LabelResourceUUID: validation.GenerateUUID(),
                validation.LabelResourceSlug: "production",
                validation.LabelProjectUUID: project.Labels[validation.LabelResourceUUID],
            },
        },
        Spec: platformv1alpha1.EnvironmentSpec{
            ProjectRef: corev1.LocalObjectReference{Name: project.Name},
            Description: "Default production environment",
        },
    }

    // Set owner reference
    if err := controllerutil.SetControllerReference(project, productionEnv, r.Scheme); err != nil {
        return fmt.Errorf("failed to set controller reference: %w", err)
    }

    if err := r.Create(ctx, productionEnv); err != nil {
        return fmt.Errorf("failed to create production environment: %w", err)
    }

    log.Info("Created default production environment", "environment", productionEnvName)
    return nil
}
```

**Call in Reconcile:**

```go
// After namespace and registry setup, before status update
if err := r.ensureDefaultEnvironment(ctx, &project, namespace.Name); err != nil {
    log.Error(err, "Failed to create default environment")
    r.updateStatusWithError(ctx, &project, fmt.Sprintf("Failed to create default environment: %v", err))
    return ctrl.Result{}, err
}
```

**Files to modify:**

- `internal/controller/project_controller.go`
- `internal/controller/project_controller_test.go`

### Phase 5: Update Validation Package

#### 5.1 Add Environment UUID Label

**File:** `pkg/validation/labels.go`

**Add:**

```go
// LabelEnvironmentUUID is the label key for environment UUID
LabelEnvironmentUUID = "platform.kibaship.com/environment-uuid"
```

**Files to modify:**

- `pkg/validation/labels.go`

#### 5.2 Update Resource Labeler

**File:** `internal/controller/resource_labeler.go`

**Add:**

- `ValidateEnvironmentLabeling()` - Validate environment labels
- Update `ValidateApplicationLabeling()` to check for environment-uuid
- Update `ValidateDeploymentLabeling()` to check for environment-uuid

**Files to modify:**

- `internal/controller/resource_labeler.go`

### Phase 6: Generate CRD Manifests

#### 6.1 Run Code Generation

```bash
# Generate deepcopy, client, etc.
make generate

# Generate CRD manifests
make manifests
```

This will create:

- `config/crd/bases/platform.operator.kibaship.com_environments.yaml`
- Updated application/deployment CRDs

#### 6.2 Update Kustomization

**File:** `config/crd/kustomization.yaml`

**Add:**

```yaml
resources:
  - bases/platform.operator.kibaship.com_projects.yaml
  - bases/platform.operator.kibaship.com_environments.yaml # NEW
  - bases/platform.operator.kibaship.com_applications.yaml
  - bases/platform.operator.kibaship.com_deployments.yaml
  - bases/platform.operator.kibaship.com_applicationdomains.yaml
```

**Files to modify:**

- `config/crd/kustomization.yaml`

### Phase 7: Update Main Entry Point

#### 7.1 Register Environment Controller

**File:** `cmd/main.go`

**Add after other controller setups:**

```go
// Setup Environment controller
if err = (&controller.EnvironmentReconciler{
    Client:   mgr.GetClient(),
    Scheme:   mgr.GetScheme(),
    Notifier: webhooks.NoopNotifier{},
}).SetupWithManager(mgr); err != nil {
    setupLog.Error(err, "unable to create controller", "controller", "Environment")
    os.Exit(1)
}

// Setup Environment webhook
if err = (&platformv1alpha1.Environment{}).SetupWebhookWithManager(mgr); err != nil {
    setupLog.Error(err, "unable to create webhook", "webhook", "Environment")
    os.Exit(1)
}
```

**Files to modify:**

- `cmd/main.go`

### Phase 8: Update All Tests

This is the MOST CRITICAL phase. We need to update every single test to reflect the new hierarchy.

#### 8.1 Test Helper Functions

**File:** `internal/controller/test_helpers.go` (CREATE NEW)

```go
package controller

import (
    "context"
    platformv1alpha1 "github.com/kibamail/kibaship/api/v1alpha1"
    "github.com/kibamail/kibaship/pkg/validation"
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateTestProject creates a test project for use in tests
func CreateTestProject(ctx context.Context, k8sClient client.Client, projectName, projectUUID, projectSlug, workspaceUUID string) (*platformv1alpha1.Project, error) {
    project := &platformv1alpha1.Project{
        ObjectMeta: metav1.ObjectMeta{
            Name: projectName,
            Labels: map[string]string{
                validation.LabelResourceUUID:  projectUUID,
                validation.LabelResourceSlug:  projectSlug,
                validation.LabelWorkspaceUUID: workspaceUUID,
            },
        },
        Spec: platformv1alpha1.ProjectSpec{},
    }
    return project, k8sClient.Create(ctx, project)
}

// CreateTestEnvironment creates a test environment for use in tests
func CreateTestEnvironment(ctx context.Context, k8sClient client.Client, envName, envUUID, envSlug, namespace, projectName, projectUUID string) (*platformv1alpha1.Environment, error) {
    environment := &platformv1alpha1.Environment{
        ObjectMeta: metav1.ObjectMeta{
            Name:      envName,
            Namespace: namespace,
            Labels: map[string]string{
                validation.LabelResourceUUID: envUUID,
                validation.LabelResourceSlug: envSlug,
                validation.LabelProjectUUID:  projectUUID,
            },
        },
        Spec: platformv1alpha1.EnvironmentSpec{
            ProjectRef: corev1.LocalObjectReference{Name: projectName},
        },
    }
    return environment, k8sClient.Create(ctx, environment)
}

// CreateTestApplication creates a test application for use in tests
func CreateTestApplication(ctx context.Context, k8sClient client.Client, appName, appUUID, appSlug, namespace, environmentName, environmentUUID, projectUUID string, appType platformv1alpha1.ApplicationType) (*platformv1alpha1.Application, error) {
    application := &platformv1alpha1.Application{
        ObjectMeta: metav1.ObjectMeta{
            Name:      appName,
            Namespace: namespace,
            Labels: map[string]string{
                validation.LabelResourceUUID:     appUUID,
                validation.LabelResourceSlug:     appSlug,
                validation.LabelEnvironmentUUID:  environmentUUID,
                validation.LabelProjectUUID:      projectUUID,
            },
        },
        Spec: platformv1alpha1.ApplicationSpec{
            EnvironmentRef: corev1.LocalObjectReference{Name: environmentName},
            Type:           appType,
        },
    }
    return application, k8sClient.Create(ctx, application)
}
```

**Files to create:**

- `internal/controller/test_helpers.go`

#### 8.2 Update Project Controller Tests

**File:** `internal/controller/project_controller_test.go`

**Test Cases to Update:**

1. ✅ "should successfully reconcile the resource"

   - After reconciliation, verify production environment was auto-created
   - Check environment has correct labels and owner reference

2. ✅ "should fail validation when platform.kibaship.com/uuid label is missing"

   - No changes needed (project-level validation)

3. ✅ "should fail validation when UUID labels have invalid format"

   - No changes needed (project-level validation)

4. ✅ "should fail when project name conflicts with existing namespace"
   - No changes needed (namespace-level logic)

**New Test Cases:**

- "should auto-create production environment on project creation"
- "should not recreate production environment if it already exists"
- "production environment should have correct owner reference to project"

**Example:**

```go
It("should auto-create production environment on project creation", func() {
    By("Reconciling the project twice")
    controllerReconciler := NewProjectReconciler(k8sClient, k8sClient.Scheme())
    _, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
    Expect(err).NotTo(HaveOccurred())
    _, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
    Expect(err).NotTo(HaveOccurred())

    By("Verifying production environment was created")
    expectedNamespace := NamespacePrefix + resourceName + NamespaceSuffix
    productionEnv := &platformv1alpha1.Environment{}
    err = k8sClient.Get(ctx, types.NamespacedName{
        Name:      "environment-production-kibaship-com",
        Namespace: expectedNamespace,
    }, productionEnv)
    Expect(err).NotTo(HaveOccurred())

    By("Verifying environment has correct labels")
    Expect(productionEnv.Labels[validation.LabelResourceSlug]).To(Equal("production"))
    Expect(productionEnv.Labels[validation.LabelProjectUUID]).To(Equal("550e8400-e29b-41d4-a716-446655440000"))

    By("Verifying environment references project correctly")
    Expect(productionEnv.Spec.ProjectRef.Name).To(Equal(resourceName))
})
```

**Files to modify:**

- `internal/controller/project_controller_test.go`

#### 8.3 Create Environment Controller Tests

**File:** `internal/controller/environment_controller_test.go` (NEW)

**Test Cases:**

1. "should successfully reconcile environment"
2. "should validate required labels"
3. "should fail when environment UUID is invalid"
4. "should fail when project doesn't exist"
5. "should create environment secret"
6. "should delete all applications when environment is deleted"
7. "should update environment status with application count"

**Files to create:**

- `internal/controller/environment_controller_test.go`

#### 8.4 Update Application Controller Tests

**File:** `internal/controller/application_controller_test.go`

**Changes Required:**

1. Create test environment before creating test application
2. Update application spec to use `EnvironmentRef` instead of `ProjectRef`
3. Add `environment-uuid` label to all test applications
4. Remove tests related to `spec.environments[]` management

**Before:**

```go
testProject := &platformv1alpha1.Project{...}
k8sClient.Create(ctx, testProject)

resource := &platformv1alpha1.Application{
    Spec: platformv1alpha1.ApplicationSpec{
        ProjectRef: corev1.LocalObjectReference{Name: "test-project"},
        Type: platformv1alpha1.ApplicationTypeDockerImage,
    },
}
```

**After:**

```go
testProject := &platformv1alpha1.Project{...}
k8sClient.Create(ctx, testProject)

testEnvironment := &platformv1alpha1.Environment{
    ObjectMeta: metav1.ObjectMeta{
        Name: "environment-production-kibaship-com",
        Namespace: "default",
        Labels: map[string]string{
            validation.LabelResourceUUID: "env-uuid-123",
            validation.LabelResourceSlug: "production",
            validation.LabelProjectUUID: "550e8400-e29b-41d4-a716-446655440000",
        },
    },
    Spec: platformv1alpha1.EnvironmentSpec{
        ProjectRef: corev1.LocalObjectReference{Name: "test-project"},
    },
}
k8sClient.Create(ctx, testEnvironment)

resource := &platformv1alpha1.Application{
    ObjectMeta: metav1.ObjectMeta{
        Labels: map[string]string{
            validation.LabelResourceUUID: "app-uuid-123",
            validation.LabelResourceSlug: "myapp",
            validation.LabelEnvironmentUUID: "env-uuid-123",
            validation.LabelProjectUUID: "550e8400-e29b-41d4-a716-446655440000",
        },
    },
    Spec: platformv1alpha1.ApplicationSpec{
        EnvironmentRef: corev1.LocalObjectReference{Name: "environment-production-kibaship-com"},
        Type: platformv1alpha1.ApplicationTypeDockerImage,
    },
}
```

**Test Cases to Update:**

1. ✅ "should successfully reconcile the resource" - Add environment creation
2. ✅ "should validate GitRepository application type" - Add environment creation
3. ✅ "should validate MySQL application type" - Add environment creation
4. ❌ REMOVE: Tests related to `spec.environments[]` array management

**Files to modify:**

- `internal/controller/application_controller_test.go`

#### 8.5 Update Deployment Controller Tests

**File:** `internal/controller/deployment_controller_test.go`

**Changes Required:**

1. Create environment before creating application
2. Add `environment-uuid` label to deployments
3. Remove `spec.environmentName` field from deployment specs

**Before:**

```go
testDeployment = &platformv1alpha1.Deployment{
    Spec: platformv1alpha1.DeploymentSpec{
        ApplicationRef: corev1.LocalObjectReference{Name: testApplication.Name},
        EnvironmentName: "production",  // REMOVE THIS
        GitRepository: &platformv1alpha1.GitRepositoryDeploymentConfig{...},
    },
}
```

**After:**

```go
testDeployment = &platformv1alpha1.Deployment{
    ObjectMeta: metav1.ObjectMeta{
        Labels: map[string]string{
            validation.LabelEnvironmentUUID: "env-uuid-123",  // ADD THIS
            // ... other labels ...
        },
    },
    Spec: platformv1alpha1.DeploymentSpec{
        ApplicationRef: corev1.LocalObjectReference{Name: testApplication.Name},
        GitRepository: &platformv1alpha1.GitRepositoryDeploymentConfig{...},
    },
}
```

**Test Cases to Update:**

1. ✅ "should create a pipeline with correct parameters" - Add environment setup
2. ✅ "should create PipelineRun with correct commit SHA" - Add environment setup
3. ✅ MySQL deployment tests - Add environment setup

**Files to modify:**

- `internal/controller/deployment_controller_test.go`

#### 8.6 Update Integration Tests

**File:** `internal/controller/application_integration_test.go`

**Review and update any integration tests to include environment creation.**

**Files to modify:**

- `internal/controller/application_integration_test.go`

#### 8.7 Update Test Suite Setup

**File:** `internal/controller/suite_test.go`

**Verify that:**

- Environment CRD is registered in scheme
- Test environment cleanup includes environments

**Files to modify:**

- `internal/controller/suite_test.go`

### Phase 9: Update Sample Manifests

#### 9.1 Create Environment Sample

**File:** `config/samples/platform_v1alpha1_environment.yaml` (NEW)

```yaml
apiVersion: platform.operator.kibaship.com/v1alpha1
kind: Environment
metadata:
  name: environment-staging-kibaship-com
  namespace: project-myproject-kibaship-com
  labels:
    platform.kibaship.com/uuid: "550e8400-e29b-41d4-a716-446655440000"
    platform.kibaship.com/slug: "staging"
    platform.kibaship.com/project-uuid: "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
spec:
  projectRef:
    name: project-myproject-kibaship-com
  description: "Staging environment for testing"
```

**Files to create:**

- `config/samples/platform_v1alpha1_environment.yaml`

#### 9.2 Update Application Sample

**File:** `config/samples/platform_v1alpha1_application.yaml`

**Change:**

```yaml
# OLD:
spec:
  projectRef:
    name: project-myproject-kibaship-com

# NEW:
spec:
  environmentRef:
    name: environment-production-kibaship-com
```

**Add label:**

```yaml
labels:
  platform.kibaship.com/environment-uuid: "env-uuid-here"
```

**Files to modify:**

- `config/samples/platform_v1alpha1_application.yaml`

#### 9.3 Update Deployment Sample

**File:** `config/samples/platform_v1alpha1_deployment.yaml`

**Remove:**

```yaml
spec:
  environmentName: "production" # REMOVE
```

**Add label:**

```yaml
labels:
  platform.kibaship.com/environment-uuid: "env-uuid-here"
```

**Files to modify:**

- `config/samples/platform_v1alpha1_deployment.yaml`

#### 9.4 Update Demo Samples

**File:** `config/samples/demo_automatic_domain_creation.yaml`

Update this file to include environment resources.

**Files to modify:**

- `config/samples/demo_automatic_domain_creation.yaml`

### Phase 10: Update Documentation

#### 10.1 Update API Documentation

- Document the new Environment CRD
- Update examples showing the new hierarchy
- Migration guide for existing deployments

#### 10.2 Update README

If there's a README explaining the resource model, update it.

### Phase 11: Testing Strategy

#### 11.1 Unit Tests Execution Order

Run tests incrementally to catch issues early:

```bash
# 1. Test Environment CRD and controller
make test FOCUS="Environment Controller"

# 2. Test Project controller (with environment auto-creation)
make test FOCUS="Project Controller"

# 3. Test Application controller (with environment references)
make test FOCUS="Application Controller"

# 4. Test Deployment controller (with environment labels)
make test FOCUS="Deployment Controller"

# 5. Run full test suite
make test
```

#### 11.2 Integration Testing

```bash
# Generate manifests
make manifests

# Install CRDs
make install

# Run operator locally
make run

# Create test resources in order:
kubectl apply -f config/samples/platform_v1alpha1_project.yaml
# Verify production environment auto-created
kubectl get environments -n project-myproject-kibaship-com

# Create additional environment
kubectl apply -f config/samples/platform_v1alpha1_environment.yaml

# Create application
kubectl apply -f config/samples/platform_v1alpha1_application.yaml

# Create deployment
kubectl apply -f config/samples/platform_v1alpha1_deployment.yaml
```

#### 11.3 Deletion Testing

Test cascading deletes work correctly:

```bash
# Delete environment - should delete all apps
kubectl delete environment environment-staging-kibaship-com -n project-myproject-kibaship-com

# Verify all applications in that environment are deleted
kubectl get applications -n project-myproject-kibaship-com
```

## Migration Strategy for Existing Data

### Option 1: Fresh Start (Recommended for Development)

1. Delete all existing Applications and Deployments
2. Apply new CRDs
3. Create Environments
4. Recreate Applications with environment references
5. Recreate Deployments

### Option 2: Data Migration (For Production)

Create a migration script:

```go
// internal/migration/add_environments.go
func MigrateApplicationsToEnvironments(ctx context.Context, client client.Client) error {
    // 1. For each Project, create a "production" environment if it doesn't exist
    // 2. For each Application, update spec.environmentRef to point to production environment
    // 3. Add environment-uuid label to all Applications and Deployments
}
```

## Files Summary

### Files to CREATE (NEW)

1. `api/v1alpha1/environment_types.go` - Environment CRD definition
2. `internal/controller/environment_controller.go` - Environment controller
3. `internal/controller/environment_controller_test.go` - Environment tests
4. `internal/controller/test_helpers.go` - Shared test helpers
5. `config/samples/platform_v1alpha1_environment.yaml` - Sample environment

### Files to MODIFY (EXISTING)

1. `api/v1alpha1/application_types.go` - Change ProjectRef → EnvironmentRef, remove Environments array
2. `api/v1alpha1/deployment_types.go` - Remove environmentName field
3. `pkg/validation/labels.go` - Add LabelEnvironmentUUID
4. `internal/controller/project_controller.go` - Add auto-create production environment
5. `internal/controller/application_controller.go` - Update to use EnvironmentRef, remove environment management
6. `internal/controller/deployment_controller.go` - Remove environmentName usage
7. `internal/controller/resource_labeler.go` - Add environment UUID validation
8. `cmd/main.go` - Register Environment controller and webhook
9. `config/crd/kustomization.yaml` - Add environment CRD
10. `internal/controller/project_controller_test.go` - Add environment auto-creation tests
11. `internal/controller/application_controller_test.go` - Update all tests to use environments
12. `internal/controller/deployment_controller_test.go` - Update all tests to use environments
13. `internal/controller/application_integration_test.go` - Add environment setup
14. `config/samples/platform_v1alpha1_application.yaml` - Update to use environmentRef
15. `config/samples/platform_v1alpha1_deployment.yaml` - Remove environmentName
16. `config/samples/demo_automatic_domain_creation.yaml` - Add environment

### Files Generated Automatically (by make manifests)

1. `config/crd/bases/platform.operator.kibaship.com_environments.yaml`
2. Updated `config/crd/bases/platform.operator.kibaship.com_applications.yaml`
3. Updated `config/crd/bases/platform.operator.kibaship.com_deployments.yaml`

## Execution Steps

### Step-by-Step Implementation

**Step 1: Scaffold Environment CRD**

- Create `environment_types.go`
- Run `make generate && make manifests`
- Verify CRD generated

**Step 2: Create Environment Controller**

- Create `environment_controller.go`
- Implement reconciliation logic
- Add to `cmd/main.go`

**Step 3: Update Project Controller**

- Add environment auto-creation
- Test that production env is created

**Step 4: Update Application CRD and Controller**

- Modify spec (ProjectRef → EnvironmentRef)
- Update controller logic
- Run `make manifests`

**Step 5: Update Deployment CRD and Controller**

- Remove environmentName
- Update controller logic
- Run `make manifests`

**Step 6: Update Labels and Validation**

- Add environment-uuid label constant
- Update validation functions

**Step 7: Write Tests**

- Create environment tests
- Update project tests
- Update application tests
- Update deployment tests

**Step 8: Run Tests Incrementally**

- Test each controller individually
- Fix any failures
- Run full suite

**Step 9: Integration Testing**

- Deploy locally
- Create resources manually
- Verify behavior

**Step 10: Update Samples and Docs**

- Create sample manifests
- Update documentation

## Critical Success Factors

1. **Cascading Deletes Must Work**

   - Deleting Environment must delete all Applications
   - Applications must have owner references to Environments

2. **Label Propagation Must Be Correct**

   - Environment UUID must propagate to Applications and Deployments
   - Project UUID must still be accessible

3. **Backward Compatibility**

   - Old resources must be migrated or clearly documented as incompatible

4. **Test Coverage**
   - Every controller must have tests with environment setup
   - Every test must create Project → Environment → Application → Deployment in order

## Risk Mitigation

### High-Risk Areas

1. **Cascading Deletes** - If owner references are wrong, orphaned resources
2. **Label Propagation** - If labels missing, resources can't be queried correctly
3. **Test Failures** - If tests don't include environment, they will all fail

### Mitigation Strategies

1. Write comprehensive tests for each phase
2. Test locally before committing each phase
3. Use test helpers to ensure consistency
4. Verify CRD generation after each change
5. Test deletion scenarios explicitly

## Timeline Estimate

- **Phase 1-2** (Environment CRD & Controller): 2-3 hours
- **Phase 3-4** (Update Application & Deployment): 2-3 hours
- **Phase 5-7** (Project Controller, Validation, CRDs): 1-2 hours
- **Phase 8** (Update ALL Tests): **4-6 hours** ⚠️ MOST TIME-CONSUMING
- **Phase 9-10** (Samples & Docs): 1-2 hours
- **Phase 11** (Integration Testing & Debugging): 2-4 hours

**Total Estimate: 12-20 hours**

## Conclusion

This is a comprehensive architectural change that touches almost every part of the system. The key to success is:

1. **Incremental Implementation** - Do one phase at a time
2. **Test After Each Change** - Don't accumulate broken tests
3. **Careful Label Management** - Ensure all UUIDs propagate correctly
4. **Owner References** - Critical for cascading deletes
5. **Patience** - This is a big refactor, take time to do it right

Once complete, the new hierarchy will provide:

- Better isolation between environments
- Clearer resource organization
- Easier environment management
- Proper cascading deletes
- More intuitive API structure

---

**Ready for implementation? Start with Phase 1 and work through sequentially. Test after each phase.**

# ApplicationDomain Implementation Plan

## Executive Summary

This document outlines a comprehensive plan to implement ApplicationDomain functionality in the KibaShip operator, enabling automatic subdomain generation for GitRepository applications and future support for custom domains with SSL certificates.

## 1. Requirements Analysis

### Functional Requirements

1. **Operator Subdomain Configuration**
   - Users must specify an operator subdomain during installation (e.g., `myapps.kibaship.com`, `planethero.com`)
   - This domain serves as the base for all application subdomains
   - Must be configurable and persistent across operator restarts

2. **Automatic Subdomain Generation**
   - Generate random subdomains for GitRepository applications
   - Format: `<app-random-slug>.<operator-domain>`
   - Automatic ApplicationDomain resource creation
   - Default domain marking (`default: true`)

3. **ApplicationDomain Resource**
   - New CRD for domain management
   - Port configuration for ingress routing
   - Reference to parent Application
   - Support for default and custom domain types

4. **Future Custom Domain Support**
   - User-defined custom domains
   - Certificate management integration with cert-manager
   - Domain validation and DNS management

### Non-Functional Requirements

1. **Security**: Domain validation, secure certificate generation
2. **Scalability**: Handle multiple domains per application
3. **Reliability**: Proper cleanup on resource deletion
4. **Observability**: Status reporting and error handling

## 2. Architecture Design

### 2.1 Operator Subdomain Configuration Strategy

#### Option A: Environment Variable (Recommended)
**Implementation**: Add `KIBASHIP_OPERATOR_DOMAIN` environment variable to manager deployment

**Pros**:
- Simple implementation
- Standard Kubernetes pattern
- Easy configuration during deployment
- Immutable during runtime (security benefit)

**Configuration**:
```yaml
# config/manager/manager.yaml
env:
- name: KIBASHIP_OPERATOR_DOMAIN
  value: "myapps.kibaship.com"
```

#### Option B: ConfigMap
**Implementation**: Create dedicated ConfigMap for operator configuration

**Pros**:
- Centralized configuration
- Can be updated without redeployment
- Supports additional configuration options

**Configuration**:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: kibaship-operator-config
  namespace: kibaship-operator
data:
  operator-domain: "myapps.kibaship.com"
```

#### Option C: Custom Resource (Future-Proof)
**Implementation**: Create `OperatorConfig` CRD for comprehensive configuration

**Pros**:
- Type-safe configuration
- Validation support
- Extensible for future features

**Recommended Approach**: Start with Environment Variable (Option A) for simplicity, migrate to ConfigMap (Option B) when more configuration options are needed.

### 2.2 ApplicationDomain CRD Specification

```go
// ApplicationDomain represents a domain configuration for an application
type ApplicationDomain struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   ApplicationDomainSpec   `json:"spec,omitempty"`
    Status ApplicationDomainStatus `json:"status,omitempty"`
}

// ApplicationDomainSpec defines the desired state of ApplicationDomain
type ApplicationDomainSpec struct {
    // ApplicationRef references the parent application
    // +kubebuilder:validation:Required
    ApplicationRef corev1.LocalObjectReference `json:"applicationRef"`

    // Domain is the full domain name (e.g., "my-app-abc123.myapps.kibaship.com" or "custom.example.com")
    // +kubebuilder:validation:Required
    // +kubebuilder:validation:Pattern=`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)*$`
    Domain string `json:"domain"`

    // Port is the application port for ingress routing
    // +kubebuilder:validation:Required
    // +kubebuilder:validation:Minimum=1
    // +kubebuilder:validation:Maximum=65535
    // +kubebuilder:default=3000
    Port int32 `json:"port"`

    // Type indicates if this is a default generated domain or custom domain
    // +kubebuilder:validation:Enum=default;custom
    // +kubebuilder:default=default
    Type ApplicationDomainType `json:"type,omitempty"`

    // Default indicates if this is the default domain for the application
    // Only one domain per application can be marked as default
    // +kubebuilder:default=false
    Default bool `json:"default,omitempty"`

    // TLSEnabled indicates if TLS/SSL should be enabled for this domain
    // +kubebuilder:default=true
    TLSEnabled bool `json:"tlsEnabled,omitempty"`

    // CertificateIssuerRef references the cert-manager issuer for custom domains
    // Only used when Type is "custom"
    // +optional
    CertificateIssuerRef *cmmeta.ObjectReference `json:"certificateIssuerRef,omitempty"`
}

// ApplicationDomainType defines the type of domain
type ApplicationDomainType string

const (
    // ApplicationDomainTypeDefault represents an auto-generated default domain
    ApplicationDomainTypeDefault ApplicationDomainType = "default"
    // ApplicationDomainTypeCustom represents a user-defined custom domain
    ApplicationDomainTypeCustom ApplicationDomainType = "custom"
)

// ApplicationDomainStatus defines the observed state of ApplicationDomain
type ApplicationDomainStatus struct {
    // Phase indicates the current phase of the domain
    // +kubebuilder:validation:Enum=Pending;Ready;Failed
    Phase ApplicationDomainPhase `json:"phase,omitempty"`

    // CertificateReady indicates if the TLS certificate is ready
    CertificateReady bool `json:"certificateReady,omitempty"`

    // IngressReady indicates if the ingress is configured and ready
    IngressReady bool `json:"ingressReady,omitempty"`

    // DNSConfigured indicates if DNS is properly configured (for custom domains)
    DNSConfigured bool `json:"dnsConfigured,omitempty"`

    // LastReconcileTime is the last time the domain was reconciled
    LastReconcileTime *metav1.Time `json:"lastReconcileTime,omitempty"`

    // Message provides human-readable status information
    Message string `json:"message,omitempty"`

    // Conditions represent the latest available observations of the domain state
    Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// ApplicationDomainPhase defines the phase of an ApplicationDomain
type ApplicationDomainPhase string

const (
    // ApplicationDomainPhasePending indicates the domain is being configured
    ApplicationDomainPhasePending ApplicationDomainPhase = "Pending"
    // ApplicationDomainPhaseReady indicates the domain is ready for use
    ApplicationDomainPhaseReady ApplicationDomainPhase = "Ready"
    // ApplicationDomainPhaseFailed indicates the domain configuration failed
    ApplicationDomainPhaseFailed ApplicationDomainPhase = "Failed"
)
```

### 2.3 Subdomain Generation Strategy

#### Random Slug Generation Algorithm
```go
// GenerateSubdomain creates a unique subdomain for an application
func GenerateSubdomain(appName string) string {
    // Extract meaningful parts from application name
    // project-myproject-app-frontend-kibaship-com -> frontend
    parts := strings.Split(appName, "-")
    var appSlug string

    // Find app slug (after "app-" prefix)
    for i, part := range parts {
        if part == "app" && i+1 < len(parts) {
            appSlug = parts[i+1]
            break
        }
    }

    if appSlug == "" {
        appSlug = "app"
    }

    // Generate random suffix (8 characters, lowercase alphanumeric)
    randomSuffix := generateRandomString(8)

    return fmt.Sprintf("%s-%s", appSlug, randomSuffix)
}

func generateRandomString(length int) string {
    const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
    // Use crypto/rand for secure random generation
}
```

#### Domain Uniqueness Validation
```go
func (r *ApplicationDomainReconciler) validateDomainUniqueness(ctx context.Context, domain string, excludeUID types.UID) error {
    var domains platformv1alpha1.ApplicationDomainList
    if err := r.List(ctx, &domains); err != nil {
        return err
    }

    for _, d := range domains.Items {
        if d.UID != excludeUID && d.Spec.Domain == domain {
            return fmt.Errorf("domain %s already exists", domain)
        }
    }
    return nil
}
```

## 3. Implementation Plan

### Phase 1: Core Infrastructure (Week 1-2)

#### 3.1 Operator Configuration Setup
**Files to Create/Modify**:
- `cmd/main.go` - Add environment variable reading
- `config/manager/manager.yaml` - Add environment variable
- `internal/controller/config.go` - Configuration management utilities

**Implementation Steps**:
1. Add `KIBASHIP_OPERATOR_DOMAIN` environment variable to manager deployment
2. Create configuration utility functions for domain access
3. Add validation for operator domain format
4. Update deployment documentation

#### 3.2 ApplicationDomain CRD Creation
**Files to Create**:
- `api/v1alpha1/applicationdomain_types.go` - CRD definition
- `api/v1alpha1/applicationdomain_webhook.go` - Validation webhook
- `config/crd/bases/platform.operator.kibaship.com_applicationdomains.yaml` - Generated CRD
- `config/samples/platform_v1alpha1_applicationdomain.yaml` - Sample resource

**Implementation Steps**:
1. Define ApplicationDomain types with proper kubebuilder annotations
2. Implement validation webhook for domain format and uniqueness
3. Generate CRD manifests using kubebuilder
4. Create comprehensive test samples

#### 3.3 ApplicationDomain Controller
**Files to Create**:
- `internal/controller/applicationdomain_controller.go` - Main controller
- `internal/controller/applicationdomain_controller_test.go` - Unit tests

**Controller Responsibilities**:
1. **Domain Validation**: Ensure domain uniqueness and format
2. **Status Management**: Update phase and conditions
3. **Cleanup**: Remove associated resources on deletion
4. **Ingress Creation**: Generate ingress resources for domains (future phase)

### Phase 2: Application Integration (Week 2-3)

#### 3.4 Application Controller Enhancement
**Files to Modify**:
- `internal/controller/application_controller.go`
- `internal/controller/application_controller_test.go`

**New Functionality**:
1. **Domain Generation**: Create ApplicationDomain for GitRepository applications
2. **Ownership Management**: Set proper owner references
3. **Status Updates**: Reflect domain status in Application status
4. **Cleanup**: Ensure domains are cleaned up on Application deletion

**Integration Logic**:
```go
func (r *ApplicationReconciler) handleGitRepositoryApplication(ctx context.Context, app *platformv1alpha1.Application) error {
    // Check if default domain already exists
    existingDomain, err := r.findDefaultDomain(ctx, app)
    if err != nil {
        return err
    }

    if existingDomain == nil {
        // Generate new default domain
        domain := r.generateDefaultDomain(app)
        if err := r.createApplicationDomain(ctx, app, domain); err != nil {
            return err
        }
    }

    return nil
}
```

#### 3.5 Domain Generation Utilities
**Files to Create**:
- `internal/controller/domain_utils.go` - Domain generation and validation
- `internal/controller/domain_utils_test.go` - Comprehensive tests

**Utilities Include**:
1. Random subdomain generation
2. Domain format validation
3. Uniqueness checking
4. Operator domain resolution

### Phase 3: Advanced Features (Week 3-4)

#### 3.6 Ingress Management
**Files to Create**:
- `internal/controller/ingress_manager.go` - Ingress resource management
- `internal/controller/ingress_manager_test.go` - Tests

**Ingress Features**:
1. **Automatic Creation**: Generate ingress for ApplicationDomains
2. **TLS Configuration**: Automatic certificate references
3. **Port Routing**: Route to correct application port
4. **Status Monitoring**: Update domain status based on ingress readiness

#### 3.7 Certificate Management Integration
**Files to Create/Modify**:
- `internal/controller/certificate_manager.go` - cert-manager integration
- Update ApplicationDomain controller for certificate lifecycle

**Certificate Features**:
1. **Default Domains**: Use operator-managed wildcard certificate
2. **Custom Domains**: Generate individual certificates via cert-manager
3. **Status Tracking**: Monitor certificate readiness
4. **Renewal Handling**: Automatic certificate renewal

### Phase 4: Testing and Documentation (Week 4-5)

#### 3.8 Comprehensive Testing
**Files to Create**:
- `internal/controller/applicationdomain_integration_test.go`
- `test/e2e/applicationdomain_test.go`
- Add ApplicationDomain tests to existing test suites

**Test Coverage**:
1. **Unit Tests**: Domain generation, validation, controller logic
2. **Integration Tests**: Application-Domain interaction
3. **E2E Tests**: Complete workflow from Application creation to domain access
4. **Webhook Tests**: Validation webhook behavior

#### 3.9 Documentation and Samples
**Files to Create/Update**:
- `docs/applicationdomain.md` - Feature documentation
- `config/samples/` - Extended samples showing domain usage
- `README.md` - Update with domain configuration instructions
- Update `operator.md` with ApplicationDomain information

## 4. Detailed Implementation Specifications

### 4.1 Environment Variable Configuration

#### Manager Deployment Updates
```yaml
# config/manager/manager.yaml
containers:
- command:
  - /manager
  args:
    - --leader-elect
    - --health-probe-bind-address=:8081
  env:
  - name: KIBASHIP_OPERATOR_DOMAIN
    value: ""  # To be set during deployment
  - name: KIBASHIP_DEFAULT_PORT
    value: "3000"
```

#### Configuration Utility Implementation
```go
// internal/controller/config.go
package controller

import (
    "fmt"
    "os"
    "regexp"
    "strconv"
)

type OperatorConfig struct {
    Domain      string
    DefaultPort int32
}

var operatorConfig *OperatorConfig

func GetOperatorConfig() (*OperatorConfig, error) {
    if operatorConfig != nil {
        return operatorConfig, nil
    }

    domain := os.Getenv("KIBASHIP_OPERATOR_DOMAIN")
    if domain == "" {
        return nil, fmt.Errorf("KIBASHIP_OPERATOR_DOMAIN environment variable is required")
    }

    // Validate domain format
    domainRegex := regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)*$`)
    if !domainRegex.MatchString(domain) {
        return nil, fmt.Errorf("invalid domain format: %s", domain)
    }

    defaultPortStr := os.Getenv("KIBASHIP_DEFAULT_PORT")
    if defaultPortStr == "" {
        defaultPortStr = "3000"
    }

    defaultPort, err := strconv.ParseInt(defaultPortStr, 10, 32)
    if err != nil {
        return nil, fmt.Errorf("invalid default port: %s", defaultPortStr)
    }

    operatorConfig = &OperatorConfig{
        Domain:      domain,
        DefaultPort: int32(defaultPort),
    }

    return operatorConfig, nil
}
```

### 4.2 ApplicationDomain Controller Implementation

#### Core Controller Structure
```go
// internal/controller/applicationdomain_controller.go
package controller

import (
    "context"
    "fmt"
    "time"

    "k8s.io/apimachinery/pkg/runtime"
    "k8s.io/apimachinery/pkg/types"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"
    "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
    "sigs.k8s.io/controller-runtime/pkg/log"

    platformv1alpha1 "github.com/kibamail/kibaship-operator/api/v1alpha1"
)

// ApplicationDomainReconciler reconciles ApplicationDomain objects
type ApplicationDomainReconciler struct {
    client.Client
    Scheme *runtime.Scheme
}

func (r *ApplicationDomainReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log := log.FromContext(ctx)

    // Fetch the ApplicationDomain instance
    var appDomain platformv1alpha1.ApplicationDomain
    if err := r.Get(ctx, req.NamespacedName, &appDomain); err != nil {
        if client.IgnoreNotFound(err) != nil {
            log.Error(err, "unable to fetch ApplicationDomain")
        }
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // Handle deletion
    if !appDomain.DeletionTimestamp.IsZero() {
        return r.handleDeletion(ctx, &appDomain)
    }

    // Add finalizer if not present
    if !controllerutil.ContainsFinalizer(&appDomain, ApplicationDomainFinalizerName) {
        controllerutil.AddFinalizer(&appDomain, ApplicationDomainFinalizerName)
        if err := r.Update(ctx, &appDomain); err != nil {
            return ctrl.Result{}, err
        }
        return ctrl.Result{Requeue: true}, nil
    }

    // Validate domain configuration
    if err := r.validateDomain(ctx, &appDomain); err != nil {
        return r.updateStatus(ctx, &appDomain, platformv1alpha1.ApplicationDomainPhaseFailed, err.Error())
    }

    // Create/update ingress resources
    if err := r.reconcileIngress(ctx, &appDomain); err != nil {
        return r.updateStatus(ctx, &appDomain, platformv1alpha1.ApplicationDomainPhaseFailed,
            fmt.Sprintf("failed to reconcile ingress: %v", err))
    }

    // Handle certificate management for custom domains
    if appDomain.Spec.Type == platformv1alpha1.ApplicationDomainTypeCustom {
        if err := r.reconcileCertificate(ctx, &appDomain); err != nil {
            return r.updateStatus(ctx, &appDomain, platformv1alpha1.ApplicationDomainPhaseFailed,
                fmt.Sprintf("failed to reconcile certificate: %v", err))
        }
    }

    // Update status to Ready
    return r.updateStatus(ctx, &appDomain, platformv1alpha1.ApplicationDomainPhaseReady, "Domain is ready")
}

const ApplicationDomainFinalizerName = "platform.operator.kibaship.com/applicationdomain-finalizer"

func (r *ApplicationDomainReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&platformv1alpha1.ApplicationDomain{}).
        Complete(r)
}
```

#### Status Management Implementation
```go
func (r *ApplicationDomainReconciler) updateStatus(ctx context.Context, appDomain *platformv1alpha1.ApplicationDomain,
    phase platformv1alpha1.ApplicationDomainPhase, message string) (ctrl.Result, error) {

    now := metav1.Now()
    appDomain.Status.Phase = phase
    appDomain.Status.Message = message
    appDomain.Status.LastReconcileTime = &now

    // Update conditions
    condition := metav1.Condition{
        Type:    "Ready",
        Status:  metav1.ConditionFalse,
        Reason:  "Reconciling",
        Message: message,
    }

    if phase == platformv1alpha1.ApplicationDomainPhaseReady {
        condition.Status = metav1.ConditionTrue
        condition.Reason = "Ready"
    } else if phase == platformv1alpha1.ApplicationDomainPhaseFailed {
        condition.Status = metav1.ConditionFalse
        condition.Reason = "Failed"
    }

    meta.SetStatusCondition(&appDomain.Status.Conditions, condition)

    if err := r.Status().Update(ctx, appDomain); err != nil {
        return ctrl.Result{}, err
    }

    return ctrl.Result{}, nil
}
```

### 4.3 Application Controller Integration

#### Domain Creation Logic
```go
// Add to application_controller.go
func (r *ApplicationReconciler) handleGitRepositoryDomains(ctx context.Context, app *platformv1alpha1.Application) error {
    // Only process GitRepository applications
    if app.Spec.Type != platformv1alpha1.ApplicationTypeGitRepository {
        return nil
    }

    // Check if default domain already exists
    var domains platformv1alpha1.ApplicationDomainList
    if err := r.List(ctx, &domains,
        client.InNamespace(app.Namespace),
        client.MatchingFields{"spec.applicationRef.name": app.Name},
    ); err != nil {
        return fmt.Errorf("failed to list domains: %v", err)
    }

    // Find existing default domain
    var defaultDomain *platformv1alpha1.ApplicationDomain
    for _, domain := range domains.Items {
        if domain.Spec.Default {
            defaultDomain = &domain
            break
        }
    }

    // Create default domain if it doesn't exist
    if defaultDomain == nil {
        return r.createDefaultDomain(ctx, app)
    }

    return nil
}

func (r *ApplicationReconciler) createDefaultDomain(ctx context.Context, app *platformv1alpha1.Application) error {
    config, err := GetOperatorConfig()
    if err != nil {
        return fmt.Errorf("failed to get operator config: %v", err)
    }

    // Generate unique subdomain
    subdomain := GenerateSubdomain(app.Name)
    fullDomain := fmt.Sprintf("%s.%s", subdomain, config.Domain)

    // Create ApplicationDomain resource
    domain := &platformv1alpha1.ApplicationDomain{
        ObjectMeta: metav1.ObjectMeta{
            Name:      fmt.Sprintf("%s-default", app.Name),
            Namespace: app.Namespace,
            Labels: map[string]string{
                "platform.operator.kibaship.com/application": app.Name,
                "platform.operator.kibaship.com/domain-type": "default",
            },
        },
        Spec: platformv1alpha1.ApplicationDomainSpec{
            ApplicationRef: corev1.LocalObjectReference{Name: app.Name},
            Domain:         fullDomain,
            Port:           config.DefaultPort,
            Type:           platformv1alpha1.ApplicationDomainTypeDefault,
            Default:        true,
            TLSEnabled:     true,
        },
    }

    // Set owner reference
    if err := controllerutil.SetControllerReference(app, domain, r.Scheme); err != nil {
        return fmt.Errorf("failed to set owner reference: %v", err)
    }

    if err := r.Create(ctx, domain); err != nil {
        return fmt.Errorf("failed to create domain: %v", err)
    }

    return nil
}
```

### 4.4 Validation Webhook Implementation

#### Webhook Structure
```go
// api/v1alpha1/applicationdomain_webhook.go
package v1alpha1

import (
    "context"
    "fmt"
    "regexp"

    "k8s.io/apimachinery/pkg/runtime"
    "k8s.io/apimachinery/pkg/util/validation/field"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/webhook"
)

func (r *ApplicationDomain) SetupWebhookWithManager(mgr ctrl.Manager) error {
    return ctrl.NewWebhookManagedBy(mgr).
        For(r).
        Complete()
}

var _ webhook.Validator = &ApplicationDomain{}

func (r *ApplicationDomain) ValidateCreate() error {
    return r.validateDomain()
}

func (r *ApplicationDomain) ValidateUpdate(old runtime.Object) error {
    return r.validateDomain()
}

func (r *ApplicationDomain) ValidateDelete() error {
    return nil
}

func (r *ApplicationDomain) validateDomain() error {
    var allErrs field.ErrorList

    // Validate domain format
    domainRegex := regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)*$`)
    if !domainRegex.MatchString(r.Spec.Domain) {
        allErrs = append(allErrs, field.Invalid(
            field.NewPath("spec", "domain"),
            r.Spec.Domain,
            "domain must be a valid DNS name"))
    }

    // Validate port range
    if r.Spec.Port < 1 || r.Spec.Port > 65535 {
        allErrs = append(allErrs, field.Invalid(
            field.NewPath("spec", "port"),
            r.Spec.Port,
            "port must be between 1 and 65535"))
    }

    // Validate certificate issuer for custom domains
    if r.Spec.Type == ApplicationDomainTypeCustom && r.Spec.TLSEnabled && r.Spec.CertificateIssuerRef == nil {
        allErrs = append(allErrs, field.Required(
            field.NewPath("spec", "certificateIssuerRef"),
            "certificate issuer is required for custom domains with TLS"))
    }

    if len(allErrs) == 0 {
        return nil
    }

    return apierrors.NewInvalid(
        schema.GroupKind{Group: "platform.operator.kibaship.com", Kind: "ApplicationDomain"},
        r.Name, allErrs)
}
```

## 5. Testing Strategy

### 5.1 Unit Tests

#### Domain Generation Tests
```go
// internal/controller/domain_utils_test.go
func TestGenerateSubdomain(t *testing.T) {
    tests := []struct {
        name     string
        appName  string
        expected string // regex pattern
    }{
        {
            name:     "standard application name",
            appName:  "project-myproject-app-frontend-kibaship-com",
            expected: `^frontend-[a-z0-9]{8}$`,
        },
        {
            name:     "complex application name",
            appName:  "project-ecommerce-app-api-gateway-kibaship-com",
            expected: `^api-[a-z0-9]{8}$`,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := GenerateSubdomain(tt.appName)
            matched, err := regexp.MatchString(tt.expected, result)
            assert.NoError(t, err)
            assert.True(t, matched, "Generated subdomain %s doesn't match pattern %s", result, tt.expected)
        })
    }
}
```

#### Controller Tests
```go
// internal/controller/applicationdomain_controller_test.go
func TestApplicationDomainReconciler_Reconcile(t *testing.T) {
    // Setup test environment with envtest
    testCases := []struct {
        name           string
        domain         *platformv1alpha1.ApplicationDomain
        expectedPhase  platformv1alpha1.ApplicationDomainPhase
        expectedError  bool
    }{
        {
            name: "successful default domain creation",
            domain: &platformv1alpha1.ApplicationDomain{
                ObjectMeta: metav1.ObjectMeta{
                    Name:      "test-app-default",
                    Namespace: "test-namespace",
                },
                Spec: platformv1alpha1.ApplicationDomainSpec{
                    ApplicationRef: corev1.LocalObjectReference{Name: "test-app"},
                    Domain:         "test-app-abc123.example.com",
                    Port:           3000,
                    Type:           platformv1alpha1.ApplicationDomainTypeDefault,
                    Default:        true,
                    TLSEnabled:     true,
                },
            },
            expectedPhase: platformv1alpha1.ApplicationDomainPhaseReady,
            expectedError: false,
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### 5.2 Integration Tests

#### Application-Domain Integration
```go
// internal/controller/application_domain_integration_test.go
func TestApplicationDomainIntegration(t *testing.T) {
    // Test that creating a GitRepository application automatically creates a default domain
    // Test that deleting an application cleans up associated domains
    // Test domain uniqueness across applications
}
```

### 5.3 End-to-End Tests

#### Complete Workflow Tests
```go
// test/e2e/applicationdomain_test.go
func TestApplicationDomainE2E(t *testing.T) {
    // Test complete workflow from Project -> Application -> ApplicationDomain -> Ingress
}
```

## 6. Future Enhancements

### 6.1 Custom Domain Support (Phase 5)

#### User-Defined Domains
- Allow users to specify custom domains for applications
- DNS validation before certificate issuance
- Support for CNAME verification
- Integration with external DNS providers

#### Implementation Approach
```go
// Custom domain creation flow
func (r *ApplicationDomainReconciler) handleCustomDomain(ctx context.Context, appDomain *platformv1alpha1.ApplicationDomain) error {
    // 1. Validate DNS configuration
    // 2. Create cert-manager Certificate resource
    // 3. Wait for certificate issuance
    // 4. Create ingress with custom domain
    // 5. Update status
}
```

### 6.2 Multi-Domain Support

#### Multiple Domains per Application
- Support for multiple custom domains pointing to same application
- Primary domain designation
- Domain-specific routing rules

### 6.3 Advanced Certificate Management

#### Wildcard Certificates
- Support for wildcard certificates for operator domains
- Automatic certificate renewal
- Certificate status monitoring and alerting

#### External Certificate Providers
- Integration with external certificate authorities
- Support for EV certificates
- Custom certificate validation hooks

### 6.4 DNS Management Integration

#### Automatic DNS Configuration
- Integration with cloud DNS providers (Route53, CloudDNS, etc.)
- Automatic A/CNAME record creation
- DNS validation for custom domains

### 6.5 Monitoring and Observability

#### Metrics and Monitoring
- Domain health metrics
- Certificate expiration alerts
- Traffic routing statistics
- SSL/TLS configuration monitoring

## 7. Migration and Upgrade Strategy

### 7.1 Existing Installation Upgrades

#### Configuration Migration
1. **Environment Variable Addition**: Add `KIBASHIP_OPERATOR_DOMAIN` to existing deployments
2. **Backward Compatibility**: Ensure existing applications continue to work
3. **Gradual Migration**: Optional domain creation for existing GitRepository applications

#### Database Schema Updates
1. **CRD Installation**: New ApplicationDomain CRD installation
2. **RBAC Updates**: Additional permissions for domain management
3. **Webhook Registration**: New validation webhooks

### 7.2 Rollback Strategy

#### Safe Rollback Procedures
1. **Resource Preservation**: Keep existing domains during operator downgrade
2. **Configuration Backup**: Backup operator configuration before upgrades
3. **Validation Testing**: Comprehensive testing before production rollouts

## 8. Security Considerations

### 8.1 Domain Security

#### Domain Validation
- Strict domain format validation
- Prevention of domain hijacking
- Subdomain namespace isolation

#### Certificate Security
- Secure certificate storage
- Proper certificate rotation
- TLS configuration best practices

### 8.2 Access Control

#### RBAC Configuration
```yaml
# Additional RBAC rules for ApplicationDomain management
rules:
- apiGroups:
  - platform.operator.kibaship.com
  resources:
  - applicationdomains
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - platform.operator.kibaship.com
  resources:
  - applicationdomains/status
  verbs:
  - get
  - patch
  - update
```

#### Network Security
- Ingress security configurations
- TLS enforcement policies
- Network policy integration

## 9. Deployment and Configuration

### 9.1 Installation Configuration

#### Helm Chart Updates
```yaml
# values.yaml additions
operator:
  domain: ""  # Must be set by user
  defaultPort: 3000

certificates:
  issuer: "letsencrypt-prod"
  wildcardSupport: true
```

#### Environment Configuration
```bash
# Installation example
helm install kibaship-operator ./chart \
  --set operator.domain=myapps.example.com \
  --set certificates.issuer=letsencrypt-prod
```

### 9.2 Validation and Health Checks

#### Startup Validation
- Operator domain validation on startup
- Certificate issuer validation
- DNS connectivity checks

#### Runtime Health Monitoring
- Domain resolution monitoring
- Certificate expiration tracking
- Ingress health verification

## 10. Documentation Requirements

### 10.1 User Documentation

#### Configuration Guide
- Operator domain setup instructions
- Certificate issuer configuration
- DNS requirements and setup

#### User Guide
- Creating applications with domains
- Custom domain configuration
- Troubleshooting common issues

### 10.2 Developer Documentation

#### API Reference
- ApplicationDomain CRD specification
- Controller implementation details
- Webhook validation rules

#### Integration Guide
- Third-party integrations
- Custom certificate providers
- Monitoring and alerting setup

## Conclusion

This comprehensive implementation plan provides a structured approach to adding ApplicationDomain functionality to the KibaShip operator. The phased approach ensures:

1. **Incremental Delivery**: Core functionality first, advanced features later
2. **Risk Mitigation**: Thorough testing and validation at each phase
3. **User Experience**: Simple configuration with powerful extensibility
4. **Production Readiness**: Security, monitoring, and operational considerations

The implementation leverages Kubernetes best practices, follows the operator pattern consistently, and provides a foundation for future enhancements while maintaining backward compatibility and operational excellence.

### Next Steps

1. **Review and Approval**: Stakeholder review of implementation plan
2. **Environment Setup**: Development environment configuration
3. **Phase 1 Implementation**: Begin with operator configuration and CRD creation
4. **Iterative Development**: Implement, test, and refine each phase
5. **Documentation**: Continuous documentation updates throughout development

This plan serves as a comprehensive roadmap for implementing production-ready domain management functionality in the KibaShip operator ecosystem.
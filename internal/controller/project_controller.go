/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	platformv1alpha1 "github.com/kibamail/kibaship-operator/api/v1alpha1"
	"github.com/kibamail/kibaship-operator/pkg/validation"
	"github.com/kibamail/kibaship-operator/pkg/webhooks"
)

const (
	// ProjectFinalizerName is the finalizer name for project cleanup
	ProjectFinalizerName = "platform.kibaship.com/project-finalizer"
)

// ProjectReconciler reconciles a Project object
type ProjectReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	NamespaceManager *NamespaceManager
	Validator        *ProjectValidator
	Notifier         webhooks.Notifier
}

// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=projects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=projects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=platform.operator.kibaship.com,resources=projects/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="*",resources="*",verbs="*"

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Project object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *ProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the Project instance
	var project platformv1alpha1.Project
	if err := r.Get(ctx, req.NamespacedName, &project); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			log.Info("Project resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Project")
		return ctrl.Result{}, err
	}

	// Handle deletion
	if project.DeletionTimestamp != nil {
		return r.handleProjectDeletion(ctx, &project)
	}

	// Add finalizer as the very first step (critical for cleanup)
	if !controllerutil.ContainsFinalizer(&project, ProjectFinalizerName) {
		controllerutil.AddFinalizer(&project, ProjectFinalizerName)
		if err := r.Update(ctx, &project); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Validate project labels (always check these)
	if err := r.Validator.ValidateRequiredLabels(&project); err != nil {
		log.Error(err, "Project label validation failed")
		r.updateStatusWithError(ctx, &project, "Failed", err.Error())
		return ctrl.Result{}, err
	}

	// Track previous phase for webhook emission
	prevPhase := project.Status.Phase

	// Check if this is a new project by looking at status
	isNewProject := project.Status.Phase == "" || project.Status.Phase == "Pending"

	if isNewProject {
		// Set status to Pending for new projects
		if project.Status.Phase == "" {
			project.Status.Phase = "Pending"
			project.Status.Message = "Initializing project"
			if err := r.Status().Update(ctx, &project); err != nil {
				log.Error(err, "Failed to update project status to Pending")
				return ctrl.Result{}, err
			}
			// emit webhook for phase change
			r.emitProjectPhaseChange(ctx, &project, prevPhase, project.Status.Phase)
			prevPhase = project.Status.Phase
		}

		// Validate uniqueness for new projects (exclude this project)
		if err := r.Validator.CheckProjectNameUniqueness(ctx, project.Name, &project); err != nil {
			log.Error(err, "Project name uniqueness validation failed")
			r.updateStatusWithError(ctx, &project, "Failed", err.Error())
			return ctrl.Result{}, err
		}
	}

	// Create or update project namespace
	namespace, err := r.NamespaceManager.CreateProjectNamespace(ctx, &project)
	if err != nil {
		log.Error(err, "Failed to create project namespace")
		r.updateStatusWithError(ctx, &project, "Failed", err.Error())
		return ctrl.Result{}, err
	}

	// Ensure registry credentials are created for this namespace
	if err := r.ensureRegistryCredentials(ctx, namespace.Name); err != nil {
		log.Error(err, "Failed to ensure registry credentials")
		r.updateStatusWithError(ctx, &project, "Failed", fmt.Sprintf("Failed to create registry credentials: %v", err))
		return ctrl.Result{}, err
	}

	// Ensure registry CA certificate is copied to this namespace
	if err := r.ensureRegistryCACertificate(ctx, namespace.Name); err != nil {
		log.Error(err, "Failed to ensure registry CA certificate")
		r.updateStatusWithError(ctx, &project, "Failed", fmt.Sprintf("Failed to copy registry CA certificate: %v", err))
		return ctrl.Result{}, err
	}

	// Ensure Docker config secret is created for registry authentication
	if err := r.ensureRegistryDockerConfig(ctx, namespace.Name); err != nil {
		log.Error(err, "Failed to ensure registry Docker config")
		r.updateStatusWithError(ctx, &project, "Failed", fmt.Sprintf("Failed to create registry Docker config: %v", err))
		return ctrl.Result{}, err
	}

	// Update status to indicate project is ready
	const readyPhase = "Ready"
	if project.Status.Phase != readyPhase {
		project.Status.Phase = readyPhase
		project.Status.NamespaceName = namespace.Name
		project.Status.Message = "Project is ready"
		now := metav1.Now()
		project.Status.LastReconcileTime = &now
		if err := r.Status().Update(ctx, &project); err != nil {
			log.Error(err, "Failed to update project status to Ready")
			return ctrl.Result{}, err
		}
		// emit webhook for phase change
		r.emitProjectPhaseChange(ctx, &project, prevPhase, project.Status.Phase)
	}

	log.Info("Successfully reconciled Project",
		"name", project.Name,
		"namespace", namespace.Name,
		"phase", project.Status.Phase,
		"uuid", project.Labels[validation.LabelResourceUUID])
	return ctrl.Result{}, nil
}

// handleProjectDeletion handles the deletion of a project and its resources
func (r *ProjectReconciler) handleProjectDeletion(ctx context.Context, project *platformv1alpha1.Project) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	log.Info("Handling project deletion", "project", project.Name)

	// Delete the project namespace (ignore NotFound errors for idempotency)
	if err := r.NamespaceManager.DeleteProjectNamespace(ctx, project); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "Failed to delete project namespace")
			return ctrl.Result{}, err
		}
		log.Info("Project namespace was already deleted", "project", project.Name)
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(project, ProjectFinalizerName)
	if err := r.Update(ctx, project); err != nil {
		log.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}

	log.Info("Successfully deleted project", "project", project.Name)
	return ctrl.Result{}, nil
}

// updateStatusWithError updates the project status with error information
func (r *ProjectReconciler) updateStatusWithError(ctx context.Context, project *platformv1alpha1.Project, phase, message string) {
	project.Status.Phase = phase
	project.Status.Message = message
	now := metav1.Now()
	project.Status.LastReconcileTime = &now

	if err := r.Status().Update(ctx, project); err != nil {
		log := logf.FromContext(ctx)
		log.Error(err, "Failed to update project status with error")
	}

}

// generateRegistryPassword generates a random 32-character password for registry credentials
func generateRegistryPassword() (string, error) {
	// Generate 32 random bytes
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	// Encode as base64 for safe string representation
	return base64.StdEncoding.EncodeToString(bytes), nil
}

// ensureRegistryCredentials creates registry credentials secret for the namespace if it doesn't exist
func (r *ProjectReconciler) ensureRegistryCredentials(ctx context.Context, namespaceName string) error {
	log := logf.FromContext(ctx)

	secretName := fmt.Sprintf("%s-registry-credentials", namespaceName)

	// Check if secret already exists
	secret := &corev1.Secret{}
	err := r.Get(ctx, client.ObjectKey{
		Namespace: namespaceName,
		Name:      secretName,
	}, secret)

	if err == nil {
		// Secret already exists
		log.Info("Registry credentials secret already exists", "namespace", namespaceName)
		return nil
	}

	if !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check registry credentials secret: %w", err)
	}

	// Generate random password
	password, err := generateRegistryPassword()
	if err != nil {
		return fmt.Errorf("failed to generate registry password: %w", err)
	}

	// Create secret
	secret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespaceName,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "kibaship-operator",
				"app.kubernetes.io/component":  "registry-credentials",
			},
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"username": namespaceName, // Username MUST match namespace name
			"password": password,
		},
	}

	if err := r.Create(ctx, secret); err != nil {
		return fmt.Errorf("failed to create registry credentials secret: %w", err)
	}

	log.Info("Created registry credentials secret",
		"namespace", namespaceName,
		"secret", secretName)
	return nil
}

// ensureRegistryCACertificate copies the registry CA certificate from registry namespace to project namespace
func (r *ProjectReconciler) ensureRegistryCACertificate(ctx context.Context, namespaceName string) error {
	log := logf.FromContext(ctx)

	const (
		registryNamespace = "registry"
		registryTLSSecret = "registry-tls"
		caCertSecretName  = "registry-ca-cert"
	)

	// Check if CA cert secret already exists in project namespace
	existingSecret := &corev1.Secret{}
	err := r.Get(ctx, client.ObjectKey{
		Namespace: namespaceName,
		Name:      caCertSecretName,
	}, existingSecret)

	if err == nil {
		// Secret already exists
		log.Info("Registry CA certificate secret already exists", "namespace", namespaceName)
		return nil
	}

	if !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check registry CA certificate secret: %w", err)
	}

	// Get the registry TLS secret from registry namespace
	registryTLS := &corev1.Secret{}
	err = r.Get(ctx, client.ObjectKey{
		Namespace: registryNamespace,
		Name:      registryTLSSecret,
	}, registryTLS)

	if err != nil {
		return fmt.Errorf("failed to get registry TLS secret: %w", err)
	}

	// Extract CA certificate
	caCert, ok := registryTLS.Data["ca.crt"]
	if !ok {
		return fmt.Errorf("ca.crt not found in registry TLS secret")
	}

	// Create CA certificate secret in project namespace
	caSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      caCertSecretName,
			Namespace: namespaceName,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "kibaship-operator",
				"app.kubernetes.io/component":  "registry-ca",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"ca.crt": caCert,
		},
	}

	if err := r.Create(ctx, caSecret); err != nil {
		return fmt.Errorf("failed to create registry CA certificate secret: %w", err)
	}

	log.Info("Created registry CA certificate secret",
		"namespace", namespaceName,
		"secret", caCertSecretName)
	return nil
}

// ensureRegistryDockerConfig creates Docker config secret for registry authentication
func (r *ProjectReconciler) ensureRegistryDockerConfig(ctx context.Context, namespaceName string) error {
	log := logf.FromContext(ctx)

	const (
		dockerConfigSecretName = "registry-docker-config"
		registryURL            = "registry.registry.svc.cluster.local"
	)

	// Check if Docker config secret already exists
	existingSecret := &corev1.Secret{}
	err := r.Get(ctx, client.ObjectKey{
		Namespace: namespaceName,
		Name:      dockerConfigSecretName,
	}, existingSecret)

	if err == nil {
		// Secret already exists
		log.Info("Registry Docker config secret already exists", "namespace", namespaceName)
		return nil
	}

	if !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check registry Docker config secret: %w", err)
	}

	// Get the registry credentials secret to extract username and password
	credentialsSecretName := fmt.Sprintf("%s-registry-credentials", namespaceName)
	credentialsSecret := &corev1.Secret{}
	err = r.Get(ctx, client.ObjectKey{
		Namespace: namespaceName,
		Name:      credentialsSecretName,
	}, credentialsSecret)

	if err != nil {
		return fmt.Errorf("failed to get registry credentials secret: %w", err)
	}

	username := string(credentialsSecret.Data["username"])
	password := string(credentialsSecret.Data["password"])

	// Generate base64-encoded auth string (username:password)
	authString := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password)))

	// Create Docker config JSON
	dockerConfigJSON := fmt.Sprintf(`{
  "auths": {
    "%s": {
      "username": "%s",
      "password": "%s",
      "auth": "%s"
    }
  }
}`, registryURL, username, password, authString)

	// Create Docker config secret with key "config.json" for BuildKit compatibility
	dockerConfigSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dockerConfigSecretName,
			Namespace: namespaceName,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "kibaship-operator",
				"app.kubernetes.io/component":  "registry-docker-config",
			},
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"config.json": dockerConfigJSON,
		},
	}

	if err := r.Create(ctx, dockerConfigSecret); err != nil {
		return fmt.Errorf("failed to create registry Docker config secret: %w", err)
	}

	log.Info("Created registry Docker config secret",
		"namespace", namespaceName,
		"secret", dockerConfigSecretName)
	return nil
}

// emitProjectPhaseChange sends a webhook if Notifier is configured and the phase actually changed.
func (r *ProjectReconciler) emitProjectPhaseChange(ctx context.Context, project *platformv1alpha1.Project, prev, next string) {
	if r.Notifier == nil {
		return
	}
	if prev == next {
		return
	}
	evt := webhooks.ProjectStatusEvent{
		Type:          "project.status.changed",
		PreviousPhase: prev,
		NewPhase:      next,
		Project:       *project,
		Timestamp:     time.Now().UTC(),
	}
	_ = r.Notifier.NotifyProjectStatusChange(ctx, evt)
}

// NewProjectReconciler creates a new ProjectReconciler with required dependencies
func NewProjectReconciler(k8sClient client.Client, scheme *runtime.Scheme) *ProjectReconciler {
	return &ProjectReconciler{
		Client:           k8sClient,
		Scheme:           scheme,
		NamespaceManager: NewNamespaceManager(k8sClient),
		Validator:        NewProjectValidator(k8sClient),
		Notifier:         webhooks.NoopNotifier{},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1alpha1.Project{}).
		Named("project").
		Complete(r)
}

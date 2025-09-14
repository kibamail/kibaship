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

package v1alpha1

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var applicationdomainlog = logf.Log.WithName("applicationdomain-resource")

// SetupWebhookWithManager will setup the manager to manage the webhooks
func (r *ApplicationDomain) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/validate-platform-operator-kibaship-com-v1alpha1-applicationdomain,mutating=false,failurePolicy=fail,sideEffects=None,groups=platform.operator.kibaship.com,resources=applicationdomains,verbs=create;update,versions=v1alpha1,name=vapplicationdomain.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = &ApplicationDomain{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *ApplicationDomain) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	applicationdomainlog.Info("validate create", "name", r.Name)

	return nil, r.validateApplicationDomain()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *ApplicationDomain) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	applicationdomainlog.Info("validate update", "name", r.Name)

	return nil, r.validateApplicationDomain()
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (r *ApplicationDomain) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	applicationdomainlog.Info("validate delete", "name", r.Name)

	// No validation needed for deletion
	return nil, nil
}

// validateApplicationDomain performs comprehensive validation of the ApplicationDomain resource
func (r *ApplicationDomain) validateApplicationDomain() error {
	var allErrs field.ErrorList

	// Validate domain format
	if err := r.validateDomainFormat(); err != nil {
		allErrs = append(allErrs, err)
	}

	// Validate port range
	if err := r.validatePort(); err != nil {
		allErrs = append(allErrs, err)
	}

	// Validate application reference
	if err := r.validateApplicationRef(); err != nil {
		allErrs = append(allErrs, err)
	}

	// Type constraints are handled during creation in the controller

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: "platform.operator.kibaship.com", Kind: "ApplicationDomain"},
		r.Name, allErrs)
}

// validateDomainFormat validates that the domain is a valid DNS name
func (r *ApplicationDomain) validateDomainFormat() *field.Error {
	domain := r.Spec.Domain

	// Check for empty domain
	if domain == "" {
		return field.Required(field.NewPath("spec", "domain"), "domain is required")
	}

	// Validate DNS format - RFC 1123 compliant
	// Must be lowercase, alphanumeric with hyphens, dots allowed
	domainRegex := regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)*$`)
	if !domainRegex.MatchString(domain) {
		return field.Invalid(
			field.NewPath("spec", "domain"),
			domain,
			"domain must be a valid DNS name (lowercase, alphanumeric, hyphens, dots only)")
	}

	// Check domain length constraints
	if len(domain) > 253 {
		return field.Invalid(
			field.NewPath("spec", "domain"),
			domain,
			"domain must be 253 characters or less")
	}

	// Validate individual labels (parts between dots)
	labels := strings.Split(domain, ".")
	for _, label := range labels {
		if len(label) == 0 {
			return field.Invalid(
				field.NewPath("spec", "domain"),
				domain,
				"domain labels cannot be empty")
		}
		if len(label) > 63 {
			return field.Invalid(
				field.NewPath("spec", "domain"),
				domain,
				fmt.Sprintf("domain label '%s' must be 63 characters or less", label))
		}
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return field.Invalid(
				field.NewPath("spec", "domain"),
				domain,
				fmt.Sprintf("domain label '%s' cannot start or end with hyphen", label))
		}
	}

	return nil
}

// validatePort validates that the port is within valid range
func (r *ApplicationDomain) validatePort() *field.Error {
	port := r.Spec.Port

	if port < 1 || port > 65535 {
		return field.Invalid(
			field.NewPath("spec", "port"),
			port,
			"port must be between 1 and 65535")
	}

	// Warn about common restricted ports (informational)
	restrictedPorts := []int32{22, 23, 25, 53, 80, 110, 443, 993, 995}
	for _, restrictedPort := range restrictedPorts {
		if port == restrictedPort {
			// This is just a warning - we still allow it but log it
			applicationdomainlog.Info("using common system port",
				"port", port,
				"domain", r.Spec.Domain,
				"warning", "this port may conflict with system services")
			break
		}
	}

	return nil
}

// validateApplicationRef validates the application reference
func (r *ApplicationDomain) validateApplicationRef() *field.Error {
	appRef := r.Spec.ApplicationRef

	if appRef.Name == "" {
		return field.Required(
			field.NewPath("spec", "applicationRef", "name"),
			"application reference name is required")
	}

	// Validate application name format (should match Application naming convention)
	appNameRegex := regexp.MustCompile(`^project-[a-z0-9]([a-z0-9-]*[a-z0-9])?-app-[a-z0-9]([a-z0-9-]*[a-z0-9])?-kibaship-com$`)
	if !appNameRegex.MatchString(appRef.Name) {
		return field.Invalid(
			field.NewPath("spec", "applicationRef", "name"),
			appRef.Name,
			"application name must follow the pattern: project-<project-name>-app-<app-name>-kibaship-com")
	}

	return nil
}

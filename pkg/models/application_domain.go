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

package models

import (
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/kibamail/kibaship/api/v1alpha1"
	"github.com/kibamail/kibaship/pkg/validation"
)

type ApplicationDomainType string
type ApplicationDomainPhase string

const (
	ApplicationDomainTypeDefault ApplicationDomainType = "default"
	ApplicationDomainTypeCustom  ApplicationDomainType = "custom"
)

const (
	ApplicationDomainPhasePending ApplicationDomainPhase = "Pending"
	ApplicationDomainPhaseReady   ApplicationDomainPhase = "Ready"
	ApplicationDomainPhaseFailed  ApplicationDomainPhase = "Failed"
)

// ApplicationDomainCreateRequest represents the request to create a new application domain
type ApplicationDomainCreateRequest struct {
	ApplicationSlug string                `json:"applicationSlug" example:"abc123de" validate:"required"`
	Domain          string                `json:"domain" example:"my-app.example.com" validate:"required"`
	Port            int32                 `json:"port" example:"3000" validate:"required,min=1,max=65535"`
	Type            ApplicationDomainType `json:"type" example:"custom"`
	Default         bool                  `json:"default" example:"false"`
	TLSEnabled      bool                  `json:"tlsEnabled" example:"true"`
}

// ApplicationDomainResponse represents the application domain data returned to clients
type ApplicationDomainResponse struct {
	UUID             string                 `json:"uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Slug             string                 `json:"slug" example:"def456gh"`
	ApplicationUUID  string                 `json:"applicationUuid" example:"550e8400-e29b-41d4-a716-446655440001"`
	ApplicationSlug  string                 `json:"applicationSlug" example:"abc123de"`
	ProjectUUID      string                 `json:"projectUuid" example:"550e8400-e29b-41d4-a716-446655440002"`
	Domain           string                 `json:"domain" example:"my-app.example.com"`
	Port             int32                  `json:"port" example:"3000"`
	Type             ApplicationDomainType  `json:"type" example:"custom"`
	Default          bool                   `json:"default" example:"false"`
	TLSEnabled       bool                   `json:"tlsEnabled" example:"true"`
	Phase            ApplicationDomainPhase `json:"phase" example:"Pending"`
	CertificateReady bool                   `json:"certificateReady" example:"false"`
	IngressReady     bool                   `json:"ingressReady" example:"false"`
	DNSConfigured    bool                   `json:"dnsConfigured" example:"false"`
	CreatedAt        time.Time              `json:"createdAt" example:"2023-01-01T12:00:00Z"`
	UpdatedAt        time.Time              `json:"updatedAt" example:"2023-01-01T12:00:00Z"`
}

// ApplicationDomain represents the internal application domain model
type ApplicationDomain struct {
	UUID             string
	Slug             string
	ApplicationUUID  string
	ApplicationSlug  string
	ProjectUUID      string
	Domain           string
	Port             int32
	Type             ApplicationDomainType
	Default          bool
	TLSEnabled       bool
	Phase            ApplicationDomainPhase
	CertificateReady bool
	IngressReady     bool
	DNSConfigured    bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// NewApplicationDomain creates a new application domain with the given parameters
func NewApplicationDomain(applicationUUID, applicationSlug, projectUUID, slug, domain string, port int32, domainType ApplicationDomainType, isDefault, tlsEnabled bool) *ApplicationDomain {
	now := time.Now()
	return &ApplicationDomain{
		UUID:             uuid.New().String(),
		Slug:             slug,
		ApplicationUUID:  applicationUUID,
		ApplicationSlug:  applicationSlug,
		ProjectUUID:      projectUUID,
		Domain:           domain,
		Port:             port,
		Type:             domainType,
		Default:          isDefault,
		TLSEnabled:       tlsEnabled,
		Phase:            ApplicationDomainPhasePending,
		CertificateReady: false,
		IngressReady:     false,
		DNSConfigured:    false,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// ToResponse converts the internal application domain to a response model
func (ad *ApplicationDomain) ToResponse() ApplicationDomainResponse {
	return ApplicationDomainResponse{
		UUID:             ad.UUID,
		Slug:             ad.Slug,
		ApplicationUUID:  ad.ApplicationUUID,
		ApplicationSlug:  ad.ApplicationSlug,
		ProjectUUID:      ad.ProjectUUID,
		Domain:           ad.Domain,
		Port:             ad.Port,
		Type:             ad.Type,
		Default:          ad.Default,
		TLSEnabled:       ad.TLSEnabled,
		Phase:            ad.Phase,
		CertificateReady: ad.CertificateReady,
		IngressReady:     ad.IngressReady,
		DNSConfigured:    ad.DNSConfigured,
		CreatedAt:        ad.CreatedAt,
		UpdatedAt:        ad.UpdatedAt,
	}
}

// Validate validates the application domain create request
func (req *ApplicationDomainCreateRequest) Validate() *ValidationErrors {
	var validationErrors []ValidationError

	if req.ApplicationSlug == "" {
		validationErrors = append(validationErrors, ValidationError{
			Field:   "applicationSlug",
			Message: "Application slug is required",
		})
	} else if !validation.ValidateSlug(req.ApplicationSlug) {
		validationErrors = append(validationErrors, ValidationError{
			Field:   "applicationSlug",
			Message: "Application slug must be 8 characters long and contain only lowercase letters and numbers",
		})
	}

	if req.Domain == "" {
		validationErrors = append(validationErrors, ValidationError{
			Field:   "domain",
			Message: "Domain is required",
		})
	} else if !isValidDomain(req.Domain) {
		validationErrors = append(validationErrors, ValidationError{
			Field:   "domain",
			Message: "Domain must be a valid domain name",
		})
	}

	if req.Port < 1 || req.Port > 65535 {
		validationErrors = append(validationErrors, ValidationError{
			Field:   "port",
			Message: "Port must be between 1 and 65535",
		})
	}

	if req.Type != "" && req.Type != ApplicationDomainTypeDefault && req.Type != ApplicationDomainTypeCustom {
		validationErrors = append(validationErrors, ValidationError{
			Field:   "type",
			Message: "Type must be either 'default' or 'custom'",
		})
	}

	if len(validationErrors) > 0 {
		return &ValidationErrors{
			Errors: validationErrors,
		}
	}

	return nil
}

// isValidDomain validates if a string is a valid domain name
func isValidDomain(domain string) bool {
	// Domain pattern matching the CRD validation
	pattern := regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)*$`)
	return pattern.MatchString(domain)
}

// ConvertFromCRD converts a Kubernetes ApplicationDomain CRD to internal model
func (ad *ApplicationDomain) ConvertFromCRD(crd *v1alpha1.ApplicationDomain, applicationSlug string) {
	ad.UUID = crd.GetLabels()[validation.LabelResourceUUID]
	ad.Slug = crd.GetLabels()[validation.LabelResourceSlug]
	ad.ApplicationUUID = crd.GetLabels()[validation.LabelApplicationUUID]
	ad.ApplicationSlug = applicationSlug
	ad.ProjectUUID = crd.GetLabels()[validation.LabelProjectUUID]
	ad.Domain = crd.Spec.Domain
	ad.Port = crd.Spec.Port
	ad.Type = ApplicationDomainType(crd.Spec.Type)
	ad.Default = crd.Spec.Default
	ad.TLSEnabled = crd.Spec.TLSEnabled
	ad.Phase = ApplicationDomainPhase(crd.Status.Phase)
	ad.CertificateReady = crd.Status.CertificateReady
	ad.IngressReady = crd.Status.IngressReady
	ad.DNSConfigured = crd.Status.DNSConfigured
	ad.CreatedAt = crd.CreationTimestamp.Time
	ad.UpdatedAt = crd.CreationTimestamp.Time
}

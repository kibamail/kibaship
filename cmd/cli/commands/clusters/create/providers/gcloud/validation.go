package gcloud

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

// GCloudValidator provides Google Cloud-specific validation logic
type GCloudValidator struct {
	validator *validator.Validate
}

// NewValidator creates a new Google Cloud validator with custom validation rules
func NewValidator() *GCloudValidator {
	v := validator.New()

	// Register Google Cloud-specific custom validators
	v.RegisterValidation("gcloud_service_account", validateServiceAccountKey)
	v.RegisterValidation("gcloud_project_id", validateProjectID)
	v.RegisterValidation("gcloud_region", validateGCloudRegion)

	return &GCloudValidator{validator: v}
}

// Config represents Google Cloud provider configuration with validation tags
type Config struct {
	ServiceAccountKey string `validate:"required,file,gcloud_service_account" json:"service_account_key"`
	ProjectID         string `validate:"required,gcloud_project_id" json:"project_id"`
	Region            string `validate:"required,gcloud_region" json:"region"`
}

// ServiceAccountKeyFile represents the structure of a Google Cloud service account key file
type ServiceAccountKeyFile struct {
	Type                    string `json:"type"`
	ProjectID               string `json:"project_id"`
	PrivateKeyID            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	ClientID                string `json:"client_id"`
	AuthURI                 string `json:"auth_uri"`
	TokenURI                string `json:"token_uri"`
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url"`
	ClientX509CertURL       string `json:"client_x509_cert_url"`
}

// Validate validates the Google Cloud configuration
func (v *GCloudValidator) Validate(config *Config) error {
	if err := v.validator.Struct(config); err != nil {
		return v.formatValidationError(err)
	}
	return nil
}

// validateServiceAccountKey validates Google Cloud service account key file
func validateServiceAccountKey(fl validator.FieldLevel) bool {
	filePath := fl.Field().String()

	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}

	// Parse as JSON
	var keyFile ServiceAccountKeyFile
	if err := json.Unmarshal(data, &keyFile); err != nil {
		return false
	}

	// Validate required fields
	if keyFile.Type != "service_account" {
		return false
	}

	if keyFile.ProjectID == "" || keyFile.PrivateKey == "" || keyFile.ClientEmail == "" {
		return false
	}

	// Validate client email format
	emailPattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.iam\.gserviceaccount\.com$`
	matched, _ := regexp.MatchString(emailPattern, keyFile.ClientEmail)

	return matched
}

// validateProjectID validates Google Cloud project ID format
func validateProjectID(fl validator.FieldLevel) bool {
	projectID := fl.Field().String()

	// GCP project IDs: 6-30 characters, lowercase letters, digits, and hyphens
	// Must start with a letter, cannot end with a hyphen
	if len(projectID) < 6 || len(projectID) > 30 {
		return false
	}

	// Check format: starts with letter, contains only lowercase letters, digits, hyphens
	// Cannot end with hyphen
	matched, _ := regexp.MatchString("^[a-z][a-z0-9-]*[a-z0-9]$", projectID)
	return matched
}

// validateGCloudRegion validates Google Cloud region format
func validateGCloudRegion(fl validator.FieldLevel) bool {
	region := fl.Field().String()

	// GCP regions follow pattern: us-central1, europe-west1, asia-southeast1, etc.
	matched, _ := regexp.MatchString("^[a-z]+-[a-z]+[0-9]+$", region)
	return matched
}

// formatValidationError formats Google Cloud validation errors with detailed messages
func (v *GCloudValidator) formatValidationError(err error) error {
	var errorMessages []string

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fieldError := range validationErrors {
			errorMessages = append(errorMessages, v.getGCloudFieldErrorMessage(fieldError))
		}
	} else {
		errorMessages = append(errorMessages, err.Error())
	}

	return fmt.Errorf("Google Cloud validation failed:\n  • %s", strings.Join(errorMessages, "\n  • "))
}

// getGCloudFieldErrorMessage returns Google Cloud-specific error messages
func (v *GCloudValidator) getGCloudFieldErrorMessage(fe validator.FieldError) string {
	field := fe.Field()

	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required for Google Cloud provider", field)
	case "file":
		return fmt.Sprintf("%s must be a valid file path", field)
	case "gcloud_service_account":
		return fmt.Sprintf("%s must be a valid Google Cloud service account key file (JSON format)", field)
	case "gcloud_project_id":
		return fmt.Sprintf("%s must be a valid Google Cloud project ID (6-30 characters, lowercase, starts with letter)", field)
	case "gcloud_region":
		return fmt.Sprintf("%s must be a valid Google Cloud region (e.g., us-central1, europe-west1)", field)
	default:
		return fmt.Sprintf("%s validation failed: %s", field, fe.Tag())
	}
}

// GetSupportedRegions returns a list of supported Google Cloud regions
func GetSupportedRegions() []string {
	return []string{
		"us-central1", "us-east1", "us-east4", "us-west1", "us-west2", "us-west3", "us-west4",
		"europe-west1", "europe-west2", "europe-west3", "europe-west4", "europe-west6", "europe-north1",
		"asia-east1", "asia-east2", "asia-northeast1", "asia-northeast2", "asia-northeast3",
		"asia-south1", "asia-southeast1", "asia-southeast2",
		"australia-southeast1",
		"southamerica-east1",
	}
}

// ValidateRegionSupported checks if the region is in the supported list
func ValidateRegionSupported(region string) bool {
	supportedRegions := GetSupportedRegions()
	for _, supportedRegion := range supportedRegions {
		if region == supportedRegion {
			return true
		}
	}
	return false
}

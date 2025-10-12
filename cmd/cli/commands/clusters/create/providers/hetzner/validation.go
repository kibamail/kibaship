package hetzner

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

// HetznerValidator provides Hetzner Cloud-specific validation logic
type HetznerValidator struct {
	validator *validator.Validate
}

// NewValidator creates a new Hetzner validator with custom validation rules
func NewValidator() *HetznerValidator {
	v := validator.New()

	// Register Hetzner-specific custom validators
	v.RegisterValidation("hetzner_token", validateHetznerToken)

	return &HetznerValidator{validator: v}
}

// Config represents Hetzner Cloud provider configuration with validation tags
type Config struct {
	Token string `validate:"required,hetzner_token" json:"token"`
}

// Validate validates the Hetzner Cloud configuration
func (v *HetznerValidator) Validate(config *Config) error {
	if err := v.validator.Struct(config); err != nil {
		return v.formatValidationError(err)
	}
	return nil
}

// validateHetznerToken validates Hetzner Cloud API token format
func validateHetznerToken(fl validator.FieldLevel) bool {
	token := fl.Field().String()

	// Hetzner tokens are typically 64 characters long and alphanumeric
	if len(token) < 32 || len(token) > 128 {
		return false
	}

	// Check if it contains valid characters (alphanumeric and some special chars)
	matched, _ := regexp.MatchString("^[A-Za-z0-9_-]+$", token)
	return matched
}

// formatValidationError formats Hetzner validation errors with detailed messages
func (v *HetznerValidator) formatValidationError(err error) error {
	var errorMessages []string

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fieldError := range validationErrors {
			errorMessages = append(errorMessages, v.getHetznerFieldErrorMessage(fieldError))
		}
	} else {
		errorMessages = append(errorMessages, err.Error())
	}

	return fmt.Errorf("Hetzner Cloud validation failed:\n  • %s", strings.Join(errorMessages, "\n  • "))
}

// getHetznerFieldErrorMessage returns Hetzner-specific error messages
func (v *HetznerValidator) getHetznerFieldErrorMessage(fe validator.FieldError) string {
	field := fe.Field()

	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required for Hetzner Cloud provider", field)
	case "hetzner_token":
		return fmt.Sprintf("%s must be a valid Hetzner Cloud API token (32-128 characters, alphanumeric)", field)
	default:
		return fmt.Sprintf("%s validation failed: %s", field, fe.Tag())
	}
}

// GetSupportedLocations returns a list of supported Hetzner Cloud locations
func GetSupportedLocations() []string {
	return []string{
		"nbg1", // Nuremberg, Germany
		"fsn1", // Falkenstein, Germany
		"hel1", // Helsinki, Finland
		"ash",  // Ashburn, VA, USA
		"hil",  // Hillsboro, OR, USA
	}
}

// ValidateLocationSupported checks if the location is in the supported list
func ValidateLocationSupported(location string) bool {
	supportedLocations := GetSupportedLocations()
	for _, supportedLocation := range supportedLocations {
		if location == supportedLocation {
			return true
		}
	}
	return false
}

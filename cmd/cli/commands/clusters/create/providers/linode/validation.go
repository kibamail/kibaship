package linode

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

// LinodeValidator provides Linode-specific validation logic
type LinodeValidator struct {
	validator *validator.Validate
}

// NewValidator creates a new Linode validator with custom validation rules
func NewValidator() *LinodeValidator {
	v := validator.New()

	// Register Linode-specific custom validators
	v.RegisterValidation("linode_token", validateLinodeToken)

	return &LinodeValidator{validator: v}
}

// Config represents Linode provider configuration with validation tags
type Config struct {
	Token string `validate:"required,linode_token" json:"token"`
}

// Validate validates the Linode configuration
func (v *LinodeValidator) Validate(config *Config) error {
	if err := v.validator.Struct(config); err != nil {
		return v.formatValidationError(err)
	}
	return nil
}

// validateLinodeToken validates Linode API token format
func validateLinodeToken(fl validator.FieldLevel) bool {
	token := fl.Field().String()

	// Linode tokens are typically 64 characters long and alphanumeric
	if len(token) != 64 {
		return false
	}

	// Check if it's a valid alphanumeric string
	matched, _ := regexp.MatchString("^[A-Za-z0-9]+$", token)
	return matched
}

// formatValidationError formats Linode validation errors with detailed messages
func (v *LinodeValidator) formatValidationError(err error) error {
	var errorMessages []string

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fieldError := range validationErrors {
			errorMessages = append(errorMessages, v.getLinodeFieldErrorMessage(fieldError))
		}
	} else {
		errorMessages = append(errorMessages, err.Error())
	}

	return fmt.Errorf("Linode validation failed:\n  • %s", strings.Join(errorMessages, "\n  • "))
}

// getLinodeFieldErrorMessage returns Linode-specific error messages
func (v *LinodeValidator) getLinodeFieldErrorMessage(fe validator.FieldError) string {
	field := fe.Field()

	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required for Linode provider", field)
	case "linode_token":
		return fmt.Sprintf("%s must be a valid Linode API token (64 character alphanumeric string)", field)
	default:
		return fmt.Sprintf("%s validation failed: %s", field, fe.Tag())
	}
}

// GetSupportedRegions returns a list of supported Linode regions
func GetSupportedRegions() []string {
	return []string{
		"us-east",      // Newark, NJ
		"us-central",   // Dallas, TX
		"us-west",      // Fremont, CA
		"us-southeast", // Atlanta, GA
		"eu-west",      // London, UK
		"eu-central",   // Frankfurt, DE
		"ap-south",     // Singapore
		"ap-northeast", // Tokyo, JP
		"ap-west",      // Mumbai, IN
		"ca-central",   // Toronto, CA
		"ap-southeast", // Sydney, AU
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

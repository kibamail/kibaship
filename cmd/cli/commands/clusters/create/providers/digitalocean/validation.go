package digitalocean

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

// DigitalOceanValidator provides DigitalOcean-specific validation logic
type DigitalOceanValidator struct {
	validator *validator.Validate
}

// NewValidator creates a new DigitalOcean validator with custom validation rules
func NewValidator() *DigitalOceanValidator {
	v := validator.New()

	// Register DigitalOcean-specific custom validators
	v.RegisterValidation("do_token", validateDOToken)

	return &DigitalOceanValidator{validator: v}
}

// Config represents DigitalOcean provider configuration with validation tags
type Config struct {
	Token string `validate:"required,do_token" json:"token"`
}

// Validate validates the DigitalOcean configuration
func (v *DigitalOceanValidator) Validate(config *Config) error {
	if err := v.validator.Struct(config); err != nil {
		return v.formatValidationError(err)
	}
	return nil
}

// validateDOToken validates DigitalOcean API token format
func validateDOToken(fl validator.FieldLevel) bool {
	token := fl.Field().String()

	// DigitalOcean tokens are typically 64 characters long and hexadecimal
	if len(token) != 64 {
		return false
	}

	// Check if it's a valid hexadecimal string
	matched, _ := regexp.MatchString("^[a-f0-9]+$", token)
	return matched
}

// formatValidationError formats DigitalOcean validation errors with detailed messages
func (v *DigitalOceanValidator) formatValidationError(err error) error {
	var errorMessages []string

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fieldError := range validationErrors {
			errorMessages = append(errorMessages, v.getDOFieldErrorMessage(fieldError))
		}
	} else {
		errorMessages = append(errorMessages, err.Error())
	}

	return fmt.Errorf("DigitalOcean validation failed:\n  • %s", strings.Join(errorMessages, "\n  • "))
}

// getDOFieldErrorMessage returns DigitalOcean-specific error messages
func (v *DigitalOceanValidator) getDOFieldErrorMessage(fe validator.FieldError) string {
	field := fe.Field()

	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required for DigitalOcean provider", field)
	case "do_token":
		return fmt.Sprintf("%s must be a valid DigitalOcean API token (64 character hexadecimal string)", field)
	default:
		return fmt.Sprintf("%s validation failed: %s", field, fe.Tag())
	}
}

// GetSupportedRegions returns a list of supported DigitalOcean regions
func GetSupportedRegions() []string {
	return []string{
		"nyc1", "nyc2", "nyc3",
		"ams2", "ams3",
		"sfo1", "sfo2", "sfo3",
		"sgp1",
		"lon1",
		"fra1",
		"tor1",
		"blr1",
		"syd1",
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

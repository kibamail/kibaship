package hetznerrobot

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

// HetznerRobotValidator provides Hetzner Robot-specific validation logic
type HetznerRobotValidator struct {
	validator *validator.Validate
}

// NewValidator creates a new Hetzner Robot validator with custom validation rules
func NewValidator() *HetznerRobotValidator {
	v := validator.New()

	// Register Hetzner Robot-specific custom validators
	v.RegisterValidation("hetzner_robot_username", validateHetznerRobotUsername)
	v.RegisterValidation("hetzner_robot_password", validateHetznerRobotPassword)
	v.RegisterValidation("hetzner_cloud_token", validateHetznerCloudToken)

	return &HetznerRobotValidator{validator: v}
}

// Config represents Hetzner Robot provider configuration with validation tags
type Config struct {
	Username   string `validate:"required,hetzner_robot_username" json:"username"`
	Password   string `validate:"required,hetzner_robot_password" json:"password"`
	CloudToken string `validate:"required,hetzner_cloud_token" json:"cloud_token"`
}

// Validate validates the Hetzner Robot configuration
func (v *HetznerRobotValidator) Validate(config *Config) error {
	if err := v.validator.Struct(config); err != nil {
		return v.formatValidationError(err)
	}
	return nil
}

// validateHetznerRobotUsername validates Hetzner Robot username format
func validateHetznerRobotUsername(fl validator.FieldLevel) bool {
	username := fl.Field().String()

	// Hetzner Robot usernames are typically alphanumeric with some special characters
	if len(username) < 3 || len(username) > 50 {
		return false
	}

	// Check if it contains valid characters
	matched, _ := regexp.MatchString("^[A-Za-z0-9._-]+$", username)
	return matched
}

// validateHetznerRobotPassword validates Hetzner Robot password requirements
func validateHetznerRobotPassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	// Basic password requirements: minimum 8 characters
	if len(password) < 8 {
		return false
	}

	// Should contain at least one letter and one number
	hasLetter, _ := regexp.MatchString("[A-Za-z]", password)
	hasNumber, _ := regexp.MatchString("[0-9]", password)

	return hasLetter && hasNumber
}

// validateHetznerCloudToken validates Hetzner Cloud token format (for Robot integration)
func validateHetznerCloudToken(fl validator.FieldLevel) bool {
	token := fl.Field().String()

	// Hetzner Cloud tokens are typically 64 characters long and alphanumeric
	if len(token) < 32 || len(token) > 128 {
		return false
	}

	// Check if it contains valid characters (alphanumeric and some special chars)
	matched, _ := regexp.MatchString("^[A-Za-z0-9_-]+$", token)
	return matched
}

// formatValidationError formats Hetzner Robot validation errors with detailed messages
func (v *HetznerRobotValidator) formatValidationError(err error) error {
	var errorMessages []string

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fieldError := range validationErrors {
			errorMessages = append(errorMessages, v.getHetznerRobotFieldErrorMessage(fieldError))
		}
	} else {
		errorMessages = append(errorMessages, err.Error())
	}

	return fmt.Errorf("Hetzner Robot validation failed:\n  • %s", strings.Join(errorMessages, "\n  • "))
}

// getHetznerRobotFieldErrorMessage returns Hetzner Robot-specific error messages
func (v *HetznerRobotValidator) getHetznerRobotFieldErrorMessage(fe validator.FieldError) string {
	field := fe.Field()

	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required for Hetzner Robot provider", field)
	case "hetzner_robot_username":
		return fmt.Sprintf("%s must be a valid Hetzner Robot username (3-50 characters, alphanumeric with ._-)", field)
	case "hetzner_robot_password":
		return fmt.Sprintf("%s must be a valid password (minimum 8 characters, at least one letter and one number)", field)
	case "hetzner_cloud_token":
		return fmt.Sprintf("%s must be a valid Hetzner Cloud token (32-128 characters, alphanumeric)", field)
	default:
		return fmt.Sprintf("%s validation failed: %s", field, fe.Tag())
	}
}

// GetSupportedDatacenters returns a list of supported Hetzner Robot datacenters
func GetSupportedDatacenters() []string {
	return []string{
		"nbg1-dc3",  // Nuremberg DC3
		"nbg1-dc8",  // Nuremberg DC8
		"nbg1-dc14", // Nuremberg DC14
		"fsn1-dc8",  // Falkenstein DC8
		"fsn1-dc14", // Falkenstein DC14
		"hel1-dc2",  // Helsinki DC2
	}
}

// ValidateDatacenterSupported checks if the datacenter is in the supported list
func ValidateDatacenterSupported(datacenter string) bool {
	supportedDatacenters := GetSupportedDatacenters()
	for _, supportedDatacenter := range supportedDatacenters {
		if datacenter == supportedDatacenter {
			return true
		}
	}
	return false
}

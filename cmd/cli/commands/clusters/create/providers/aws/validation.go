package aws

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

// AWSValidator provides AWS-specific validation logic
type AWSValidator struct {
	validator *validator.Validate
}

// NewValidator creates a new AWS validator with custom validation rules
func NewValidator() *AWSValidator {
	v := validator.New()

	// Register AWS-specific custom validators
	_ = v.RegisterValidation("aws_access_key", validateAWSAccessKey)
	_ = v.RegisterValidation("aws_secret_key", validateAWSSecretKey)
	_ = v.RegisterValidation("aws_region", validateAWSRegion)

	return &AWSValidator{validator: v}
}

// Config represents AWS provider configuration with validation tags
type Config struct {
	AccessKeyID     string `validate:"required,aws_access_key" json:"access_key_id"`
	SecretAccessKey string `validate:"required,aws_secret_key" json:"secret_access_key"`
	Region          string `validate:"required,aws_region" json:"region"`
}

// Validate validates the AWS configuration
func (v *AWSValidator) Validate(config *Config) error {
	if err := v.validator.Struct(config); err != nil {
		return v.formatValidationError(err)
	}
	return nil
}

// validateAWSAccessKey validates AWS Access Key ID format
func validateAWSAccessKey(fl validator.FieldLevel) bool {
	accessKey := fl.Field().String()

	// AWS Access Key ID format: 20 characters, starts with AKIA, ASIA, or AROA
	if len(accessKey) != 20 {
		return false
	}

	validPrefixes := []string{"AKIA", "ASIA", "AROA"}
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(accessKey, prefix) {
			// Check if remaining characters are alphanumeric
			remaining := accessKey[4:]
			matched, _ := regexp.MatchString("^[A-Z0-9]+$", remaining)
			return matched
		}
	}

	return false
}

// validateAWSSecretKey validates AWS Secret Access Key format
func validateAWSSecretKey(fl validator.FieldLevel) bool {
	secretKey := fl.Field().String()

	// AWS Secret Access Key: 40 characters, base64-like format
	if len(secretKey) != 40 {
		return false
	}

	// Check if it contains valid base64 characters
	matched, _ := regexp.MatchString("^[A-Za-z0-9+/]+$", secretKey)
	return matched
}

// validateAWSRegion validates AWS region format
func validateAWSRegion(fl validator.FieldLevel) bool {
	region := fl.Field().String()

	// AWS regions follow pattern: us-east-1, eu-west-2, ap-southeast-1, etc.
	matched, _ := regexp.MatchString("^[a-z]{2,3}-[a-z]+-[0-9]+$", region)
	return matched
}

// formatValidationError formats AWS validation errors with detailed messages
func (v *AWSValidator) formatValidationError(err error) error {
	var errorMessages []string

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fieldError := range validationErrors {
			errorMessages = append(errorMessages, v.getAWSFieldErrorMessage(fieldError))
		}
	} else {
		errorMessages = append(errorMessages, err.Error())
	}

	return fmt.Errorf("AWS validation failed:\n  • %s", strings.Join(errorMessages, "\n  • "))
}

// getAWSFieldErrorMessage returns AWS-specific error messages
func (v *AWSValidator) getAWSFieldErrorMessage(fe validator.FieldError) string {
	field := fe.Field()

	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required for AWS provider", field)
	case "aws_access_key":
		return fmt.Sprintf("%s must be a valid AWS Access Key ID (20 characters, starting with AKIA/ASIA/AROA)", field)
	case "aws_secret_key":
		return fmt.Sprintf("%s must be a valid AWS Secret Access Key (40 characters, base64 format)", field)
	case "aws_region":
		return fmt.Sprintf("%s must be a valid AWS region (e.g., us-east-1, eu-west-2)", field)
	default:
		return fmt.Sprintf("%s validation failed: %s", field, fe.Tag())
	}
}

// GetSupportedRegions returns a list of supported AWS regions
func GetSupportedRegions() []string {
	return []string{
		"us-east-1", "us-east-2", "us-west-1", "us-west-2",
		"eu-west-1", "eu-west-2", "eu-west-3", "eu-central-1", "eu-north-1",
		"ap-southeast-1", "ap-southeast-2", "ap-northeast-1", "ap-northeast-2", "ap-south-1",
		"ca-central-1", "sa-east-1",
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

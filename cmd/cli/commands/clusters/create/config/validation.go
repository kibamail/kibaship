package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/cobra"

	"github.com/kibamail/kibaship/cmd/cli/internal/styles"
)

// validateFile checks if a file exists and is readable
func validateFile(fl validator.FieldLevel) bool {
	filePath := fl.Field().String()
	if filePath == "" {
		return true // Allow empty files for optional fields
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return false
	}

	return !info.IsDir()
}

// validateDomainName validates domain/subdomain format for cluster names
func validateDomainName(fl validator.FieldLevel) bool {
	domain := fl.Field().String()

	// Domain must be at least 4 characters (e.g., a.co)
	if len(domain) < 4 {
		return false
	}

	// Domain must contain at least one dot
	if !strings.Contains(domain, ".") {
		return false
	}

	// Split by dots and validate each part
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return false
	}

	// Each part must be valid (alphanumeric and hyphens, not starting/ending with hyphen)
	for _, part := range parts {
		if len(part) == 0 {
			return false
		}

		// Must start and end with alphanumeric
		if !isAlphanumeric(part[0]) || !isAlphanumeric(part[len(part)-1]) {
			return false
		}

		// All characters must be alphanumeric or hyphen
		for _, char := range part {
			if !isAlphanumeric(byte(char)) && char != '-' {
				return false
			}
		}
	}

	return true
}

// isAlphanumeric checks if a byte is alphanumeric
func isAlphanumeric(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

// validatePaaSFeatures validates PaaS features selection
func validatePaaSFeatures(fl validator.FieldLevel) bool {
	features := fl.Field().String()

	// Empty is allowed (will use default)
	if features == "" {
		return true
	}

	// Handle "none" case
	if features == "none" {
		return true
	}

	// Split by comma and validate each feature
	featureList := strings.Split(features, ",")
	validFeatures := map[string]bool{
		"mysql":    true,
		"valkey":   true,
		"postgres": true,
	}

	// Check each feature
	for _, feature := range featureList {
		feature = strings.TrimSpace(feature)
		if feature == "" {
			return false // Empty feature not allowed
		}
		if !validFeatures[feature] {
			return false // Invalid feature
		}
	}

	// Check for duplicates
	seenFeatures := make(map[string]bool)
	for _, feature := range featureList {
		feature = strings.TrimSpace(feature)
		if seenFeatures[feature] {
			return false // Duplicate feature
		}
		seenFeatures[feature] = true
	}

	return true
}

// validateKindStorage validates Kind cluster storage configuration
func validateKindStorage(fl validator.FieldLevel) bool {
	// Get the storage value
	storageStr := fl.Field().String()
	if storageStr == "" {
		return false // Storage is required for Kind clusters
	}

	// Parse storage as integer
	var storage int
	if _, err := fmt.Sscanf(storageStr, "%d", &storage); err != nil {
		return false // Must be a valid number
	}

	// Get the parent struct to access nodes
	parent := fl.Parent()
	if !parent.IsValid() {
		return false
	}

	// Get nodes field
	nodesField := parent.FieldByName("Nodes")
	if !nodesField.IsValid() {
		return false
	}

	nodesStr := nodesField.String()
	var nodes int
	if _, err := fmt.Sscanf(nodesStr, "%d", &nodes); err != nil {
		return false // Nodes must be a valid number
	}

	// Apply storage validation logic:
	// - Minimum 50 GB per node if 3+ nodes
	// - Minimum 75 GB per node if 1-2 nodes
	// - This ensures at least 75 GB total storage in cluster
	var minStoragePerNode int
	if nodes >= 3 {
		minStoragePerNode = 50
	} else {
		minStoragePerNode = 75
	}

	return storage >= minStoragePerNode
}

// extractCoreFlags extracts and validates core command flags
func extractCoreFlags(cmd *cobra.Command) *CreateConfig {
	config := &CreateConfig{}

	// Get core flags
	provider, _ := cmd.Flags().GetString("provider")
	configuration, _ := cmd.Flags().GetString("configuration")
	domain, _ := cmd.Flags().GetString("domain")
	email, _ := cmd.Flags().GetString("email")
	paasFeatures, _ := cmd.Flags().GetString("paas-features")

	config.Provider = provider
	config.Configuration = configuration
	config.Domain = domain
	config.Email = email
	config.PaaSFeatures = paasFeatures

	// Set default PaaS features if not specified
	if config.PaaSFeatures == "" {
		config.PaaSFeatures = "mysql,valkey,postgres"
	}

	return config
}

// extractTerraformStateFlags extracts Terraform state configuration flags
func extractTerraformStateFlags(cmd *cobra.Command) (string, string, string, string) {
	stateS3Bucket, _ := cmd.Flags().GetString("state-s3-bucket")
	stateS3Region, _ := cmd.Flags().GetString("state-s3-region")
	stateS3AccessKey, _ := cmd.Flags().GetString("state-s3-access-key")
	stateS3AccessSecret, _ := cmd.Flags().GetString("state-s3-access-secret")
	return stateS3Bucket, stateS3Region, stateS3AccessKey, stateS3AccessSecret
}

// buildProviderConfig builds provider-specific configuration
func buildProviderConfig(config *CreateConfig, cmd *cobra.Command) error {
	switch config.Provider {
	case ProviderAWS:
		awsConfig, err := buildAWSConfig(cmd)
		if err != nil {
			return fmt.Errorf("AWS configuration validation failed: %w", err)
		}
		config.AWS = awsConfig

	case ProviderDigitalOcean:
		doConfig, err := buildDigitalOceanConfig(cmd)
		if err != nil {
			return fmt.Errorf("digital Ocean configuration validation failed: %w", err)
		}
		config.DigitalOcean = doConfig

	case ProviderHetzner:
		hetznerConfig, err := buildHetznerConfig(cmd)
		if err != nil {
			return fmt.Errorf("hetzner configuration validation failed: %w", err)
		}
		config.Hetzner = hetznerConfig

	case ProviderHetznerRobot:
		hetznerRobotConfig, err := buildHetznerRobotConfig(cmd)
		if err != nil {
			return fmt.Errorf("hetzner Robot configuration validation failed: %w", err)
		}
		config.HetznerRobot = hetznerRobotConfig

	case ProviderLinode:
		linodeConfig, err := buildLinodeConfig(cmd)
		if err != nil {
			return fmt.Errorf("linode configuration validation failed: %w", err)
		}
		config.Linode = linodeConfig

	case ProviderGCloud:
		gcloudConfig, err := buildGCloudConfig(cmd)
		if err != nil {
			return fmt.Errorf("google Cloud configuration validation failed: %w", err)
		}
		config.GCloud = gcloudConfig

	default:
		return fmt.Errorf("unsupported provider: %s", config.Provider)
	}
	return nil
}

// ValidateCreateCommand validates the core create command flags and builds the configuration
func ValidateCreateCommand(cmd *cobra.Command) (*CreateConfig, error) {
	// Extract core flags
	config := extractCoreFlags(cmd)

	// Derive cluster name from domain by replacing dots with dashes
	if config.Domain != "" {
		config.Name = strings.ReplaceAll(config.Domain, ".", "-")
	}

	// Get Terraform state flags
	stateS3Bucket, stateS3Region, stateS3AccessKey, stateS3AccessSecret := extractTerraformStateFlags(cmd)

	// Validate that we have either configuration file OR provider flags
	if config.Configuration == "" && config.Provider == "" {
		return nil, fmt.Errorf("must specify either --configuration file or --provider with credentials")
	}

	// If both configuration file and provider are specified, configuration file takes precedence
	// and provider flag will be ignored (this is intentional for CLI flag overrides)

	// If using configuration file, load and parse it
	if config.Configuration != "" {
		if err := validate.Var(config.Configuration, "required,file"); err != nil {
			return nil, fmt.Errorf("configuration file validation failed: %w", err)
		}

		// Load configuration from YAML file
		yamlConfig, err := LoadConfigFromYAML(config.Configuration)
		if err != nil {
			return nil, fmt.Errorf("failed to load YAML configuration: %w", err)
		}

		// Override with CLI flags if provided (CLI flags take precedence)
		if config.Domain != "" {
			yamlConfig.Domain = config.Domain
			// Derive cluster name from domain
			yamlConfig.Name = strings.ReplaceAll(config.Domain, ".", "-")
		}
		if config.Email != "" {
			yamlConfig.Email = config.Email
		}
		if config.PaaSFeatures != "" {
			yamlConfig.PaaSFeatures = config.PaaSFeatures
		}

		// Override Terraform state with CLI flags if provided
		if stateS3Bucket != "" {
			yamlConfig.TerraformState.S3Bucket = stateS3Bucket
		}
		if stateS3Region != "" {
			yamlConfig.TerraformState.S3Region = stateS3Region
		}
		if stateS3AccessKey != "" {
			yamlConfig.TerraformState.S3AccessKey = stateS3AccessKey
		}
		if stateS3AccessSecret != "" {
			yamlConfig.TerraformState.S3AccessSecret = stateS3AccessSecret
		}

		// Validate the final configuration
		if err := validate.Struct(yamlConfig); err != nil {
			return nil, formatValidationError(err)
		}

		return yamlConfig, nil
	}

	// Validate provider selection
	if err := validate.Var(config.Provider,
		"required,oneof=aws digital-ocean hetzner hetzner-robot linode gcloud"); err != nil {
		return nil, fmt.Errorf("invalid provider: must be one of: aws, digital-ocean, hetzner, hetzner-robot, linode, gcloud")
	}

	// Validate Terraform state configuration
	if err := validateTerraformStateFlags(stateS3Bucket, stateS3Region,
		stateS3AccessKey, stateS3AccessSecret); err != nil {
		return nil, fmt.Errorf("terraform state configuration validation failed: %w", err)
	}
	config.TerraformState = &TerraformStateConfig{
		S3Bucket:       stateS3Bucket,
		S3Region:       stateS3Region,
		S3AccessKey:    stateS3AccessKey,
		S3AccessSecret: stateS3AccessSecret,
	}

	// Build provider-specific configuration based on selected provider
	if err := buildProviderConfig(config, cmd); err != nil {
		return nil, err
	}

	// Validate required fields are present in the final configuration
	if config.Domain == "" {
		return nil, fmt.Errorf("cluster domain is required (specify in YAML or use --domain flag)")
	}
	if config.Name == "" {
		return nil, fmt.Errorf("cluster name could not be derived from domain")
	}
	if config.Email == "" {
		return nil, fmt.Errorf("email is required (specify in YAML or use --email flag)")
	}
	if config.TerraformState == nil || config.TerraformState.S3Bucket == "" {
		return nil, fmt.Errorf("terraform state S3 bucket is required (specify in YAML or use --state-s3-bucket flag)")
	}
	if config.TerraformState.S3Region == "" {
		return nil, fmt.Errorf("terraform state S3 region is required (specify in YAML or use --state-s3-region flag)")
	}
	if config.TerraformState.S3AccessKey == "" {
		return nil, fmt.Errorf("terraform state S3 access key is required " +
			"(specify in YAML or use --state-s3-access-key flag)")
	}
	if config.TerraformState.S3AccessSecret == "" {
		return nil, fmt.Errorf("terraform state S3 access secret is required " +
			"(specify in YAML or use --state-s3-access-secret flag)")
	}

	// Final validation of the complete configuration
	if err := validate.Struct(config); err != nil {
		return nil, formatValidationError(err)
	}

	return config, nil
}

// buildAWSConfig extracts and validates AWS-specific flags
func buildAWSConfig(cmd *cobra.Command) (*AWSConfig, error) {
	accessKeyID, _ := cmd.Flags().GetString("provider-aws-access-key-id")
	secretAccessKey, _ := cmd.Flags().GetString("provider-aws-secret-access-key")
	region, _ := cmd.Flags().GetString("provider-aws-region")

	config := &AWSConfig{
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		Region:          region,
	}

	// Validate required AWS fields
	if accessKeyID == "" {
		return nil, fmt.Errorf("--provider-aws-access-key-id is required for AWS provider")
	}
	if secretAccessKey == "" {
		return nil, fmt.Errorf("--provider-aws-secret-access-key is required for AWS provider")
	}
	if region == "" {
		return nil, fmt.Errorf("--provider-aws-region is required for AWS provider")
	}

	return config, nil
}

// buildDigitalOceanConfig extracts and validates DigitalOcean-specific flags
func buildDigitalOceanConfig(cmd *cobra.Command) (*DigitalOceanConfig, error) {
	token, _ := cmd.Flags().GetString("provider-digital-ocean-token")
	nodes, _ := cmd.Flags().GetString("provider-digital-ocean-nodes")
	nodesSize, _ := cmd.Flags().GetString("provider-digital-ocean-nodes-size")
	region, _ := cmd.Flags().GetString("provider-digital-ocean-region")

	config := &DigitalOceanConfig{
		Token:     token,
		Nodes:     nodes,
		NodesSize: nodesSize,
		Region:    region,
	}

	if token == "" {
		return nil, fmt.Errorf("--provider-digital-ocean-token is required for DigitalOcean provider")
	}
	if nodes == "" {
		return nil, fmt.Errorf("--provider-digital-ocean-nodes is required for DigitalOcean provider")
	}
	if nodesSize == "" {
		return nil, fmt.Errorf("--provider-digital-ocean-nodes-size is required for DigitalOcean provider")
	}
	if region == "" {
		return nil, fmt.Errorf("--provider-digital-ocean-region is required for DigitalOcean provider")
	}

	return config, nil
}

// buildHetznerConfig extracts and validates Hetzner Cloud-specific flags
func buildHetznerConfig(cmd *cobra.Command) (*HetznerConfig, error) {
	token, _ := cmd.Flags().GetString("provider-hetzner-token")

	config := &HetznerConfig{
		Token: token,
	}

	if token == "" {
		return nil, fmt.Errorf("--provider-hetzner-token is required for Hetzner provider")
	}

	return config, nil
}

// buildHetznerRobotConfig extracts and validates Hetzner Robot-specific flags
func buildHetznerRobotConfig(cmd *cobra.Command) (*HetznerRobotConfig, error) {
	username, _ := cmd.Flags().GetString("provider-hetzner-robot-username")
	password, _ := cmd.Flags().GetString("provider-hetzner-robot-password")
	cloudToken, _ := cmd.Flags().GetString("provider-hetzner-robot-cloud-token")

	config := &HetznerRobotConfig{
		Username:   username,
		Password:   password,
		CloudToken: cloudToken,
	}

	if username == "" {
		return nil, fmt.Errorf("--provider-hetzner-robot-username is required for Hetzner Robot provider")
	}
	if password == "" {
		return nil, fmt.Errorf("--provider-hetzner-robot-password is required for Hetzner Robot provider")
	}
	if cloudToken == "" {
		return nil, fmt.Errorf("--provider-hetzner-robot-cloud-token is required for Hetzner Robot provider")
	}

	return config, nil
}

// buildLinodeConfig extracts and validates Linode-specific flags
func buildLinodeConfig(cmd *cobra.Command) (*LinodeConfig, error) {
	token, _ := cmd.Flags().GetString("provider-linode-token")

	config := &LinodeConfig{
		Token: token,
	}

	if token == "" {
		return nil, fmt.Errorf("--provider-linode-token is required for Linode provider")
	}

	return config, nil
}

// buildGCloudConfig extracts and validates Google Cloud-specific flags
func buildGCloudConfig(cmd *cobra.Command) (*GCloudConfig, error) {
	serviceAccountKey, _ := cmd.Flags().GetString("provider-gcloud-service-account-key")
	projectID, _ := cmd.Flags().GetString("provider-gcloud-project-id")
	region, _ := cmd.Flags().GetString("provider-gcloud-region")

	config := &GCloudConfig{
		ServiceAccountKey: serviceAccountKey,
		ProjectID:         projectID,
		Region:            region,
	}

	if serviceAccountKey == "" {
		return nil, fmt.Errorf("--provider-gcloud-service-account-key is required for Google Cloud provider")
	}
	if projectID == "" {
		return nil, fmt.Errorf("--provider-gcloud-project-id is required for Google Cloud provider")
	}
	if region == "" {
		return nil, fmt.Errorf("--provider-gcloud-region is required for Google Cloud provider")
	}

	return config, nil
}

// formatValidationError formats validation errors with beautiful styling
func formatValidationError(err error) error {
	var errorMessages []string

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fieldError := range validationErrors {
			errorMessages = append(errorMessages, getFieldErrorMessage(fieldError))
		}
	} else {
		errorMessages = append(errorMessages, err.Error())
	}

	// Format with styling
	formattedError := fmt.Sprintf("%s\n", styles.HelpStyle.Render("Validation Errors:"))
	for _, msg := range errorMessages {
		formattedError += fmt.Sprintf("  %s %s\n",
			styles.CommandStyle.Render("•"),
			styles.DescriptionStyle.Render(msg))
	}

	return fmt.Errorf("%s", formattedError)
}

// getFieldErrorMessage returns a user-friendly error message for field validation errors
func getFieldErrorMessage(fe validator.FieldError) string {
	field := strings.ToLower(fe.Field())

	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, fe.Param())
	case "file":
		return fmt.Sprintf("%s must be a valid file path", field)
	case "required_with":
		return fmt.Sprintf("%s is required when using this provider", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "domain_name":
		return fmt.Sprintf("%s must be a valid domain name (e.g., app.kibaship.com, paas.kibaship.app)", field)
	case "paas_features":
		return fmt.Sprintf("%s must be a valid combination of: mysql, valkey, postgres, or 'none' "+
			"(comma-separated, no duplicates)", field)
	default:
		return fmt.Sprintf("%s validation failed: %s", field, fe.Tag())
	}
}

// validateTerraformStateFlags validates the Terraform state configuration flags
func validateTerraformStateFlags(bucket, region, accessKey, accessSecret string) error {
	// All Terraform state flags are required
	if bucket == "" {
		return fmt.Errorf("--state-s3-bucket is required for Terraform state storage")
	}
	if region == "" {
		return fmt.Errorf("--state-s3-region is required for Terraform state storage")
	}
	if accessKey == "" {
		return fmt.Errorf("--state-s3-access-key is required for Terraform state storage")
	}
	if accessSecret == "" {
		return fmt.Errorf("--state-s3-access-secret is required for Terraform state storage")
	}

	// Validate S3 bucket name format
	if err := validate.Var(bucket, "required,min=3,max=63"); err != nil {
		return fmt.Errorf("S3 bucket name must be between 3 and 63 characters")
	}

	// Validate AWS region format for S3
	if err := validate.Var(region, "required"); err != nil {
		return fmt.Errorf("S3 region is required")
	}

	// Basic validation for access credentials (non-empty)
	if len(accessKey) < 16 {
		return fmt.Errorf("S3 access key appears to be invalid (too short)")
	}
	if len(accessSecret) < 32 {
		return fmt.Errorf("S3 access secret appears to be invalid (too short)")
	}

	return nil
}

package clusters

import (
	"fmt"
	"os"
	"path/filepath"
)

// DestroyOptions holds the configuration for cluster destruction
type DestroyOptions struct {
	Name  string
	Force bool
}

// DestroyCluster destroys a Kibaship cluster and cleans up all resources
func DestroyCluster(
	opts DestroyOptions,
	printStep func(int, string),
	printProgress func(string),
	printSuccess func(string),
	printError func(string),
	printInfo func(string),
	printWarning func(string),
) error {
	// Validate cluster name
	if err := ValidateClusterName(opts.Name); err != nil {
		return err
	}

	// Check if cluster exists
	exists, err := ClusterExistsByName(opts.Name)
	if err != nil {
		return fmt.Errorf("failed to check if cluster exists: %w", err)
	}

	if !exists {
		printWarning(fmt.Sprintf("Cluster '%s' not found", opts.Name))
		printInfo("Use 'kibaship clusters list' to see available clusters")
		return nil
	}

	// Get cluster info before destruction
	clusterInfo, err := GetKibashipClusterInfo(opts.Name)
	if err != nil {
		printWarning(fmt.Sprintf("Could not retrieve cluster info: %v", err))
	} else {
		printInfo(fmt.Sprintf("Cluster: %s", clusterInfo.Name))
		printInfo(fmt.Sprintf("Status: %s", clusterInfo.Status))
		printInfo(fmt.Sprintf("Nodes: %d", clusterInfo.Nodes))
		if clusterInfo.Age != "" {
			printInfo(fmt.Sprintf("Age: %s", clusterInfo.Age))
		}
		fmt.Println()
	}

	// Confirm destruction unless force flag is used
	if !opts.Force {
		printWarning("This will permanently delete the cluster and all its data!")
		printInfo("Use --force to skip this confirmation")
		return fmt.Errorf("cluster destruction cancelled (use --force to proceed)")
	}

	printStep(1, fmt.Sprintf("Destroying cluster '%s'...", opts.Name))
	printProgress("This may take a few minutes...")

	// Delete the Kind cluster
	if err := DeleteKindCluster(opts.Name); err != nil {
		printError(fmt.Sprintf("Failed to delete cluster: %v", err))
		return err
	}
	printSuccess(fmt.Sprintf("Cluster '%s' destroyed successfully", opts.Name))

	printStep(2, "Cleaning up configuration files...")
	if err := cleanupClusterFiles(opts.Name); err != nil {
		printWarning(fmt.Sprintf("Failed to clean up some files: %v", err))
		// Don't fail the operation for cleanup issues
	} else {
		printSuccess("Configuration files cleaned up")
	}

	printStep(3, "Verifying cluster removal...")
	// Verify cluster is actually gone
	if exists, err := ClusterExistsByName(opts.Name); err != nil {
		printWarning(fmt.Sprintf("Could not verify cluster removal: %v", err))
	} else if exists {
		printError("Cluster still exists after deletion attempt")
		return fmt.Errorf("cluster deletion may have failed")
	} else {
		printSuccess("Cluster removal verified")
	}

	printSuccess("🎉 Cluster destruction complete!")
	printInfo("The cluster and all its resources have been permanently removed")
	printInfo("Use 'kibaship clusters list' to see remaining clusters")

	return nil
}

// cleanupClusterFiles removes temporary files associated with the cluster
func cleanupClusterFiles(clusterName string) error {
	var errors []error

	// Clean up Kind config file
	configPath := GetKindConfigPath(clusterName)
	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		errors = append(errors, fmt.Errorf("failed to remove config file %s: %w", configPath, err))
	}

	// Clean up any other temporary files that might exist
	tempDir := os.TempDir()
	fullClusterName := GetKibashipClusterName(clusterName)

	// Look for any files with the cluster name pattern
	pattern := filepath.Join(tempDir, fmt.Sprintf("*%s*", fullClusterName))
	matches, err := filepath.Glob(pattern)
	if err == nil {
		for _, match := range matches {
			if err := os.Remove(match); err != nil && !os.IsNotExist(err) {
				errors = append(errors, fmt.Errorf("failed to remove file %s: %w", match, err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %v", errors)
	}

	return nil
}

// DestroyAllClusters destroys all Kibaship clusters
func DestroyAllClusters(
	force bool,
	printStep func(int, string),
	printProgress func(string),
	printSuccess func(string),
	printError func(string),
	printInfo func(string),
	printWarning func(string),
) error {
	clusters, err := ListKibashipClusters()
	if err != nil {
		return fmt.Errorf("failed to list clusters: %w", err)
	}

	if len(clusters) == 0 {
		printInfo("No Kibaship clusters found to destroy")
		return nil
	}

	printInfo(fmt.Sprintf("Found %d cluster(s) to destroy:", len(clusters)))
	for _, fullName := range clusters {
		userFriendlyName := GetUserClusterName(fullName)
		printInfo(fmt.Sprintf("  - %s", userFriendlyName))
	}
	fmt.Println()

	if !force {
		printWarning("This will permanently delete ALL Kibaship clusters and their data!")
		printInfo("Use --force to skip this confirmation")
		return fmt.Errorf("cluster destruction cancelled (use --force to proceed)")
	}

	successCount := 0
	errorCount := 0

	for i, fullName := range clusters {
		userFriendlyName := GetUserClusterName(fullName)
		printStep(i+1, fmt.Sprintf("Destroying cluster '%s'...", userFriendlyName))

		opts := DestroyOptions{
			Name:  userFriendlyName,
			Force: true, // Already confirmed above
		}

		if err := DestroyCluster(
			opts, printStep, printProgress, printSuccess, printError, printInfo, printWarning,
		); err != nil {
			printError(fmt.Sprintf("Failed to destroy cluster '%s': %v", userFriendlyName, err))
			errorCount++
		} else {
			successCount++
		}
		fmt.Println()
	}

	if errorCount > 0 {
		printWarning(fmt.Sprintf("Destroyed %d cluster(s), %d failed", successCount, errorCount))
		return fmt.Errorf("%d cluster(s) failed to destroy", errorCount)
	} else {
		printSuccess(fmt.Sprintf("🎉 All %d cluster(s) destroyed successfully!", successCount))
		return nil
	}
}

// ValidateDestroyOptions validates the cluster destruction options
func ValidateDestroyOptions(opts DestroyOptions) error {
	if opts.Name == "" {
		return fmt.Errorf("cluster name cannot be empty")
	}

	return ValidateClusterName(opts.Name)
}

// GetClusterDestructionInfo returns information about what will be destroyed
func GetClusterDestructionInfo(clusterName string) (map[string]interface{}, error) {
	info := make(map[string]interface{})

	// Check if cluster exists
	exists, err := ClusterExistsByName(clusterName)
	if err != nil {
		return nil, err
	}

	info["exists"] = exists
	if !exists {
		return info, nil
	}

	// Get cluster details
	clusterInfo, err := GetKibashipClusterInfo(clusterName)
	if err != nil {
		return nil, err
	}

	info["name"] = clusterInfo.Name
	info["full_name"] = clusterInfo.FullName
	info["status"] = clusterInfo.Status
	info["nodes"] = clusterInfo.Nodes
	info["age"] = clusterInfo.Age
	info["context"] = clusterInfo.Context

	// List what will be destroyed
	resources := []string{
		"Kind cluster and all nodes",
		"All deployed applications and services",
		"All persistent volumes and data",
		"Cluster configuration files",
		"kubectl context",
	}

	info["resources_to_destroy"] = resources

	return info, nil
}

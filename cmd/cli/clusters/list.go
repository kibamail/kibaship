package clusters

import (
	"fmt"
	"sort"
	"strings"
)

// ListClusters lists all Kibaship-managed clusters with detailed information
func ListClusters(
	printInfo func(string),
	printTable func([]string, [][]string),
	printSuccess func(string),
	printWarning func(string),
) error {
	clusters, err := ListKibashipClusterInfo()
	if err != nil {
		return fmt.Errorf("failed to list clusters: %w", err)
	}

	if len(clusters) == 0 {
		printWarning("No Kibaship clusters found")
		printInfo("Use 'kibaship clusters create <name>' to create a new cluster")
		return nil
	}

	// Sort clusters by name
	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].Name < clusters[j].Name
	})

	// Prepare table data
	headers := []string{"NAME", "STATUS", "NODES", "AGE", "ENDPOINT"}
	rows := make([][]string, 0, len(clusters))

	for _, cluster := range clusters {
		endpoint := cluster.Endpoint
		if endpoint == "" {
			endpoint = "N/A"
		}
		// Truncate long endpoints for better display
		if len(endpoint) > 30 {
			endpoint = endpoint[:27] + "..."
		}

		rows = append(rows, []string{
			cluster.Name,
			cluster.Status,
			fmt.Sprintf("%d", cluster.Nodes),
			cluster.Age,
			endpoint,
		})
	}

	// Display the table
	printTable(headers, rows)

	// Display summary
	runningCount := 0
	for _, cluster := range clusters {
		if cluster.Status == "Running" {
			runningCount++
		}
	}

	printSuccess(fmt.Sprintf("Found %d cluster(s) (%d running)", len(clusters), runningCount))
	printInfo("Use 'kibaship clusters create <name>' to create a new cluster")
	printInfo("Use 'kubectl config use-context <context>' to switch clusters")

	return nil
}

// GetClusterSummary returns a summary of cluster information for display
func GetClusterSummary() (map[string]interface{}, error) {
	clusters, err := ListKibashipClusterInfo()
	if err != nil {
		return nil, err
	}

	summary := make(map[string]interface{})
	summary["total"] = len(clusters)

	runningCount := 0
	totalNodes := 0

	for _, cluster := range clusters {
		if cluster.Status == "Running" {
			runningCount++
		}
		totalNodes += cluster.Nodes
	}

	summary["running"] = runningCount
	summary["stopped"] = len(clusters) - runningCount
	summary["total_nodes"] = totalNodes

	return summary, nil
}

// ValidateClusterName validates that a cluster name is valid for Kibaship
func ValidateClusterName(name string) error {
	if name == "" {
		return fmt.Errorf("cluster name cannot be empty")
	}

	// Remove prefix if present for validation
	userFriendlyName := GetUserClusterName(name)

	if len(userFriendlyName) < 1 {
		return fmt.Errorf("cluster name must be at least 1 character long")
	}

	if len(userFriendlyName) > 50 {
		return fmt.Errorf("cluster name must be less than 50 characters long")
	}

	// Check for valid characters (alphanumeric, hyphens, underscores)
	for _, char := range userFriendlyName {
		if (char < 'a' || char > 'z') &&
			(char < 'A' || char > 'Z') &&
			(char < '0' || char > '9') &&
			char != '-' && char != '_' {
			return fmt.Errorf("cluster name can only contain alphanumeric characters, hyphens, and underscores")
		}
	}

	// Cannot start or end with hyphen
	if strings.HasPrefix(userFriendlyName, "-") || strings.HasSuffix(userFriendlyName, "-") {
		return fmt.Errorf("cluster name cannot start or end with a hyphen")
	}

	return nil
}

// ClusterExists checks if a Kibaship cluster exists
func ClusterExistsByName(name string) (bool, error) {
	clusters, err := ListKibashipClusters()
	if err != nil {
		return false, err
	}

	fullName := GetKibashipClusterName(name)
	for _, cluster := range clusters {
		if cluster == fullName {
			return true, nil
		}
	}

	return false, nil
}

// GetClusterByName returns detailed information about a specific cluster
func GetClusterByName(name string) (*ClusterInfo, error) {
	exists, err := ClusterExistsByName(name)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, fmt.Errorf("cluster '%s' not found", name)
	}

	return GetKibashipClusterInfo(name)
}

// FormatClusterInfo formats cluster information for display
func FormatClusterInfo(cluster *ClusterInfo) []string {
	var info []string

	info = append(info, fmt.Sprintf("Name: %s", cluster.Name))
	info = append(info, fmt.Sprintf("Full Name: %s", cluster.FullName))
	info = append(info, fmt.Sprintf("Status: %s", cluster.Status))
	info = append(info, fmt.Sprintf("Nodes: %d", cluster.Nodes))
	info = append(info, fmt.Sprintf("Age: %s", cluster.Age))
	info = append(info, fmt.Sprintf("Context: %s", cluster.Context))

	if cluster.Endpoint != "" {
		info = append(info, fmt.Sprintf("Endpoint: %s", cluster.Endpoint))
	}

	if !cluster.CreatedAt.IsZero() {
		info = append(info, fmt.Sprintf("Created: %s", cluster.CreatedAt.Format("2006-01-02 15:04:05")))
	}

	return info
}

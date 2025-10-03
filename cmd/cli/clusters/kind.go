package clusters

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

const (
	// StatusRunning represents a running cluster status
	StatusRunning = "Running"
	// StatusUnknown represents an unknown status
	StatusUnknown = "unknown"
)

const (
	// KibashipClusterPrefix is the prefix used for all Kibaship-managed Kind clusters
	KibashipClusterPrefix = "kibaship-"
)

// GetKibashipClusterName returns the full cluster name with Kibaship prefix
func GetKibashipClusterName(userProvidedName string) string {
	if strings.HasPrefix(userProvidedName, KibashipClusterPrefix) {
		return userProvidedName
	}
	return KibashipClusterPrefix + userProvidedName
}

// GetUserClusterName returns the user-friendly name without the Kibaship prefix
func GetUserClusterName(fullClusterName string) string {
	if strings.HasPrefix(fullClusterName, KibashipClusterPrefix) {
		return strings.TrimPrefix(fullClusterName, KibashipClusterPrefix)
	}
	return fullClusterName
}

// IsKibashipCluster checks if a cluster name belongs to Kibaship
func IsKibashipCluster(clusterName string) bool {
	return strings.HasPrefix(clusterName, KibashipClusterPrefix)
}

// ClusterConfig holds the configuration for creating a Kind cluster
type ClusterConfig struct {
	Name              string
	ControlPlaneNodes int
	WorkerNodes       int
	ConfigPath        string
}

// KindClusterConfigTemplate is the template for Kind cluster configuration
const KindClusterConfigTemplate = `# Kind cluster configuration for Kibaship operator
# Disables default CNI so we can install Cilium explicitly
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
networking:
  disableDefaultCNI: true
  kubeProxyMode: none
nodes:
{{- range $i := .ControlPlaneRange }}
  - role: control-plane
    kubeadmConfigPatches:
      - |
        kind: InitConfiguration
        nodeRegistration:
          kubeletExtraArgs:
            node-labels: "ingress.kibaship.com/ready=true"
{{- end }}
{{- range $i := .WorkerRange }}
  - role: worker
{{- end }}
`

// GenerateKindConfig creates a Kind cluster configuration file
func GenerateKindConfig(config ClusterConfig) error {
	tmpl, err := template.New("kindconfig").Parse(KindClusterConfigTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse Kind config template: %w", err)
	}

	// Create template data
	data := struct {
		ControlPlaneRange []int
		WorkerRange       []int
	}{
		ControlPlaneRange: make([]int, config.ControlPlaneNodes),
		WorkerRange:       make([]int, config.WorkerNodes),
	}

	// Initialize ranges
	for i := 0; i < config.ControlPlaneNodes; i++ {
		data.ControlPlaneRange[i] = i
	}
	for i := 0; i < config.WorkerNodes; i++ {
		data.WorkerRange[i] = i
	}

	// Create config file
	file, err := os.Create(config.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to create Kind config file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close config file: %v\n", closeErr)
		}
	}()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to write Kind config: %w", err)
	}

	return nil
}

// CreateKindCluster creates a new Kind cluster with the specified configuration
func CreateKindCluster(config ClusterConfig) error {
	// Ensure cluster name has Kibaship prefix
	fullClusterName := GetKibashipClusterName(config.Name)

	// Check if cluster already exists
	if exists, err := ClusterExists(fullClusterName); err != nil {
		return fmt.Errorf("failed to check if cluster exists: %w", err)
	} else if exists {
		return fmt.Errorf("cluster '%s' already exists", config.Name)
	}

	// Generate Kind config
	if err := GenerateKindConfig(config); err != nil {
		return err
	}

	// Create the cluster with full name
	cmd := exec.Command("kind", "create", "cluster",
		"--name", fullClusterName,
		"--config", config.ConfigPath)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create Kind cluster: %w", err)
	}

	return nil
}

// ClusterExists checks if a Kind cluster with the given name already exists
func ClusterExists(name string) (bool, error) {
	cmd := exec.Command("kind", "get", "clusters")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to list Kind clusters: %w", err)
	}

	clusters := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, cluster := range clusters {
		if strings.TrimSpace(cluster) == name {
			return true, nil
		}
	}

	return false, nil
}

// DeleteKindCluster deletes a Kind cluster
func DeleteKindCluster(name string) error {
	fullClusterName := GetKibashipClusterName(name)
	cmd := exec.Command("kind", "delete", "cluster", "--name", fullClusterName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete Kind cluster: %w", err)
	}

	return nil
}

// ListKindClusters returns a list of existing Kind clusters
func ListKindClusters() ([]string, error) {
	cmd := exec.Command("kind", "get", "clusters")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list Kind clusters: %w", err)
	}

	if strings.TrimSpace(string(output)) == "" {
		return []string{}, nil
	}

	clusters := strings.Split(strings.TrimSpace(string(output)), "\n")
	var result []string
	for _, cluster := range clusters {
		if trimmed := strings.TrimSpace(cluster); trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result, nil
}

// ListKibashipClusters returns a list of Kibaship-managed Kind clusters
func ListKibashipClusters() ([]string, error) {
	allClusters, err := ListKindClusters()
	if err != nil {
		return nil, err
	}

	var kibashipClusters []string
	for _, cluster := range allClusters {
		if IsKibashipCluster(cluster) {
			kibashipClusters = append(kibashipClusters, cluster)
		}
	}

	return kibashipClusters, nil
}

// GetKindConfigPath returns the path where the Kind config should be stored
func GetKindConfigPath(clusterName string) string {
	fullClusterName := GetKibashipClusterName(clusterName)
	return filepath.Join(os.TempDir(), fmt.Sprintf("%s-kind.config.yaml", fullClusterName))
}

// SetKubectlContext sets the kubectl context to the specified Kind cluster
func SetKubectlContext(clusterName string) error {
	fullClusterName := GetKibashipClusterName(clusterName)
	contextName := fmt.Sprintf("kind-%s", fullClusterName)
	cmd := exec.Command("kubectl", "config", "use-context", contextName)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set kubectl context to %s: %w", contextName, err)
	}

	return nil
}

// GetClusterInfo returns basic information about the cluster
func GetClusterInfo(clusterName string) (map[string]string, error) {
	info := make(map[string]string)

	fullClusterName := GetKibashipClusterName(clusterName)

	// Get cluster context
	contextName := fmt.Sprintf("kind-%s", fullClusterName)
	info["context"] = contextName

	// Get cluster endpoint
	cmd := exec.Command("kubectl", "cluster-info", "--context", contextName)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster info: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Kubernetes control plane") {
			parts := strings.Split(line, " at ")
			if len(parts) > 1 {
				info["endpoint"] = strings.TrimSpace(parts[1])
			}
		}
	}

	// Get node count
	cmd = exec.Command("kubectl", "get", "nodes", "--no-headers", "--context", contextName)
	output, err = cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get node count: %w", err)
	}

	nodeCount := len(strings.Split(strings.TrimSpace(string(output)), "\n"))
	if strings.TrimSpace(string(output)) == "" {
		nodeCount = 0
	}
	info["nodes"] = fmt.Sprintf("%d", nodeCount)

	return info, nil
}

// ClusterInfo represents information about a Kibaship cluster
type ClusterInfo struct {
	Name      string    // User-friendly name (without prefix)
	FullName  string    // Full cluster name (with prefix)
	Status    string    // Running, Stopped, etc.
	Age       string    // How long the cluster has been running
	Nodes     int       // Number of nodes
	Context   string    // kubectl context name
	Endpoint  string    // API server endpoint
	CreatedAt time.Time // Creation time
}

// GetKibashipClusterInfo returns detailed information about a Kibaship cluster
func GetKibashipClusterInfo(clusterName string) (*ClusterInfo, error) {
	fullClusterName := GetKibashipClusterName(clusterName)
	contextName := fmt.Sprintf("kind-%s", fullClusterName)

	info := &ClusterInfo{
		Name:     GetUserClusterName(fullClusterName),
		FullName: fullClusterName,
		Context:  contextName,
		Status:   "Unknown",
	}

	// Check if cluster exists
	exists, err := ClusterExists(fullClusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to check cluster existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("cluster '%s' not found", clusterName)
	}

	// Get cluster endpoint
	cmd := exec.Command("kubectl", "cluster-info", "--context", contextName)
	if output, err := cmd.Output(); err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "Kubernetes control plane") {
				parts := strings.Split(line, " at ")
				if len(parts) > 1 {
					info.Endpoint = strings.TrimSpace(parts[1])
					info.Status = StatusRunning
				}
			}
		}
	}

	// Get node count
	cmd = exec.Command("kubectl", "get", "nodes", "--no-headers", "--context", contextName)
	if output, err := cmd.Output(); err == nil {
		nodeLines := strings.Split(strings.TrimSpace(string(output)), "\n")
		if strings.TrimSpace(string(output)) != "" {
			info.Nodes = len(nodeLines)
		}
	}

	// Try to get creation time from Docker container
	cmd = exec.Command("docker", "inspect", fullClusterName+"-control-plane", "--format", "{{.Created}}")
	if output, err := cmd.Output(); err == nil {
		if createdAt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(string(output))); err == nil {
			info.CreatedAt = createdAt
			info.Age = formatAge(time.Since(createdAt))
		}
	}

	return info, nil
}

// ListKibashipClusterInfo returns detailed information about all Kibaship clusters
func ListKibashipClusterInfo() ([]*ClusterInfo, error) {
	clusterNames, err := ListKibashipClusters()
	if err != nil {
		return nil, err
	}

	var clusters []*ClusterInfo
	for _, fullName := range clusterNames {
		userFriendlyName := GetUserClusterName(fullName)
		if clusterInfo, err := GetKibashipClusterInfo(userFriendlyName); err == nil {
			clusters = append(clusters, clusterInfo)
		}
	}

	return clusters, nil
}

// formatAge formats a duration into a human-readable age string
func formatAge(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	} else {
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

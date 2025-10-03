package clusters

import (
	"fmt"
	"os"
	"time"
)

// CreateOptions holds the configuration for cluster creation
type CreateOptions struct {
	Name               string
	ControlPlaneNodes  int
	WorkerNodes        int
	CiliumVersion      string
	SkipInfrastructure bool
}

// DefaultCreateOptions returns default options for cluster creation
func DefaultCreateOptions() CreateOptions {
	return CreateOptions{
		ControlPlaneNodes:  1,
		WorkerNodes:        0,
		CiliumVersion:      "1.18.0",
		SkipInfrastructure: false,
	}
}

// CreateCluster creates a complete Kibaship cluster with Phase 1 and 2 components
func CreateCluster(
	opts CreateOptions,
	printStep func(int, string),
	printProgress func(string),
	printSuccess func(string),
	printError func(string),
	printInfo func(string),
) error {

	// Phase 1: Prerequisites & Cluster Creation
	printStep(1, "Checking prerequisites...")
	if err := CheckPrerequisites(); err != nil {
		printError(fmt.Sprintf("Prerequisites check failed: %v", err))
		return err
	}
	printSuccess("All prerequisites are satisfied")

	// Display tool versions
	versions := GetToolVersions()
	for tool, version := range versions {
		printInfo(fmt.Sprintf("%s: %s", tool, version))
	}

	printStep(2, fmt.Sprintf("Generating Kind cluster configuration for '%s'...", opts.Name))
	configPath := GetKindConfigPath(opts.Name)
	config := ClusterConfig{
		Name:              opts.Name,
		ControlPlaneNodes: opts.ControlPlaneNodes,
		WorkerNodes:       opts.WorkerNodes,
		ConfigPath:        configPath,
	}

	if err := GenerateKindConfig(config); err != nil {
		printError(fmt.Sprintf("Failed to generate Kind config: %v", err))
		return err
	}
	printSuccess(fmt.Sprintf("Kind config generated at %s", configPath))
	printInfo(fmt.Sprintf("Control plane nodes: %d", opts.ControlPlaneNodes))
	printInfo(fmt.Sprintf("Worker nodes: %d", opts.WorkerNodes))

	printStep(3, fmt.Sprintf("Creating Kind cluster '%s'...", opts.Name))
	printProgress("This may take a few minutes...")
	if err := CreateKindCluster(config); err != nil {
		printError(fmt.Sprintf("Failed to create Kind cluster: %v", err))
		return err
	}
	printSuccess(fmt.Sprintf("Kind cluster '%s' created successfully", opts.Name))

	printStep(4, "Setting kubectl context...")
	if err := SetKubectlContext(opts.Name); err != nil {
		printError(fmt.Sprintf("Failed to set kubectl context: %v", err))
		return err
	}
	printSuccess(fmt.Sprintf("kubectl context set to kind-%s", opts.Name))

	printStep(5, "Retrieving cluster information...")
	clusterInfo, err := GetClusterInfo(opts.Name)
	if err != nil {
		printError(fmt.Sprintf("Failed to get cluster info: %v", err))
		return err
	}
	printSuccess("Cluster is ready")
	for key, value := range clusterInfo {
		printInfo(fmt.Sprintf("%s: %s", key, value))
	}

	if opts.SkipInfrastructure {
		printSuccess("🎉 Cluster creation complete! (Infrastructure installation skipped)")
		printInfo("To install infrastructure later, run the cluster without --skip-infrastructure")
		return nil
	}

	// Phase 2: Core Infrastructure (CNI & Gateway API)
	printStep(6, "Installing Gateway API CRDs (v1.3.0)...")
	printProgress("Installing 9 custom Gateway API CRDs...")
	if err := InstallGatewayAPI(opts.Name); err != nil {
		printError(fmt.Sprintf("Failed to install Gateway API: %v", err))
		return err
	}
	printSuccess("Gateway API CRDs installed successfully")

	printStep(7, fmt.Sprintf("Installing Cilium CNI (v%s)...", opts.CiliumVersion))
	printProgress("This may take several minutes...")
	if err := InstallCilium(opts.Name, opts.CiliumVersion); err != nil {
		printError(fmt.Sprintf("Failed to install Cilium: %v", err))
		return err
	}
	printSuccess("Cilium CNI installed successfully")

	printStep(8, "Labeling all nodes for ingress...")
	if err := LabelAllNodesForIngress(opts.Name); err != nil {
		printError(fmt.Sprintf("Failed to label nodes for ingress: %v", err))
		return err
	}
	printSuccess("All nodes labeled for ingress")

	// Phase 3: Certificate Management
	printStep(9, fmt.Sprintf("Installing cert-manager (%s)...", CertManagerVersion))
	printProgress("This may take several minutes...")

	certManagerConfig := DefaultCertManagerConfig()
	if err := InstallCertManager(opts.Name, certManagerConfig); err != nil {
		printError(fmt.Sprintf("Failed to install cert-manager: %v", err))
		return err
	}
	printSuccess("cert-manager installed successfully")

	// Phase 4: Build & CI/CD Infrastructure
	printStep(10, fmt.Sprintf("Installing Tekton Pipelines (%s)...", TektonVersion))
	printProgress("Installing 75 Tekton manifests...")
	if err := InstallTekton(opts.Name); err != nil {
		printError(fmt.Sprintf("Failed to install Tekton: %v", err))
		return err
	}
	printSuccess("Tekton Pipelines installed successfully")

	printStep(11, "Installing Valkey Operator (v0.0.59)...")
	printProgress("Installing Valkey operator manifests...")
	if err := InstallValkey(opts.Name); err != nil {
		printError(fmt.Sprintf("Failed to install Valkey: %v", err))
		return err
	}
	printSuccess("Valkey Operator installed successfully")

	// Final verification
	printProgress("Verifying installation...")
	time.Sleep(5 * time.Second) // Give components a moment to settle

	if err := VerifyGatewayAPI(opts.Name); err != nil {
		printError(fmt.Sprintf("Gateway API verification failed: %v", err))
		return err
	}

	if err := VerifyCilium(opts.Name); err != nil {
		printError(fmt.Sprintf("Cilium verification failed: %v", err))
		return err
	}

	if err := VerifyCertManager(opts.Name); err != nil {
		printError(fmt.Sprintf("cert-manager verification failed: %v", err))
		return err
	}

	if err := VerifyTekton(opts.Name); err != nil {
		printError(fmt.Sprintf("Tekton verification failed: %v", err))
		return err
	}

	if err := VerifyValkey(opts.Name); err != nil {
		printError(fmt.Sprintf("Valkey verification failed: %v", err))
		return err
	}

	// Get final status
	ciliumStatus, err := GetCiliumStatus(opts.Name)
	if err == nil {
		printInfo(fmt.Sprintf("Cilium DaemonSet: %s", ciliumStatus["daemonset"]))
		printInfo(fmt.Sprintf("Cilium Operator: %s", ciliumStatus["operator"]))
	}

	certManagerStatus, err := GetCertManagerStatus(opts.Name)
	if err == nil {
		printInfo(fmt.Sprintf("cert-manager Controller: %s", certManagerStatus["cert-manager"]))
		printInfo(fmt.Sprintf("cert-manager Webhook: %s", certManagerStatus["cert-manager-webhook"]))
		printInfo(fmt.Sprintf("cert-manager CA Injector: %s", certManagerStatus["cert-manager-cainjector"]))
	}

	tektonStatus, err := GetTektonStatus(opts.Name)
	if err == nil {
		printInfo(fmt.Sprintf("Tekton Controller: %s", tektonStatus["controller"]))
		printInfo(fmt.Sprintf("Tekton Webhook: %s", tektonStatus["webhook"]))
		printInfo(fmt.Sprintf("Tekton Events: %s", tektonStatus["events"]))
		printInfo(fmt.Sprintf("Tekton Resolvers: %s", tektonStatus["resolvers"]))
	}

	valkeyStatus, err := GetValkeyStatus(opts.Name)
	if err == nil {
		printInfo(fmt.Sprintf("Valkey Operator: %s", valkeyStatus["operator"]))
	}

	printSuccess("🎉 Cluster creation complete!")
	printInfo("Your Kibaship cluster is ready with:")
	printInfo("  ✓ Kind cluster with custom networking")
	printInfo("  ✓ Gateway API CRDs (v1.3.0) - 9 custom CRDs")
	printInfo("  ✓ Cilium CNI with Gateway API support")
	printInfo("  ✓ All nodes labeled for ingress")
	printInfo(fmt.Sprintf("  ✓ cert-manager (%s) with HA configuration", CertManagerVersion))
	printInfo(fmt.Sprintf("  ✓ Tekton Pipelines (%s) for CI/CD", TektonVersion))
	printInfo("  ✓ Valkey Operator (v0.0.59) for Redis-compatible databases")
	printInfo("")
	printInfo("Next steps:")
	fullClusterName := GetKibashipClusterName(opts.Name)
	printInfo(fmt.Sprintf("  kubectl config use-context kind-%s", fullClusterName))
	printInfo("  kubectl get nodes")
	printInfo("  kubectl get pods -A")

	return nil
}

// ValidateCreateOptions validates the cluster creation options
func ValidateCreateOptions(opts CreateOptions) error {
	if err := ValidateClusterName(opts.Name); err != nil {
		return err
	}

	// Check if cluster already exists
	if exists, err := ClusterExistsByName(opts.Name); err != nil {
		return fmt.Errorf("failed to check if cluster exists: %w", err)
	} else if exists {
		return fmt.Errorf("cluster '%s' already exists", opts.Name)
	}

	if opts.ControlPlaneNodes < 1 {
		return fmt.Errorf("must have at least 1 control-plane node")
	}

	if opts.ControlPlaneNodes > 3 {
		return fmt.Errorf("maximum 3 control-plane nodes supported")
	}

	if opts.WorkerNodes < 0 {
		return fmt.Errorf("worker nodes cannot be negative")
	}

	if opts.WorkerNodes > 10 {
		return fmt.Errorf("maximum 10 worker nodes supported")
	}

	if opts.CiliumVersion == "" {
		return fmt.Errorf("cilium version cannot be empty")
	}

	return nil
}

// CleanupOnFailure cleans up resources if cluster creation fails
func CleanupOnFailure(clusterName string, printInfo func(string)) {
	printInfo("Cleaning up failed cluster creation...")

	// Remove Kind cluster if it exists
	if exists, err := ClusterExists(clusterName); err == nil && exists {
		if err := DeleteKindCluster(clusterName); err != nil {
			printInfo(fmt.Sprintf("Failed to cleanup cluster: %v", err))
		} else {
			printInfo("Cluster cleaned up successfully")
		}
	}

	// Remove config file
	configPath := GetKindConfigPath(clusterName)
	if err := os.Remove(configPath); err == nil {
		printInfo("Config file cleaned up")
	}
}

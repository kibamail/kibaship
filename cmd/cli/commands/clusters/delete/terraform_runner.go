package delete

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kibamail/kibaship-operator/cmd/cli/commands/clusters/create"
	"github.com/kibamail/kibaship-operator/cmd/cli/internal/styles"
)

// buildTerraformFiles creates the directory structure and compiles Terraform templates for deletion
func buildTerraformFiles(config *create.CreateConfig) error {
	// Create .kibaship directory structure
	kibashipDir := ".kibaship"
	clusterDir := filepath.Join(kibashipDir, config.Name)
	provisionDir := filepath.Join(clusterDir, "provision")
	bootstrapDir := filepath.Join(clusterDir, "bootstrap")

	// Create all directories
	dirs := []string{kibashipDir, clusterDir, provisionDir, bootstrapDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Use the same template compilation logic as create command
	return create.BuildTerraformFilesForConfig(config)
}

// runTerraformInit runs terraform init in the provision directory with backend configuration
func runTerraformInit(config *create.CreateConfig) error {
	provisionDir := filepath.Join(".kibaship", config.Name, "provision")

	// Prepare backend configuration arguments
	backendArgs := []string{
		"init",
		fmt.Sprintf("-backend-config=bucket=%s", config.TerraformState.S3Bucket),
		fmt.Sprintf("-backend-config=key=clusters/%s/provision.terraform.tfstate", config.Name),
		fmt.Sprintf("-backend-config=region=%s", config.TerraformState.S3Region),
		fmt.Sprintf("-backend-config=access_key=%s", config.TerraformState.S3AccessKey),
		fmt.Sprintf("-backend-config=secret_key=%s", config.TerraformState.S3AccessSecret),
		"-backend-config=encrypt=true",
	}

	// Create terraform command
	cmd := exec.Command("terraform", backendArgs...)
	cmd.Dir = provisionDir

	// Set up environment variables
	cmd.Env = os.Environ()

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start terraform init: %w", err)
	}

	// Stream output in real-time using channels for synchronization
	done := make(chan bool, 2)
	go func() {
		streamOutput(stdout, "")
		done <- true
	}()
	go func() {
		streamOutput(stderr, "")
		done <- true
	}()

	// Wait for command to complete
	err = cmd.Wait()

	// Wait for both output streams to finish
	<-done
	<-done

	if err != nil {
		return fmt.Errorf("terraform init failed: %w", err)
	}

	return nil
}

// runTerraformDestroy runs terraform destroy in the provision directory
func runTerraformDestroy(config *create.CreateConfig) error {
	provisionDir := filepath.Join(".kibaship", config.Name, "provision")

	// Set up TF_VAR environment variables for Terraform
	env := os.Environ()
	env = append(env, fmt.Sprintf("TF_VAR_cluster_name=%s", config.Name))
	env = append(env, fmt.Sprintf("TF_VAR_cluster_email=%s", config.Email))
	env = append(env, fmt.Sprintf("TF_VAR_paas_features=%s", config.PaaSFeatures))

	// Add Terraform state configuration variables
	env = append(env, fmt.Sprintf("TF_VAR_terraform_state_bucket=%s", config.TerraformState.S3Bucket))
	env = append(env, fmt.Sprintf("TF_VAR_terraform_state_region=%s", config.TerraformState.S3Region))
	env = append(env, fmt.Sprintf("TF_VAR_terraform_state_access_key=%s", config.TerraformState.S3AccessKey))
	env = append(env, fmt.Sprintf("TF_VAR_terraform_state_secret_key=%s", config.TerraformState.S3AccessSecret))

	// Add provider-specific environment variables
	switch config.Provider {
	case "digital-ocean":
		if config.DigitalOcean != nil {
			env = append(env, fmt.Sprintf("TF_VAR_do_token=%s", config.DigitalOcean.Token))
			env = append(env, fmt.Sprintf("TF_VAR_do_region=%s", config.DigitalOcean.Region))
			env = append(env, fmt.Sprintf("TF_VAR_do_node_count=%s", config.DigitalOcean.Nodes))
			env = append(env, fmt.Sprintf("TF_VAR_do_node_size=%s", config.DigitalOcean.NodesSize))
		}
	}

	// Create terraform destroy command with auto-approve
	cmd := exec.Command("terraform", "destroy", "-auto-approve")
	cmd.Dir = provisionDir
	cmd.Env = env

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start terraform destroy: %w", err)
	}

	// Stream output in real-time using channels for synchronization
	done := make(chan bool, 2)
	go func() {
		streamOutput(stdout, "")
		done <- true
	}()
	go func() {
		streamOutput(stderr, "")
		done <- true
	}()

	// Wait for command to complete
	err = cmd.Wait()

	// Wait for both output streams to finish
	<-done
	<-done

	if err != nil {
		return fmt.Errorf("terraform destroy failed: %w", err)
	}

	return nil
}

// streamOutput streams command output to console with proper formatting using styles
func streamOutput(reader io.Reader, prefix string) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()

		// Add color formatting for different types of output using styles
		if strings.Contains(line, "Initializing") {
			fmt.Printf("%s %s\n",
				styles.CommandStyle.Render("🔄"),
				styles.DescriptionStyle.Render(prefix+line))
		} else if strings.Contains(line, "Successfully") || strings.Contains(line, "Success!") {
			fmt.Printf("%s %s\n",
				styles.TitleStyle.Render("✅"),
				styles.TitleStyle.Render(prefix+line))
		} else if strings.Contains(line, "Destroying...") || strings.Contains(line, "Destruction complete") {
			fmt.Printf("%s %s\n",
				styles.CommandStyle.Render("🗑️"),
				styles.DescriptionStyle.Render(prefix+line))
		} else if strings.Contains(line, "Refreshing") || strings.Contains(line, "Reading") {
			fmt.Printf("%s %s\n",
				styles.CommandStyle.Render("🔄"),
				styles.DescriptionStyle.Render(prefix+line))
		} else if strings.Contains(line, "Plan:") || strings.Contains(line, "Destroy complete!") {
			fmt.Printf("%s %s\n",
				styles.TitleStyle.Render("📊"),
				styles.TitleStyle.Render(prefix+line))
		} else if strings.Contains(line, "Error") || strings.Contains(line, "error") {
			fmt.Printf("%s %s\n",
				styles.CommandStyle.Render("❌"),
				styles.CommandStyle.Render(prefix+line))
		} else if strings.Contains(line, "Warning") || strings.Contains(line, "warning") {
			fmt.Printf("%s %s\n",
				styles.CommandStyle.Render("⚠️"),
				styles.DescriptionStyle.Render(prefix+line))
		} else if strings.Contains(line, "Downloading") || strings.Contains(line, "Installing") {
			fmt.Printf("%s %s\n",
				styles.CommandStyle.Render("📥"),
				styles.DescriptionStyle.Render(prefix+line))
		} else if strings.HasPrefix(line, "Terraform") {
			fmt.Printf("%s %s\n",
				styles.TitleStyle.Render("🏗️"),
				styles.TitleStyle.Render(prefix+line))
		} else if line != "" {
			fmt.Printf("   %s\n", styles.DescriptionStyle.Render(prefix+line))
		}
	}
}

// checkTerraformInstalled checks if terraform is available in PATH
func checkTerraformInstalled() error {
	cmd := exec.Command("terraform", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("terraform is not installed or not available in PATH. Please install Terraform: https://terraform.io/downloads")
	}
	return nil
}

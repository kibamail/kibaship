package create

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"text/template"
)

//go:embed terraform/providers/*
var terraformTemplates embed.FS

// ListEmbeddedTemplates lists all embedded template files (for debugging)
func ListEmbeddedTemplates() error {
	fmt.Println("Embedded template files:")
	return fs.WalkDir(terraformTemplates, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			fmt.Printf("  %s\n", path)
		}
		return nil
	})
}

// buildTerraformFiles creates the directory structure and compiles Terraform templates
func buildTerraformFiles(config *CreateConfig) error {
	return BuildTerraformFilesForConfig(config)
}

// BuildTerraformFilesForConfig creates the directory structure and compiles Terraform templates (exported for delete command)
func BuildTerraformFilesForConfig(config *CreateConfig) error {
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

	// Get provider-specific template path
	providerPath := fmt.Sprintf("terraform/providers/%s", config.Provider)

	// Compile provision templates
	if err := compileTemplate(providerPath, "provision.tf.tpl", filepath.Join(provisionDir, "main.tf"), config); err != nil {
		return fmt.Errorf("failed to compile provision template: %w", err)
	}

	if err := compileTemplate(providerPath, "vars.tf.tpl", filepath.Join(provisionDir, "vars.tf"), config); err != nil {
		return fmt.Errorf("failed to compile provision vars template: %w", err)
	}

	// Compile bootstrap templates
	if err := compileTemplate(providerPath, "bootstrap.tf.tpl", filepath.Join(bootstrapDir, "main.tf"), config); err != nil {
		return fmt.Errorf("failed to compile bootstrap template: %w", err)
	}

	if err := compileTemplate(providerPath, "vars.tf.tpl", filepath.Join(bootstrapDir, "vars.tf"), config); err != nil {
		return fmt.Errorf("failed to compile bootstrap vars template: %w", err)
	}

	return nil
}

// compileTemplate loads a template from embedded filesystem and compiles it to a file
func compileTemplate(providerPath, templateName, outputPath string, config *CreateConfig) error {
	// Read template from embedded filesystem
	templatePath := filepath.Join(providerPath, templateName)
	templateContent, err := terraformTemplates.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template %s: %w", templatePath, err)
	}

	// Parse template
	tmpl, err := template.New(templateName).Parse(string(templateContent))
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", templateName, err)
	}

	// Create output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", outputPath, err)
	}
	defer outputFile.Close()

	// Execute template with configuration data
	if err := tmpl.Execute(outputFile, config); err != nil {
		return fmt.Errorf("failed to execute template %s: %w", templateName, err)
	}

	return nil
}

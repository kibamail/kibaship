package clusters

import (
	"fmt"
	"os/exec"
	"strings"
)

// CheckPrerequisites verifies that all required tools are installed
func CheckPrerequisites() error {
	tools := []struct {
		name    string
		command []string
		check   func(output string) bool
	}{
		{
			name:    "Kind",
			command: []string{"kind", "version"},
			check: func(output string) bool {
				return strings.Contains(output, "kind")
			},
		},
		{
			name:    "Helm",
			command: []string{"helm", "version", "--short"},
			check: func(output string) bool {
				return strings.Contains(output, "v3.")
			},
		},
		{
			name:    "kubectl",
			command: []string{"kubectl", "version", "--client", "--output=yaml"},
			check: func(output string) bool {
				return strings.Contains(output, "clientVersion")
			},
		},
		{
			name:    "Docker",
			command: []string{"docker", "version", "--format", "{{.Client.Version}}"},
			check: func(output string) bool {
				return len(strings.TrimSpace(output)) > 0
			},
		},
	}

	for _, tool := range tools {
		if err := checkTool(tool.name, tool.command, tool.check); err != nil {
			return err
		}
	}

	return nil
}

func checkTool(name string, command []string, check func(string) bool) error {
	cmd := exec.Command(command[0], command[1:]...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("%s is not installed or not working properly. Please install %s first", name, name)
	}

	if !check(string(output)) {
		return fmt.Errorf("%s installation appears to be invalid. Output: %s", name, string(output))
	}

	return nil
}

// GetToolVersions returns version information for all required tools
func GetToolVersions() map[string]string {
	versions := make(map[string]string)

	tools := []struct {
		name    string
		command []string
		extract func(string) string
	}{
		{
			name:    "kind",
			command: []string{"kind", "version"},
			extract: func(output string) string {
				lines := strings.Split(output, "\n")
				for _, line := range lines {
					if strings.Contains(line, "kind") {
						return strings.TrimSpace(line)
					}
				}
				return StatusUnknown
			},
		},
		{
			name:    "helm",
			command: []string{"helm", "version", "--short"},
			extract: func(output string) string {
				return strings.TrimSpace(output)
			},
		},
		{
			name:    "kubectl",
			command: []string{"kubectl", "version", "--client", "--short"},
			extract: func(output string) string {
				return strings.TrimSpace(output)
			},
		},
		{
			name:    "docker",
			command: []string{"docker", "version", "--format", "{{.Client.Version}}"},
			extract: func(output string) string {
				return "Docker " + strings.TrimSpace(output)
			},
		},
	}

	for _, tool := range tools {
		cmd := exec.Command(tool.command[0], tool.command[1:]...)
		if output, err := cmd.CombinedOutput(); err == nil {
			versions[tool.name] = tool.extract(string(output))
		} else {
			versions[tool.name] = "not found"
		}
	}

	return versions
}

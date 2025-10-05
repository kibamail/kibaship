package clusters

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

const (
	// StatusReady represents a ready condition status
	StatusReady = "Ready"
	// ConditionTrue represents a true condition status
	ConditionTrue = "True"
)

// PodInfo represents information about a Kubernetes pod
type PodInfo struct {
	Name     string
	Ready    string
	Status   string
	Restarts string
	Age      string
	IP       string
	Node     string
}

// NodeInfo represents information about a Kubernetes node
type NodeInfo struct {
	Name    string
	Status  string
	Roles   string
	Age     string
	Version string
}

// MonitoringConfig holds configuration for pod monitoring
type MonitoringConfig struct {
	Namespace     string
	ClusterName   string
	ComponentName string
	Interval      time.Duration
	Timeout       time.Duration
	PrintProgress func(string)
	PrintInfo     func(string)
}

// Spinner represents a simple text-based spinner
type Spinner struct {
	frames []string
	index  int
}

// NewSpinner creates a new spinner with default frames
func NewSpinner() *Spinner {
	return &Spinner{
		frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		index:  0,
	}
}

// Next returns the next frame of the spinner
func (s *Spinner) Next() string {
	frame := s.frames[s.index]
	s.index = (s.index + 1) % len(s.frames)
	return frame
}

// GetPodsInNamespace retrieves pod information for a specific namespace
func GetPodsInNamespace(namespace, contextName string) ([]PodInfo, error) {
	cmd := exec.Command("kubectl", "get", "pods", "-n", namespace, "--context", contextName, "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get pods in namespace %s: %w", namespace, err)
	}

	var podList struct {
		Items []struct {
			Metadata struct {
				Name              string            `json:"name"`
				CreationTimestamp string            `json:"creationTimestamp"`
				Labels            map[string]string `json:"labels"`
			} `json:"metadata"`
			Status struct {
				Phase             string `json:"phase"`
				PodIP             string `json:"podIP"`
				ContainerStatuses []struct {
					Ready        bool  `json:"ready"`
					RestartCount int32 `json:"restartCount"`
				} `json:"containerStatuses"`
			} `json:"status"`
			Spec struct {
				NodeName string `json:"nodeName"`
			} `json:"spec"`
		} `json:"items"`
	}

	if err := json.Unmarshal(output, &podList); err != nil {
		return nil, fmt.Errorf("failed to parse pod JSON: %w", err)
	}

	pods := make([]PodInfo, 0, len(podList.Items))
	for _, item := range podList.Items {
		pod := PodInfo{
			Name:   item.Metadata.Name,
			Status: item.Status.Phase,
			IP:     item.Status.PodIP,
			Node:   item.Spec.NodeName,
		}

		// Calculate age
		if creationTime, err := time.Parse(time.RFC3339, item.Metadata.CreationTimestamp); err == nil {
			age := time.Since(creationTime)
			pod.Age = formatAge(age)
		} else {
			pod.Age = "Unknown"
		}

		// Calculate ready status and restarts
		if len(item.Status.ContainerStatuses) > 0 {
			readyCount := 0
			totalRestarts := int32(0)
			for _, container := range item.Status.ContainerStatuses {
				if container.Ready {
					readyCount++
				}
				totalRestarts += container.RestartCount
			}
			pod.Ready = fmt.Sprintf("%d/%d", readyCount, len(item.Status.ContainerStatuses))
			pod.Restarts = fmt.Sprintf("%d", totalRestarts)
		} else {
			pod.Ready = "0/0"
			pod.Restarts = "0"
		}

		pods = append(pods, pod)
	}

	return pods, nil
}

// GetNodesInCluster retrieves node information for the cluster
func GetNodesInCluster(contextName string) ([]NodeInfo, error) {
	cmd := exec.Command("kubectl", "get", "nodes", "--context", contextName, "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get nodes: %w", err)
	}

	var nodeList struct {
		Items []struct {
			Metadata struct {
				Name              string            `json:"name"`
				CreationTimestamp string            `json:"creationTimestamp"`
				Labels            map[string]string `json:"labels"`
			} `json:"metadata"`
			Status struct {
				Conditions []struct {
					Type   string `json:"type"`
					Status string `json:"status"`
				} `json:"conditions"`
				NodeInfo struct {
					KubeletVersion string `json:"kubeletVersion"`
				} `json:"nodeInfo"`
			} `json:"status"`
		} `json:"items"`
	}

	if err := json.Unmarshal(output, &nodeList); err != nil {
		return nil, fmt.Errorf("failed to parse node JSON: %w", err)
	}

	nodes := make([]NodeInfo, 0, len(nodeList.Items))
	for _, item := range nodeList.Items {
		node := NodeInfo{
			Name:    item.Metadata.Name,
			Version: item.Status.NodeInfo.KubeletVersion,
		}

		// Calculate age
		if creationTime, err := time.Parse(time.RFC3339, item.Metadata.CreationTimestamp); err == nil {
			age := time.Since(creationTime)
			node.Age = formatAge(age)
		} else {
			node.Age = "Unknown"
		}

		// Determine node status
		node.Status = "NotReady"
		for _, condition := range item.Status.Conditions {
			if condition.Type == StatusReady && condition.Status == ConditionTrue {
				node.Status = StatusReady
				break
			}
		}

		// Determine roles
		roles := []string{}
		if _, exists := item.Metadata.Labels["node-role.kubernetes.io/control-plane"]; exists {
			roles = append(roles, "control-plane")
		}
		if _, exists := item.Metadata.Labels["node-role.kubernetes.io/master"]; exists {
			roles = append(roles, "master")
		}
		if len(roles) == 0 {
			roles = append(roles, "worker")
		}
		node.Roles = strings.Join(roles, ",")

		nodes = append(nodes, node)
	}

	return nodes, nil
}

// PrintPodTable prints a beautiful table of pod information
func PrintPodTable(pods []PodInfo, componentName string) {
	PrintPodTableWithSpinner(pods, componentName, "")
}

// PrintPodTableWithSpinner prints a beautiful table of pod information with an optional spinner
func PrintPodTableWithSpinner(pods []PodInfo, componentName, spinnerText string) {
	if len(pods) == 0 {
		fmt.Printf("   No pods found in namespace\n")
		return
	}

	// Define styles
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00D4AA")).
		Bold(true)

	readyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#10B981"))

	notReadyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F59E0B"))

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#EF4444"))

	// Print component header
	fmt.Printf("\n   📦 %s Pods Status:\n", componentName)

	// Print table header
	fmt.Printf("   %s  %s  %s  %s  %s\n",
		headerStyle.Render(padRight("NAME", 35)),
		headerStyle.Render(padRight("READY", 8)),
		headerStyle.Render(padRight("STATUS", 12)),
		headerStyle.Render(padRight("RESTARTS", 10)),
		headerStyle.Render(padRight("AGE", 8)))

	// Print separator
	fmt.Printf("   %s  %s  %s  %s  %s\n",
		strings.Repeat("─", 35),
		strings.Repeat("─", 8),
		strings.Repeat("─", 12),
		strings.Repeat("─", 10),
		strings.Repeat("─", 8))

	// Print pod rows
	for _, pod := range pods {
		// Choose style based on status
		var statusStyle lipgloss.Style
		readyParts := strings.Split(pod.Ready, "/")
		isReady := len(readyParts) == 2 && readyParts[0] == readyParts[1]
		switch {
		case pod.Status == "Running" && isReady:
			statusStyle = readyStyle
		case pod.Status == "Pending" || pod.Status == "ContainerCreating":
			statusStyle = notReadyStyle
		default:
			statusStyle = errorStyle
		}

		fmt.Printf("   %s  %s  %s  %s  %s\n",
			padRight(pod.Name, 35),
			statusStyle.Render(padRight(pod.Ready, 8)),
			statusStyle.Render(padRight(pod.Status, 12)),
			padRight(pod.Restarts, 10),
			padRight(pod.Age, 8))
	}

	// Print spinner if provided
	if spinnerText != "" {
		spinnerStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Italic(true)
		fmt.Printf("   %s %s\n", spinnerText, spinnerStyle.Render("Monitoring..."))
	}

	fmt.Println()
}

// PrintNodeTable prints a beautiful table of node information
func PrintNodeTable(nodes []NodeInfo, componentName string) {
	PrintNodeTableWithSpinner(nodes, componentName, "")
}

// PrintNodeTableWithSpinner prints a beautiful table of node information with an optional spinner
func PrintNodeTableWithSpinner(nodes []NodeInfo, componentName, spinnerText string) {
	if len(nodes) == 0 {
		fmt.Printf("   No nodes found in cluster\n")
		return
	}

	// Define styles
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00D4AA")).
		Bold(true)

	readyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#10B981"))

	notReadyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#EF4444"))

	// Print component header
	fmt.Printf("\n   🖥️  %s Nodes Status:\n", componentName)

	// Print table header
	fmt.Printf("   %s  %s  %s  %s  %s\n",
		headerStyle.Render(padRight("NAME", 35)),
		headerStyle.Render(padRight("STATUS", 12)),
		headerStyle.Render(padRight("ROLES", 15)),
		headerStyle.Render(padRight("AGE", 8)),
		headerStyle.Render(padRight("VERSION", 12)))

	// Print separator
	fmt.Printf("   %s  %s  %s  %s  %s\n",
		strings.Repeat("─", 35),
		strings.Repeat("─", 12),
		strings.Repeat("─", 15),
		strings.Repeat("─", 8),
		strings.Repeat("─", 12))

	// Print node rows
	for _, node := range nodes {
		// Choose style based on status
		var statusStyle lipgloss.Style
		if node.Status == "Ready" {
			statusStyle = readyStyle
		} else {
			statusStyle = notReadyStyle
		}

		fmt.Printf("   %s  %s  %s  %s  %s\n",
			padRight(node.Name, 35),
			statusStyle.Render(padRight(node.Status, 12)),
			padRight(node.Roles, 15),
			padRight(node.Age, 8),
			padRight(node.Version, 12))
	}

	// Print spinner if provided
	if spinnerText != "" {
		spinnerStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Italic(true)
		fmt.Printf("   %s %s\n", spinnerText, spinnerStyle.Render("Monitoring..."))
	}

	fmt.Println()
}

// padRight pads a string to the right with spaces
func padRight(s string, length int) string {
	if len(s) >= length {
		return s[:length]
	}
	return s + strings.Repeat(" ", length-len(s))
}

// MonitorPodsWithTimeout monitors pods in a namespace for a specified duration
func MonitorPodsWithTimeout(config MonitoringConfig) error {
	fullClusterName := GetKibashipClusterName(config.ClusterName)
	contextName := fmt.Sprintf("kind-%s", fullClusterName)

	config.PrintProgress(fmt.Sprintf("Monitoring %s pods (will check every %v for %v)...",
		config.ComponentName, config.Interval, config.Timeout))

	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	ticker := time.NewTicker(config.Interval)
	defer ticker.Stop()

	spinner := NewSpinner()

	// Initial check
	pods, err := GetPodsInNamespace(config.Namespace, contextName)
	if err != nil {
		config.PrintInfo(fmt.Sprintf("Could not get pod status: %v", err))
	} else {
		PrintPodTableWithSpinner(pods, config.ComponentName, spinner.Next())
	}

	for {
		select {
		case <-ctx.Done():
			config.PrintInfo(fmt.Sprintf("Monitoring timeout reached for %s. Proceeding to next step...", config.ComponentName))
			return nil
		case <-ticker.C:
			pods, err := GetPodsInNamespace(config.Namespace, contextName)
			if err != nil {
				config.PrintInfo(fmt.Sprintf("Could not get pod status: %v", err))
				continue
			}
			PrintPodTableWithSpinner(pods, config.ComponentName, spinner.Next())
		}
	}
}

// MonitorComponentInstallation provides a convenient wrapper for monitoring component installations
// with the standard 25-second interval and 2-minute timeout
func MonitorComponentInstallation(clusterName, namespace, componentName string, printProgress, printInfo func(string)) {
	config := MonitoringConfig{
		Namespace:     namespace,
		ClusterName:   clusterName,
		ComponentName: componentName,
		Interval:      25 * time.Second,
		Timeout:       2 * time.Minute,
		PrintProgress: printProgress,
		PrintInfo:     printInfo,
	}

	// Run monitoring in a separate goroutine so it doesn't block
	go func() {
		_ = MonitorPodsWithTimeout(config)
	}()

	// Wait for the timeout duration
	time.Sleep(config.Timeout)
}

// MonitorCiliumInstallation monitors both nodes and pods during Cilium installation
func MonitorCiliumInstallation(clusterName, componentName string, printProgress, printInfo func(string)) {
	fullClusterName := GetKibashipClusterName(clusterName)
	contextName := fmt.Sprintf("kind-%s", fullClusterName)

	timeout := 5 * time.Minute
	checkInterval := 10 * time.Second
	spinnerInterval := 200 * time.Millisecond

	msg := fmt.Sprintf("Monitoring %s nodes and pods (will check every %v for %v)...",
		componentName, checkInterval, timeout)
	printProgress(msg)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	checkTicker := time.NewTicker(checkInterval)
	defer checkTicker.Stop()

	spinnerTicker := time.NewTicker(spinnerInterval)
	defer spinnerTicker.Stop()

	spinner := NewSpinner()

	// Cache the latest status
	var cachedNodes []NodeInfo
	var cachedPods []PodInfo
	var lastError error

	// Function to fetch current status
	fetchStatus := func() {
		// Monitor nodes
		nodes, err := GetNodesInCluster(contextName)
		if err != nil {
			lastError = err
		} else {
			cachedNodes = nodes
			lastError = nil
		}

		// Monitor Cilium pods
		pods, err := GetPodsInNamespace("kube-system", contextName)
		if err != nil {
			lastError = err
		} else {
			// Filter pods to show only Cilium-related ones
			ciliumPods := []PodInfo{}
			for _, pod := range pods {
				if strings.Contains(pod.Name, "cilium") {
					ciliumPods = append(ciliumPods, pod)
				}
			}
			cachedPods = ciliumPods
		}
	}

	// Function to display current status with spinner
	displayStatus := func() {
		if lastError != nil {
			printInfo(fmt.Sprintf("Could not get status: %v", lastError))
			return
		}

		if len(cachedNodes) > 0 {
			PrintNodeTableWithSpinner(cachedNodes, componentName, spinner.Next())
		}

		if len(cachedPods) > 0 {
			PrintPodTableWithSpinner(cachedPods, componentName, spinner.Next())
		}
	}

	// Initial fetch and display
	fetchStatus()
	displayStatus()

	// Monitor in a loop
	for {
		select {
		case <-ctx.Done():
			printInfo(fmt.Sprintf("Monitoring timeout reached for %s. Proceeding to next step...", componentName))
			return
		case <-checkTicker.C:
			// Fetch fresh data from Kubernetes
			fetchStatus()
			displayStatus()
		case <-spinnerTicker.C:
			// Just update the spinner animation with cached data
			displayStatus()
		}
	}
}

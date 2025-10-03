# CLI Styling Guide

This document outlines the available styling functions for the Kibaship CLI to ensure consistent, beautiful output throughout the application.

## Available Styling Functions

### Basic Output Functions

```go
// Success messages (green with checkmark)
printSuccess("Cluster created successfully!")

// Error messages (red with X)
printError("Failed to create cluster")

// Info messages (blue with info icon)
printInfo("Checking cluster status...")

// Warning messages (yellow with warning icon)
printWarning("Cluster creation coming soon!")

// Progress messages (purple with spinner)
printProgress("Installing Kibaship operator...")

// Step-by-step instructions (numbered with primary color)
printStep(1, "Creating Kind cluster")
printStep(2, "Installing operator")
printStep(3, "Configuring kubectl")
```

### Table Output

```go
// Print formatted tables
headers := []string{"Name", "Status", "Age"}
rows := [][]string{
    {"my-cluster", "Running", "2h"},
    {"test-cluster", "Pending", "5m"},
}
printTable(headers, rows)
```

### Banner Functions

```go
// Print the main banner (used in help screens)
printBanner()
```

## Color Palette

The CLI uses a consistent color palette:

- **Primary Color**: `#00D4AA` (Teal) - Used for main elements, commands
- **Secondary Color**: `#7C3AED` (Purple) - Used for accents
- **Accent Color**: `#F59E0B` (Amber) - Used for command names
- **Text Color**: `#E5E7EB` (Light Gray) - Used for regular text
- **Muted Color**: `#9CA3AF` (Gray) - Used for descriptions
- **Success Color**: `#10B981` (Green) - Used for success messages
- **Error Color**: `#EF4444` (Red) - Used for error messages
- **Info Color**: `#3B82F6` (Blue) - Used for info messages
- **Warning Color**: `#F59E0B` (Amber) - Used for warnings
- **Progress Color**: `#8B5CF6` (Purple) - Used for progress indicators

## Usage Examples

### Cluster Creation Implementation

```go
func createCluster(name string) {
    printStep(1, "Checking if Kind is installed...")
    
    if !isKindInstalled() {
        printError("Kind is not installed. Please install Kind first.")
        return
    }
    
    printSuccess("Kind found!")
    
    printStep(2, fmt.Sprintf("Creating cluster '%s'...", name))
    printProgress("This may take a few minutes...")
    
    if err := createKindCluster(name); err != nil {
        printError(fmt.Sprintf("Failed to create cluster: %v", err))
        return
    }
    
    printSuccess(fmt.Sprintf("Cluster '%s' created successfully!", name))
    
    printStep(3, "Installing Kibaship operator...")
    // ... implementation
    
    printSuccess("Setup complete! 🎉")
    printInfo(fmt.Sprintf("Use 'kubectl config use-context kind-%s' to switch to this cluster", name))
}
```

### List Implementation

```go
func listClusters() {
    printInfo("Fetching cluster information...")
    
    clusters := getClusters() // Your implementation
    
    if len(clusters) == 0 {
        printWarning("No clusters found")
        printInfo("Use 'kibaship clusters create <name>' to create a new cluster")
        return
    }
    
    headers := []string{"Name", "Status", "Age", "Nodes"}
    var rows [][]string
    
    for _, cluster := range clusters {
        rows = append(rows, []string{
            cluster.Name,
            cluster.Status,
            cluster.Age,
            fmt.Sprintf("%d", cluster.NodeCount),
        })
    }
    
    printTable(headers, rows)
    printInfo(fmt.Sprintf("Found %d cluster(s)", len(clusters)))
}
```

## Best Practices

1. **Consistent Icons**: Use the predefined functions with their icons for consistency
2. **Step-by-Step Operations**: Use `printStep()` for multi-step processes
3. **Progress Feedback**: Use `printProgress()` for long-running operations
4. **Clear Success/Error States**: Always provide clear feedback on operation results
5. **Helpful Information**: Use `printInfo()` to provide next steps or additional context
6. **Tables for Lists**: Use `printTable()` for structured data display

## Custom Styling

If you need custom styling beyond the provided functions, use the existing color palette:

```go
customStyle := lipgloss.NewStyle().
    Foreground(primaryColor).
    Bold(true).
    Padding(1, 2)

fmt.Println(customStyle.Render("Custom styled text"))
```

This ensures consistency with the overall CLI design while allowing flexibility for specific use cases.

package hetznerrobot

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net"
)

// NetworkRanges represents the generated network IP ranges for a cluster
type NetworkRanges struct {
	ClusterNetworkIPRange       string `json:"cluster_network_ip_range"`
	ClusterVSwitchSubnetIPRange string `json:"cluster_vswitch_subnet_ip_range"`
	ClusterSubnetIPRange        string `json:"cluster_subnet_ip_range"`
}

// GenerateClusterNetworkRanges generates non-conflicting IP ranges for cluster networking
// Uses less common private IP ranges to reduce conflicts with existing infrastructure
func GenerateClusterNetworkRanges() (*NetworkRanges, error) {
	// Use less common private IP ranges to avoid conflicts
	// RFC 1918 defines:
	// - 10.0.0.0/8 (10.0.0.0 to 10.255.255.255) - commonly used
	// - 172.16.0.0/12 (172.16.0.0 to 172.31.255.255) - less common
	// - 192.168.0.0/16 (192.168.0.0 to 192.168.255.255) - commonly used

	// We'll use the 172.16.0.0/12 range for better conflict avoidance
	// Generate a random /16 within the 172.16.0.0/12 range

	// Generate random second octet between 16-31 (172.16.0.0/12 range)
	secondOctet, err := generateRandomInt(16, 31)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random second octet: %w", err)
	}

	// Generate random third octet for additional uniqueness
	thirdOctet, err := generateRandomInt(0, 255)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random third octet: %w", err)
	}

	// Main cluster network: /16 for maximum flexibility (65,534 IPs)
	clusterNetworkIPRange := fmt.Sprintf("172.%d.0.0/16", secondOctet)

	// VSwitch subnet: /20 within the cluster network (4,094 IPs)
	// IMPORTANT: Never use .0.0 as it conflicts with the main network gateway
	// Start at .16.0 minimum to avoid gateway conflicts
	vswitchThirdOctet := (thirdOctet / 16) * 16 // Align to /20 boundary
	if vswitchThirdOctet == 0 {
		vswitchThirdOctet = 16 // Skip .0.0 to avoid gateway conflict with main network
	}
	clusterVSwitchSubnetIPRange := fmt.Sprintf("172.%d.%d.0/20", secondOctet, vswitchThirdOctet)

	// Load balancer subnet: /20 within the cluster network, non-overlapping with vSwitch
	// Use next /20 block after vSwitch subnet
	lbThirdOctet := vswitchThirdOctet + 16
	if lbThirdOctet > 240 {
		// If we're near the end, wrap around but skip .0.0
		lbThirdOctet = 16 // Safe fallback
		if vswitchThirdOctet == 16 {
			lbThirdOctet = 32 // If vSwitch is at 16, use 32
		}
	}
	clusterSubnetIPRange := fmt.Sprintf("172.%d.%d.0/20", secondOctet, lbThirdOctet)

	// Validate that the ranges don't overlap
	if err := validateNonOverlapping(clusterVSwitchSubnetIPRange, clusterSubnetIPRange); err != nil {
		return nil, fmt.Errorf("generated overlapping subnets: %w", err)
	}

	return &NetworkRanges{
		ClusterNetworkIPRange:       clusterNetworkIPRange,
		ClusterVSwitchSubnetIPRange: clusterVSwitchSubnetIPRange,
		ClusterSubnetIPRange:        clusterSubnetIPRange,
	}, nil
}

// generateRandomInt generates a cryptographically secure random integer between min and max (inclusive)
func generateRandomInt(min, max int) (int, error) {
	if min > max {
		return 0, fmt.Errorf("min (%d) cannot be greater than max (%d)", min, max)
	}

	// Calculate the range
	rangeSize := max - min + 1

	// Generate random number
	randomBig, err := rand.Int(rand.Reader, big.NewInt(int64(rangeSize)))
	if err != nil {
		return 0, fmt.Errorf("failed to generate random number: %w", err)
	}

	return int(randomBig.Int64()) + min, nil
}

// validateNonOverlapping ensures two CIDR ranges don't overlap
func validateNonOverlapping(cidr1, cidr2 string) error {
	_, net1, err := net.ParseCIDR(cidr1)
	if err != nil {
		return fmt.Errorf("invalid CIDR %s: %w", cidr1, err)
	}

	_, net2, err := net.ParseCIDR(cidr2)
	if err != nil {
		return fmt.Errorf("invalid CIDR %s: %w", cidr2, err)
	}

	// Check if net1 contains net2's network address
	if net1.Contains(net2.IP) {
		return fmt.Errorf("CIDR %s overlaps with %s", cidr1, cidr2)
	}

	// Check if net2 contains net1's network address
	if net2.Contains(net1.IP) {
		return fmt.Errorf("CIDR %s overlaps with %s", cidr2, cidr1)
	}

	return nil
}

// GetNetworkInfo returns detailed information about the generated network ranges
func (nr *NetworkRanges) GetNetworkInfo() map[string]interface{} {
	return map[string]interface{}{
		"cluster_network": map[string]interface{}{
			"cidr":        nr.ClusterNetworkIPRange,
			"description": "Main cluster network (all cluster traffic)",
			"size":        "65,534 IPs (/16)",
		},
		"vswitch_subnet": map[string]interface{}{
			"cidr":        nr.ClusterVSwitchSubnetIPRange,
			"description": "VSwitch subnet (Hetzner Robot servers)",
			"size":        "4,094 IPs (/20)",
		},
		"loadbalancer_subnet": map[string]interface{}{
			"cidr":        nr.ClusterSubnetIPRange,
			"description": "Load balancer subnet (Hetzner Cloud resources)",
			"size":        "4,094 IPs (/20)",
		},
	}
}

// ValidateNetworkRanges validates that the provided network ranges are valid and non-overlapping
func ValidateNetworkRanges(ranges *NetworkRanges) error {
	// Validate each CIDR
	cidrs := []string{
		ranges.ClusterNetworkIPRange,
		ranges.ClusterVSwitchSubnetIPRange,
		ranges.ClusterSubnetIPRange,
	}

	for _, cidr := range cidrs {
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			return fmt.Errorf("invalid CIDR %s: %w", cidr, err)
		}
	}

	// Validate that subnets are within the main cluster network
	_, clusterNet, _ := net.ParseCIDR(ranges.ClusterNetworkIPRange)
	_, vswitchNet, _ := net.ParseCIDR(ranges.ClusterVSwitchSubnetIPRange)
	_, lbNet, _ := net.ParseCIDR(ranges.ClusterSubnetIPRange)

	if !clusterNet.Contains(vswitchNet.IP) {
		return fmt.Errorf("vSwitch subnet %s is not within cluster network %s",
			ranges.ClusterVSwitchSubnetIPRange, ranges.ClusterNetworkIPRange)
	}

	if !clusterNet.Contains(lbNet.IP) {
		return fmt.Errorf("load balancer subnet %s is not within cluster network %s",
			ranges.ClusterSubnetIPRange, ranges.ClusterNetworkIPRange)
	}

	// Validate that subnets don't overlap with each other
	return validateNonOverlapping(ranges.ClusterVSwitchSubnetIPRange, ranges.ClusterSubnetIPRange)
}

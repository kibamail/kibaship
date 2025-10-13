package hetznerrobot

import (
	"net"
	"strings"
	"testing"
)

func TestGenerateClusterNetworkRanges_NoGatewayConflict(t *testing.T) {
	// Run multiple times to test randomness
	for i := 0; i < 100; i++ {
		ranges, err := GenerateClusterNetworkRanges()
		if err != nil {
			t.Fatalf("Failed to generate network ranges: %v", err)
		}

		// Parse CIDRs
		_, mainNet, err := net.ParseCIDR(ranges.ClusterNetworkIPRange)
		if err != nil {
			t.Fatalf("Invalid main network CIDR %s: %v", ranges.ClusterNetworkIPRange, err)
		}

		_, vswitchNet, err := net.ParseCIDR(ranges.ClusterVSwitchSubnetIPRange)
		if err != nil {
			t.Fatalf("Invalid vSwitch subnet CIDR %s: %v", ranges.ClusterVSwitchSubnetIPRange, err)
		}

		_, lbNet, err := net.ParseCIDR(ranges.ClusterSubnetIPRange)
		if err != nil {
			t.Fatalf("Invalid LB subnet CIDR %s: %v", ranges.ClusterSubnetIPRange, err)
		}

		// Calculate gateways (first IP + 1 in each subnet)
		mainGateway := calculateGateway(mainNet)
		vswitchGateway := calculateGateway(vswitchNet)
		lbGateway := calculateGateway(lbNet)

		// Check: vSwitch gateway must be different from main gateway
		if mainGateway == vswitchGateway {
			t.Errorf("Gateway conflict detected!\n"+
				"  Main network: %s → Gateway: %s\n"+
				"  vSwitch subnet: %s → Gateway: %s\n"+
				"  This will cause Hetzner Cloud error!",
				ranges.ClusterNetworkIPRange, mainGateway,
				ranges.ClusterVSwitchSubnetIPRange, vswitchGateway)
		}

		// Check: LB gateway must be different from main gateway
		if mainGateway == lbGateway {
			t.Errorf("Gateway conflict detected!\n"+
				"  Main network: %s → Gateway: %s\n"+
				"  LB subnet: %s → Gateway: %s",
				ranges.ClusterNetworkIPRange, mainGateway,
				ranges.ClusterSubnetIPRange, lbGateway)
		}

		// Check: vSwitch subnet must not start at .0.0
		if strings.HasSuffix(ranges.ClusterVSwitchSubnetIPRange, ".0.0/20") {
			t.Errorf("vSwitch subnet starts at .0.0 which conflicts with main network gateway: %s",
				ranges.ClusterVSwitchSubnetIPRange)
		}

		// Verify subnets don't overlap
		if vswitchNet.Contains(lbNet.IP) || lbNet.Contains(vswitchNet.IP) {
			t.Errorf("Subnet overlap detected!\n"+
				"  vSwitch: %s\n"+
				"  LB: %s",
				ranges.ClusterVSwitchSubnetIPRange,
				ranges.ClusterSubnetIPRange)
		}
	}
}

func TestGenerateClusterNetworkRanges_ValidateStructure(t *testing.T) {
	ranges, err := GenerateClusterNetworkRanges()
	if err != nil {
		t.Fatalf("Failed to generate network ranges: %v", err)
	}

	// Check format: 172.X.0.0/16
	if !strings.HasPrefix(ranges.ClusterNetworkIPRange, "172.") ||
		!strings.HasSuffix(ranges.ClusterNetworkIPRange, ".0.0/16") {
		t.Errorf("Invalid main network format: %s (expected 172.X.0.0/16)", ranges.ClusterNetworkIPRange)
	}

	// Check format: 172.X.Y.0/20 where Y >= 16
	if !strings.HasPrefix(ranges.ClusterVSwitchSubnetIPRange, "172.") ||
		!strings.HasSuffix(ranges.ClusterVSwitchSubnetIPRange, ".0/20") {
		t.Errorf("Invalid vSwitch subnet format: %s (expected 172.X.Y.0/20)", ranges.ClusterVSwitchSubnetIPRange)
	}

	// Check format: 172.X.Y.0/20
	if !strings.HasPrefix(ranges.ClusterSubnetIPRange, "172.") ||
		!strings.HasSuffix(ranges.ClusterSubnetIPRange, ".0/20") {
		t.Errorf("Invalid LB subnet format: %s (expected 172.X.Y.0/20)", ranges.ClusterSubnetIPRange)
	}
}

// calculateGateway calculates the gateway IP for a network (first IP + 1)
func calculateGateway(ipNet *net.IPNet) string {
	gateway := make(net.IP, len(ipNet.IP))
	copy(gateway, ipNet.IP)

	// Add 1 to the last octet
	if len(gateway) == 4 || gateway.To4() != nil {
		ipv4 := gateway.To4()
		ipv4[3] += 1
		return ipv4.String()
	}

	return ""
}

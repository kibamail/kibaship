package hetznerrobot

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/cosi-project/runtime/pkg/resource"
	"github.com/siderolabs/talos/pkg/machinery/client"
	"github.com/siderolabs/talos/pkg/machinery/resources/network"
)

// TalosAddress represents a network address from Talos
type TalosAddress struct {
	LinkName string
	Address  string // CIDR format (e.g., "192.168.1.1/24")
	Family   string // "inet4" or "inet6"
	Scope    string // "host", "link", or "global"
}

// TalosLink represents a network interface from Talos
type TalosLink struct {
	Name   string
	HWAddr string
	State  string
	MTU    uint32
}

// TalosRoute represents a network route from Talos
type TalosRoute struct {
	Destination  string // CIDR format (e.g., "0.0.0.0/0" for default route)
	Gateway      string
	OutLinkName  string
	Family       string
	Scope        string
	Protocol     string
}

// NetworkInterface represents a detected network interface with its IPs
type NetworkInterface struct {
	LinkName    string
	HasPrivateIP bool
	HasPublicIP  bool
	PrivateIPs  []string // IPs without CIDR
	PublicIPs   []string // IPs without CIDR
	PrivateCIDRs []string // Full CIDR addresses
	PublicCIDRs  []string // Full CIDR addresses
}

// NetworkInterfaceDetectionResult contains the result of interface detection
type NetworkInterfaceDetectionResult struct {
	PrivateInterface string
	PublicInterface  string
	AllInterfaces    map[string]*NetworkInterface
}

// GatewayInfo contains gateway information for public and private networks
type GatewayInfo struct {
	PublicGateway  string
	PrivateGateway string
}

// DiscoveredNetworkInfo contains all discovered network information for a server
type DiscoveredNetworkInfo struct {
	Addresses  []TalosAddress
	Links      []TalosLink
	Routes     []TalosRoute
	Interfaces *NetworkInterfaceDetectionResult
	Gateways   *GatewayInfo
}

// NetworkDiscoveryService handles fetching network information from Talos
type NetworkDiscoveryService struct {
	client *client.Client
}

// NewNetworkDiscoveryService creates a new network discovery service
func NewNetworkDiscoveryService(c *client.Client) *NetworkDiscoveryService {
	return &NetworkDiscoveryService{client: c}
}

// DiscoverAll fetches all network information from Talos
func (s *NetworkDiscoveryService) DiscoverAll(ctx context.Context) (*DiscoveredNetworkInfo, error) {
	// Fetch addresses
	addresses, err := s.fetchAddresses(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch addresses: %w", err)
	}

	// Fetch links
	links, err := s.fetchLinks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch links: %w", err)
	}

	// Fetch routes
	routes, err := s.fetchRoutes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch routes: %w", err)
	}

	// Detect interfaces
	interfaces := DetectNetworkInterfaces(addresses)

	// Detect gateways
	gateways := DetectGateways(routes, addresses)

	return &DiscoveredNetworkInfo{
		Addresses:  addresses,
		Links:      links,
		Routes:     routes,
		Interfaces: interfaces,
		Gateways:   gateways,
	}, nil
}

// fetchAddresses fetches network addresses from Talos
func (s *NetworkDiscoveryService) fetchAddresses(ctx context.Context) ([]TalosAddress, error) {
	items, err := s.client.COSI.List(ctx, resource.NewMetadata(network.NamespaceName, network.AddressStatusType, "", resource.VersionUndefined))
	if err != nil {
		return nil, fmt.Errorf("failed to list address resources: %w", err)
	}

	var addresses []TalosAddress
	for _, item := range items.Items {
		spec, ok := item.Spec().(*network.AddressStatusSpec)
		if !ok {
			continue
		}

		addresses = append(addresses, TalosAddress{
			LinkName: spec.LinkName,
			Address:  spec.Address.String(),
			Family:   string(spec.Family),
			Scope:    string(spec.Scope),
		})
	}

	return addresses, nil
}

// fetchLinks fetches network links (interfaces) from Talos
func (s *NetworkDiscoveryService) fetchLinks(ctx context.Context) ([]TalosLink, error) {
	items, err := s.client.COSI.List(ctx, resource.NewMetadata(network.NamespaceName, network.LinkStatusType, "", resource.VersionUndefined))
	if err != nil {
		return nil, fmt.Errorf("failed to list link resources: %w", err)
	}

	var links []TalosLink
	for _, item := range items.Items {
		spec, ok := item.Spec().(*network.LinkStatusSpec)
		if !ok {
			continue
		}

		links = append(links, TalosLink{
			Name:   item.Metadata().ID(),
			HWAddr: spec.HardwareAddr.String(),
			State:  spec.OperationalState.String(),
			MTU:    spec.MTU,
		})
	}

	return links, nil
}

// fetchRoutes fetches network routes from Talos
func (s *NetworkDiscoveryService) fetchRoutes(ctx context.Context) ([]TalosRoute, error) {
	items, err := s.client.COSI.List(ctx, resource.NewMetadata(network.NamespaceName, network.RouteStatusType, "", resource.VersionUndefined))
	if err != nil {
		return nil, fmt.Errorf("failed to list route resources: %w", err)
	}

	var routes []TalosRoute
	for _, item := range items.Items {
		spec, ok := item.Spec().(*network.RouteStatusSpec)
		if !ok {
			continue
		}

		routes = append(routes, TalosRoute{
			Destination: spec.Destination.String(),
			Gateway:     spec.Gateway.String(),
			OutLinkName: spec.OutLinkName,
			Family:      string(spec.Family),
			Scope:       string(spec.Scope),
			Protocol:    string(spec.Protocol),
		})
	}

	return routes, nil
}

// PublicInterfaceInfo contains detected public interface information
// This matches the TypeScript TalosNetworkDetection.publicInterface structure
type PublicInterfaceInfo struct {
	Name          string // Interface name (e.g., "enp5s0")
	Gateway       string // Gateway IP (e.g., "65.109.58.65")
	IPAddress     string // IP without CIDR (e.g., "65.109.58.113")
	AddressSubnet string // IP with CIDR (e.g., "65.109.58.113/26")
}

// FindPublicInterface detects the public interface using the default route
// This exactly matches the TypeScript findPublicInterface() method
func FindPublicInterface(links []TalosLink, addresses []TalosAddress, routes []TalosRoute, nodeIP string) *PublicInterfaceInfo {
	// Step 1: Find the default route (empty dst = 0.0.0.0/0)
	// Matches TypeScript: route.spec.dst === '' && route.spec.gateway && route.spec.table === 'main'
	var defaultRoute *TalosRoute
	for i := range routes {
		route := &routes[i]
		if route.Family == "inet4" &&
			route.Destination == "0.0.0.0/0" &&
			route.Gateway != "" &&
			route.Gateway != "<nil>" {
			defaultRoute = route
			break
		}
	}

	if defaultRoute == nil {
		return nil
	}

	// Step 2: Extract interface name and gateway from default route
	publicInterfaceName := defaultRoute.OutLinkName
	gateway := defaultRoute.Gateway

	// Step 3: Find the IP address on this interface that matches the node IP
	// Matches TypeScript: addr.spec.linkName === publicInterfaceName && addr.spec.address.startsWith(nodeIp)
	var publicAddress *TalosAddress
	for i := range addresses {
		addr := &addresses[i]
		if addr.LinkName == publicInterfaceName &&
			addr.Family == "inet4" &&
			strings.HasPrefix(addr.Address, nodeIP) {
			publicAddress = addr
			break
		}
	}

	if publicAddress == nil {
		return nil
	}

	// Step 4: Extract IP without CIDR
	ipAddress := strings.Split(publicAddress.Address, "/")[0]

	// Step 5: Return all information together
	return &PublicInterfaceInfo{
		Name:          publicInterfaceName,
		Gateway:       gateway,
		IPAddress:     ipAddress,
		AddressSubnet: publicAddress.Address, // Full address with CIDR
	}
}

// DetectNetworkInterfaces analyzes addresses and groups them by interface
// This is kept for compatibility but is now secondary to FindPublicInterface
func DetectNetworkInterfaces(addresses []TalosAddress) *NetworkInterfaceDetectionResult {
	interfaces := make(map[string]*NetworkInterface)

	// Parse all addresses and group by interface
	for _, addr := range addresses {
		// Skip loopback and external virtual interfaces
		if addr.LinkName == "lo" || addr.LinkName == "external" {
			continue
		}

		// Skip IPv6 addresses for simplicity
		if addr.Family == "inet6" {
			continue
		}

		// Extract IP without CIDR
		ip := strings.Split(addr.Address, "/")[0]
		isPrivate := IsPrivateIP(ip)

		// Initialize interface if not exists
		if _, exists := interfaces[addr.LinkName]; !exists {
			interfaces[addr.LinkName] = &NetworkInterface{
				LinkName:     addr.LinkName,
				HasPrivateIP: false,
				HasPublicIP:  false,
				PrivateIPs:   []string{},
				PublicIPs:    []string{},
				PrivateCIDRs: []string{},
				PublicCIDRs:  []string{},
			}
		}

		iface := interfaces[addr.LinkName]

		if isPrivate {
			iface.HasPrivateIP = true
			iface.PrivateIPs = append(iface.PrivateIPs, ip)
			iface.PrivateCIDRs = append(iface.PrivateCIDRs, addr.Address)
		} else {
			iface.HasPublicIP = true
			iface.PublicIPs = append(iface.PublicIPs, ip)
			iface.PublicCIDRs = append(iface.PublicCIDRs, addr.Address)
		}
	}

	// Priority logic for private interface selection:
	// 1. Interface with private IP and no public IP (dedicated private network)
	// 2. Interface with private IPs (may also have public)
	// 3. Fallback to first interface with private IP
	var privateInterface string
	var publicInterface string

	// Find private interface (interface with only private IPs)
	for _, iface := range interfaces {
		if iface.HasPrivateIP && !iface.HasPublicIP {
			privateInterface = iface.LinkName
			break
		}
	}

	// If no dedicated private interface, find any with private IP
	if privateInterface == "" {
		for _, iface := range interfaces {
			if iface.HasPrivateIP {
				privateInterface = iface.LinkName
				break
			}
		}
	}

	// Find public interface
	for _, iface := range interfaces {
		if iface.HasPublicIP {
			publicInterface = iface.LinkName
			break
		}
	}

	return &NetworkInterfaceDetectionResult{
		PrivateInterface: privateInterface,
		PublicInterface:  publicInterface,
		AllInterfaces:    interfaces,
	}
}

// DetectGateways extracts gateway information from routes
// This is kept for compatibility but FindPublicInterface is now the primary method
func DetectGateways(routes []TalosRoute, addresses []TalosAddress) *GatewayInfo {
	var publicGateway, privateGateway string

	for _, route := range routes {
		// Skip routes without gateways
		if route.Gateway == "" || route.Gateway == "<nil>" {
			continue
		}

		// Skip IPv6 routes
		if route.Family == "inet6" {
			continue
		}

		// Default route (0.0.0.0/0) is typically the public gateway
		if route.Destination == "0.0.0.0/0" && publicGateway == "" {
			publicGateway = route.Gateway
			continue
		}

		// Routes with private destinations indicate private gateways
		if IsPrivateIP(route.Gateway) && privateGateway == "" {
			privateGateway = route.Gateway
		}
	}

	return &GatewayInfo{
		PublicGateway:  publicGateway,
		PrivateGateway: privateGateway,
	}
}

// IsPrivateIP checks if an IP address is in private range (RFC 1918)
func IsPrivateIP(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// Define private IP ranges
	privateRanges := []string{
		"10.0.0.0/8",       // 10.0.0.0 to 10.255.255.255
		"172.16.0.0/12",    // 172.16.0.0 to 172.31.255.255
		"192.168.0.0/16",   // 192.168.0.0 to 192.168.255.255
	}

	for _, cidr := range privateRanges {
		_, ipNet, _ := net.ParseCIDR(cidr)
		if ipNet.Contains(parsedIP) {
			return true
		}
	}

	return false
}

// GetPrimaryPrivateIP gets the primary private IP address for a given interface
func GetPrimaryPrivateIP(addresses []TalosAddress, interfaceName string) string {
	for _, addr := range addresses {
		if addr.LinkName == interfaceName && addr.Family == "inet4" {
			ip := strings.Split(addr.Address, "/")[0]
			if IsPrivateIP(ip) {
				return ip
			}
		}
	}
	return ""
}

// GetPrimaryPrivateCIDR gets the primary private IP with CIDR for a given interface
func GetPrimaryPrivateCIDR(addresses []TalosAddress, interfaceName string) string {
	for _, addr := range addresses {
		if addr.LinkName == interfaceName && addr.Family == "inet4" {
			ip := strings.Split(addr.Address, "/")[0]
			if IsPrivateIP(ip) {
				return addr.Address
			}
		}
	}
	return ""
}

// GetPrimaryPublicIP gets the primary public IP address for a given interface
func GetPrimaryPublicIP(addresses []TalosAddress, interfaceName string) string {
	for _, addr := range addresses {
		if addr.LinkName == interfaceName && addr.Family == "inet4" {
			ip := strings.Split(addr.Address, "/")[0]
			if !IsPrivateIP(ip) {
				return ip
			}
		}
	}
	return ""
}

// GetPrimaryPublicCIDR gets the primary public IP with CIDR for a given interface
func GetPrimaryPublicCIDR(addresses []TalosAddress, interfaceName string) string {
	for _, addr := range addresses {
		if addr.LinkName == interfaceName && addr.Family == "inet4" {
			ip := strings.Split(addr.Address, "/")[0]
			if !IsPrivateIP(ip) {
				return addr.Address
			}
		}
	}
	return ""
}

// CalculateGatewayFromCIDR calculates the gateway IP from a CIDR address
// For Hetzner, the gateway is typically at the .1 or .65 address in the subnet
func CalculateGatewayFromCIDR(cidr string) (string, error) {
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", fmt.Errorf("invalid CIDR: %w", err)
	}

	// Get the network address
	networkIP := ip.Mask(ipNet.Mask)

	// For /26 networks (64 addresses), gateway is typically at .65 (first usable IP + 1)
	// For other networks, try .1 first
	ones, _ := ipNet.Mask.Size()

	if ones == 26 {
		// /26 network - gateway is typically at network + 1 (e.g., .65 for .64/26)
		gateway := make(net.IP, len(networkIP))
		copy(gateway, networkIP)
		gateway[len(gateway)-1] += 1
		return gateway.String(), nil
	}

	// For other networks, gateway is typically at network + 1
	gateway := make(net.IP, len(networkIP))
	copy(gateway, networkIP)
	gateway[len(gateway)-1] += 1
	return gateway.String(), nil
}

// GetLinkByName finds a link by its name
func GetLinkByName(links []TalosLink, name string) *TalosLink {
	for _, link := range links {
		if link.Name == name {
			return &link
		}
	}
	return nil
}

// Helper to check if resource list is empty
func isResourceListEmpty(list *resource.List) bool {
	return list == nil || len(list.Items) == 0
}

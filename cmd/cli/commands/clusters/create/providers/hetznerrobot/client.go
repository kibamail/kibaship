package hetznerrobot

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/floshodan/hrobot-go/hrobot"
)

// Client wraps the hrobot-go client with additional functionality
type Client struct {
	hrobotClient *hrobot.Client
	username     string
	password     string
}

// RescueResponse represents the direct API response for rescue activation
type RescueResponse struct {
	Rescue struct {
		ServerIP     string   `json:"server_ip"`
		ServerNumber int      `json:"server_number"`
		OS           string   `json:"os"`
		Active       bool     `json:"active"`
		Password     string   `json:"password"`
		AuthorizedKey []string `json:"authorized_key"`
	} `json:"rescue"`
}

// Server represents a Hetzner Robot server with formatted information
type Server struct {
	ID         string
	Name       string
	IP         string
	Status     string
	Product    string
	DC         string
	Traffic    string
	Cancelled  bool
	PaidUntil  string
	ServerType string
}

// VSwitch represents a Hetzner Robot vswitch with formatted information
type VSwitch struct {
	ID                   string
	Name                 string
	VLAN                 int
	Cancelled            bool
	HasCloudNetworkAttached bool
}

// VSwitchServer represents a server attached to a vswitch with its status
type VSwitchServer struct {
	ServerIP     string
	ServerNumber int
	Status       string // "ready", "in process", "failed"
}

// VSwitchDetails represents detailed vswitch information including attached servers
type VSwitchDetails struct {
	VSwitch
	AttachedServers []VSwitchServer
}

// ServerRescueStatus represents the rescue mode status of a server
type ServerRescueStatus struct {
	ServerID       string
	ServerName     string
	ServerIP       string
	InRescueMode   bool
	RescuePassword string
	Status         string // "checking", "enabling", "resetting", "pinging", "ready", "failed"
}

// ServerDetails represents detailed server information including rescue availability
type ServerDetails struct {
	ServerNumber int    `json:"server_number"`
	ServerName   string `json:"server_name"`
	ServerIP     string `json:"server_ip"`
	Product      string `json:"product"`
	DC           string `json:"dc"`
	Status       string `json:"status"`
	Rescue       bool   `json:"rescue"`
	Reset        bool   `json:"reset"`
	VNC          bool   `json:"vnc"`
	Windows      bool   `json:"windows"`
}

// NewClient creates a new Hetzner Robot client wrapper
func NewClient(username, password string) *Client {
	hrobotClient := hrobot.NewClient(hrobot.WithBasicAuth(username, password))
	return &Client{
		hrobotClient: hrobotClient,
		username:     username,
		password:     password,
	}
}

// NewClientWithToken creates a new Hetzner Robot client wrapper using token format "username:password"
func NewClientWithToken(token string) (*Client, error) {
	parts := strings.SplitN(token, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid token format, expected 'username:password'")
	}
	
	hrobotClient := hrobot.NewClient(hrobot.WithToken(token))
	return &Client{
		hrobotClient: hrobotClient,
		username:     parts[0],
		password:     parts[1],
	}, nil
}

// ListServers fetches all servers from Hetzner Robot API and returns formatted server information
func (c *Client) ListServers(ctx context.Context) ([]Server, error) {
	serverList, _, err := c.hrobotClient.Server.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch servers from Hetzner Robot API: %w", err)
	}

	servers := make([]Server, 0, len(serverList))
	for _, srv := range serverList {
		server := Server{
			ID:         strconv.Itoa(srv.ServerNumber),
			Name:       srv.ServerName, // Correct field name
			IP:         srv.ServerIP,
			Status:     srv.Status,
			Product:    srv.Product,
			DC:         srv.Dc,
			Traffic:    srv.Traffic,
			Cancelled:  srv.Cancelled,
			PaidUntil:  srv.PaidUntil,
			ServerType: srv.Product, // Using product as server type
		}



		servers = append(servers, server)
	}

	return servers, nil
}

// GetServerByID fetches detailed information for a specific server
func (c *Client) GetServerByID(ctx context.Context, serverID string) (*Server, error) {
	srv, _, err := c.hrobotClient.Server.GetServerById(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch server %s from Hetzner Robot API: %w", serverID, err)
	}

	server := &Server{
		ID:         strconv.Itoa(srv.ServerNumber),
		Name:       srv.ServerName, // Correct field name
		IP:         srv.ServerIP,
		Status:     srv.Status,
		Product:    srv.Product,
		DC:         srv.Dc,
		Traffic:    srv.Traffic,
		Cancelled:  srv.Cancelled,
		PaidUntil:  srv.PaidUntil,
		ServerType: srv.Product,
	}

	return server, nil
}

// ValidateCredentials tests the connection to Hetzner Robot API
func (c *Client) ValidateCredentials(ctx context.Context) error {
	_, _, err := c.hrobotClient.Server.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to validate Hetzner Robot credentials: %w", err)
	}
	return nil
}

// ListVSwitches fetches all vswitches from Hetzner Robot API and returns formatted vswitch information
func (c *Client) ListVSwitches(ctx context.Context) ([]VSwitch, error) {
	vswitchList, _, err := c.hrobotClient.VSwitch.GetVSwitchList(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch vswitches from Hetzner Robot API: %w", err)
	}

	vswitches := make([]VSwitch, 0, len(vswitchList))
	for _, vs := range vswitchList {
		vswitch := VSwitch{
			ID:        strconv.Itoa(vs.ID),
			Name:      vs.Name,
			VLAN:      vs.Vlan,
			Cancelled: vs.Cancelled,
		}
		vswitches = append(vswitches, vswitch)
	}

	return vswitches, nil
}

// CreateVSwitch creates a new vswitch with the given name and VLAN ID
func (c *Client) CreateVSwitch(ctx context.Context, name string, vlanID int) (*VSwitch, error) {
	opts := &hrobot.AddvSwitchOps{
		Name:    name,
		Vlan_ID: vlanID,
	}

	vswitchSingle, _, err := c.hrobotClient.VSwitch.AddVSwitch(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create vswitch: %w", err)
	}

	vswitch := &VSwitch{
		ID:        strconv.Itoa(vswitchSingle.ID),
		Name:      vswitchSingle.Name,
		VLAN:      vswitchSingle.Vlan,
		Cancelled: vswitchSingle.Cancelled,
	}

	return vswitch, nil
}

// DeleteVSwitch cancels/deletes a vswitch by ID with immediate cancellation
func (c *Client) DeleteVSwitch(ctx context.Context, vswitchID string) error {
	// Use "now" as cancellation date for immediate cancellation
	_, err := c.hrobotClient.VSwitch.CancelVSwitch(ctx, vswitchID, "now")
	if err != nil {
		return fmt.Errorf("failed to delete vswitch %s: %w", vswitchID, err)
	}
	return nil
}

// GetVSwitchDetails fetches detailed information for a specific vswitch including attached servers
func (c *Client) GetVSwitchDetails(ctx context.Context, vswitchID string) (*VSwitchDetails, error) {
	vswitchSingle, _, err := c.hrobotClient.VSwitch.GetVSwitchById(ctx, vswitchID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch vswitch details: %w", err)
	}

	// Check if vSwitch has cloud network attachments
	hasCloudNetwork := len(vswitchSingle.CloudNetwork) > 0

	details := &VSwitchDetails{
		VSwitch: VSwitch{
			ID:                      strconv.Itoa(vswitchSingle.ID),
			Name:                    vswitchSingle.Name,
			VLAN:                    vswitchSingle.Vlan,
			Cancelled:               vswitchSingle.Cancelled,
			HasCloudNetworkAttached: hasCloudNetwork,
		},
		AttachedServers: make([]VSwitchServer, 0),
	}

	// Parse attached servers from the interface{} slice
	for _, serverInterface := range vswitchSingle.Server {
		if serverMap, ok := serverInterface.(map[string]interface{}); ok {
			server := VSwitchServer{}

			if serverIP, ok := serverMap["server_ip"].(string); ok {
				server.ServerIP = serverIP
			}

			if serverNumber, ok := serverMap["server_number"].(float64); ok {
				server.ServerNumber = int(serverNumber)
			}

			if status, ok := serverMap["status"].(string); ok {
				server.Status = status
			}

			details.AttachedServers = append(details.AttachedServers, server)
		}
	}

	return details, nil
}

// AttachServersToVSwitch attaches multiple servers to a vswitch in a single API call
func (c *Client) AttachServersToVSwitch(ctx context.Context, vswitchID string, serverIDs []string) error {
	if len(serverIDs) == 0 {
		return nil
	}

	// The hrobot-go library expects interface{} for the server parameter
	// We need to pass the slice of server IDs
	_, err := c.hrobotClient.VSwitch.AddToServer(ctx, vswitchID, serverIDs)
	if err != nil {
		return fmt.Errorf("failed to attach servers %v to vswitch %s: %w", serverIDs, vswitchID, err)
	}
	return nil
}

// GetServerRescueStatus fetches the rescue mode status for a specific server
func (c *Client) GetServerRescueStatus(ctx context.Context, serverID string) (*ServerRescueStatus, error) {
	rescue, _, err := c.hrobotClient.Boot.GetRescue(ctx, serverID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rescue status for server %s: %w", serverID, err)
	}

	status := &ServerRescueStatus{
		ServerID:     serverID,
		ServerIP:     rescue.ServerIP,
		InRescueMode: rescue.Active,
		Status:       "checking",
	}

	// Extract password if available (interface{} could be string or nil)
	if rescue.Password != nil {
		if password, ok := rescue.Password.(string); ok {
			status.RescuePassword = password
		}
	}

	return status, nil
}

// EnableServerRescueMode enables rescue mode for a specific server using direct HTTP API call
func (c *Client) EnableServerRescueMode(ctx context.Context, serverID string) (*ServerRescueStatus, error) {
	// Create the API URL
	apiURL := fmt.Sprintf("https://robot-ws.your-server.de/boot/%s/rescue", serverID)

	// Prepare form data
	data := url.Values{}
	data.Set("os", "linux")

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.username, c.password)

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse JSON response
	var rescueResp RescueResponse
	if err := json.Unmarshal(body, &rescueResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Create status from response
	status := &ServerRescueStatus{
		ServerID:       serverID,
		ServerIP:       rescueResp.Rescue.ServerIP,
		InRescueMode:   rescueResp.Rescue.Active,
		RescuePassword: rescueResp.Rescue.Password,
		Status:         "enabling",
	}

	return status, nil
}

// DisableServerRescueMode disables rescue mode for a specific server using direct HTTP API call
func (c *Client) DisableServerRescueMode(ctx context.Context, serverID string) error {
	// Create the API URL
	apiURL := fmt.Sprintf("https://robot-ws.your-server.de/boot/%s/rescue", serverID)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "DELETE", apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set basic auth
	req.SetBasicAuth(c.username, c.password)

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body for error details
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ResetServer performs a hardware reset on a specific server
func (c *Client) ResetServer(ctx context.Context, serverID string) error {
	// Use "hw" for hardware reset
	_, _, err := c.hrobotClient.Reset.ExecuteReset(ctx, serverID, "hw")
	if err != nil {
		return fmt.Errorf("failed to reset server %s: %w", serverID, err)
	}
	return nil
}

// CheckServerRescueAvailability checks if rescue mode is available for a specific server
func (c *Client) CheckServerRescueAvailability(ctx context.Context, serverID string) (bool, error) {
	server, _, err := c.hrobotClient.Server.GetServerById(ctx, serverID)
	if err != nil {
		return false, fmt.Errorf("failed to get server details for %s: %w", serverID, err)
	}

	return server.Rescue, nil
}

// CheckAllServersRescueAvailability checks rescue availability for all servers and returns any that don't support it
func (c *Client) CheckAllServersRescueAvailability(ctx context.Context, servers []Server) ([]Server, error) {
	unsupportedServers := make([]Server, 0)

	for _, server := range servers {
		available, err := c.CheckServerRescueAvailability(ctx, server.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to check rescue availability for server %s: %w", server.Name, err)
		}

		if !available {
			unsupportedServers = append(unsupportedServers, server)
		}
	}

	return unsupportedServers, nil
}

package config

// CreateConfig represents the configuration for creating a cluster
type CreateConfig struct {
	// Core configuration
	Provider      string `validate:"required,oneof=aws digital-ocean hetzner hetzner-robot linode gcloud kind"`
	Configuration string `validate:"omitempty,file"`
	Domain        string `validate:"required,domain_name"`
	Name          string `validate:"-"` // Derived from Domain by replacing dots with dashes
	Email         string `validate:"required,email"`
	PaaSFeatures  string `validate:"omitempty,paas_features"`

	// Terraform state configuration
	TerraformState *TerraformStateConfig `validate:"omitempty"`

	// Provider-specific configurations will be validated by their respective providers
	AWS          *AWSConfig          `validate:"omitempty"`
	DigitalOcean *DigitalOceanConfig `validate:"omitempty"`
	Hetzner      *HetznerConfig      `validate:"omitempty"`
	HetznerRobot *HetznerRobotConfig `validate:"omitempty"`
	Linode       *LinodeConfig       `validate:"omitempty"`
	GCloud       *GCloudConfig       `validate:"omitempty"`
	Kind         *KindConfig         `validate:"omitempty"`
}

// AWSConfig represents AWS-specific configuration
type AWSConfig struct {
	AccessKeyID     string `validate:"required_with=AWSConfig"`
	SecretAccessKey string `validate:"required_with=AWSConfig"`
	Region          string `validate:"required_with=AWSConfig"`
}

// DigitalOceanConfig represents DigitalOcean-specific configuration
type DigitalOceanConfig struct {
	Token     string `validate:"required_with=DigitalOceanConfig"`
	Nodes     string `validate:"required_with=DigitalOceanConfig"`
	NodesSize string `validate:"required_with=DigitalOceanConfig"`
	Region    string `validate:"required_with=DigitalOceanConfig"`
}

// HetznerConfig represents Hetzner Cloud-specific configuration
type HetznerConfig struct {
	Token string `validate:"required_with=HetznerConfig"`
}

// HetznerRobotServer represents a selected Hetzner Robot server for templating
type HetznerRobotServer struct {
	ID                        string `json:"id"`
	Name                      string `json:"name"`
	IP                        string `json:"ip"`
	PrivateIP                 string `json:"private_ip"`
	Status                    string `json:"status"`
	Product                   string `json:"product"`
	DC                        string `json:"dc"`
	Role                      string `json:"role"` // "control-plane" or "worker"
	RescuePassword            string `json:"rescue_password"`

	// Talos network configuration
	PublicNetworkInterface    string `json:"public_network_interface"`
	PublicAddressSubnet       string `json:"public_address_subnet"`
	PublicIPv4Gateway         string `json:"public_ipv4_gateway"`
	PrivateAddressSubnet      string `json:"private_address_subnet"`
	PrivateIPv4Gateway        string `json:"private_ipv4_gateway"`
	InstallationDisk          string `json:"installation_disk"`
}

// HetznerRobotConfig represents Hetzner Robot-specific configuration
type HetznerRobotConfig struct {
	Username        string                     `validate:"required_with=HetznerRobotConfig"`
	Password        string                     `validate:"required_with=HetznerRobotConfig"`
	CloudToken      string                     `validate:"required_with=HetznerRobotConfig"`
	SelectedServers []HetznerRobotServer       `validate:"omitempty"`
	RescuePasswords map[string]string          `validate:"omitempty"`
	VSwitchID       string                     `validate:"omitempty"`

	// Network configuration for cloud resources
	NetworkConfig   *HetznerRobotNetworkConfig `validate:"omitempty"`

	// Talos configuration
	TalosConfig     *HetznerRobotTalosConfig   `validate:"omitempty"`
}

// HetznerRobotNetworkConfig represents network configuration for Hetzner Robot clusters
type HetznerRobotNetworkConfig struct {
	Location                      string `json:"location" yaml:"location"`
	NetworkZone                   string `json:"network_zone" yaml:"network-zone"`
	ClusterNetworkIPRange         string `json:"cluster_network_ip_range" yaml:"cluster-network-ip-range"`
	ClusterVSwitchSubnetIPRange   string `json:"cluster_vswitch_subnet_ip_range" yaml:"cluster-vswitch-subnet-ip-range"`
	ClusterSubnetIPRange          string `json:"cluster_subnet_ip_range" yaml:"cluster-subnet-ip-range"`
}

// HetznerRobotTalosConfig represents Talos configuration for Hetzner Robot clusters
type HetznerRobotTalosConfig struct {
	ClusterEndpoint           string `json:"cluster_endpoint" yaml:"cluster-endpoint"`
	VLANID                    int    `json:"vlan_id" yaml:"vlan-id"`
	VSwitchSubnetIPRange      string `json:"vswitch_subnet_ip_range" yaml:"vswitch-subnet-ip-range"`
}

// LinodeConfig represents Linode-specific configuration
type LinodeConfig struct {
	Token string `validate:"required_with=LinodeConfig"`
}

// GCloudConfig represents Google Cloud-specific configuration
type GCloudConfig struct {
	ServiceAccountKey string `validate:"required_with=GCloudConfig,file"`
	ProjectID         string `validate:"required_with=GCloudConfig"`
	Region            string `validate:"required_with=GCloudConfig"`
}

// KindConfig represents Kind (Kubernetes in Docker) specific configuration
type KindConfig struct {
	Nodes             string `validate:"required_with=KindConfig,number"`
	Storage           string `validate:"required_with=KindConfig,kind_storage"`
	KubernetesVersion string `validate:"omitempty"`
}

// TerraformStateConfig represents Terraform state storage configuration
type TerraformStateConfig struct {
	S3Bucket       string `validate:"required_with=TerraformStateConfig"`
	S3Region       string `validate:"required_with=TerraformStateConfig"`
	S3AccessKey    string `validate:"required_with=TerraformStateConfig"`
	S3AccessSecret string `validate:"required_with=TerraformStateConfig"`
}

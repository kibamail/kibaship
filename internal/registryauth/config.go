package registryauth

// Config holds the configuration for the registry auth service
// All values are hardcoded since the deployment environment is fully known
type Config struct {
	JWT struct {
		Issuer         string
		ExpirationSec  int
		PrivateKeyPath string
	}
	Registry struct {
		ServiceName string
	}
	Server struct {
		Listen string
	}
	Cache struct {
		TTLSeconds int
	}
}

// LoadConfig returns the configuration with hardcoded values
// The deployment knows:
// - Keys are in /etc/registry-auth-keys/ (mounted from registry-auth-keys Secret)
// - Registry service is called "docker-registry"
// - We run on port 8080
// - Tokens valid for 5 minutes
// - Credentials cached for 5 minutes
func LoadConfig() Config {
	cfg := Config{}

	// JWT configuration
	cfg.JWT.Issuer = "registry-token-issuer"
	cfg.JWT.ExpirationSec = 300 // 5 minutes
	cfg.JWT.PrivateKeyPath = "/etc/registry-auth-keys/tls.key"

	// Registry configuration
	cfg.Registry.ServiceName = "docker-registry"

	// Server configuration
	cfg.Server.Listen = ":8080"

	// Cache configuration
	cfg.Cache.TTLSeconds = 300 // 5 minutes

	return cfg
}

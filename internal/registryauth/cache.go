package registryauth

import (
	"sync"
	"time"
)

// CachedCredentials represents cached credentials with expiration
type CachedCredentials struct {
	Username  string
	Password  string
	ExpiresAt time.Time
}

// CredentialCache provides thread-safe caching of credentials
type CredentialCache struct {
	cache map[string]CachedCredentials
	ttl   time.Duration
	mu    sync.RWMutex
}

// NewCredentialCache creates a new credential cache
func NewCredentialCache(ttlSeconds int) *CredentialCache {
	return &CredentialCache{
		cache: make(map[string]CachedCredentials),
		ttl:   time.Duration(ttlSeconds) * time.Second,
	}
}

// Get retrieves cached credentials if they exist and haven't expired
func (cc *CredentialCache) Get(namespace string) (*Credentials, bool) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	cached, ok := cc.cache[namespace]
	if !ok {
		return nil, false
	}

	if time.Now().After(cached.ExpiresAt) {
		return nil, false
	}

	return &Credentials{
		Username: cached.Username,
		Password: cached.Password,
	}, true
}

// Set stores credentials in the cache with TTL
func (cc *CredentialCache) Set(namespace string, creds *Credentials) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	cc.cache[namespace] = CachedCredentials{
		Username:  creds.Username,
		Password:  creds.Password,
		ExpiresAt: time.Now().Add(cc.ttl),
	}
}

// Cleanup removes expired entries from the cache
func (cc *CredentialCache) Cleanup() {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	now := time.Now()
	for namespace, cached := range cc.cache {
		if now.After(cached.ExpiresAt) {
			delete(cc.cache, namespace)
		}
	}
}

// StartCleanupRoutine starts a background goroutine to periodically clean up expired entries
func (cc *CredentialCache) StartCleanupRoutine() {
	ticker := time.NewTicker(2 * time.Minute)
	go func() {
		for range ticker.C {
			cc.Cleanup()
		}
	}()
}

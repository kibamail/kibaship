package registryauth

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

// Handler handles authentication requests from Docker clients
type Handler struct {
	validator      *Validator
	tokenGenerator *TokenGenerator
	serviceName    string
}

// TokenResponse is the response format expected by Docker clients
type TokenResponse struct {
	Token       string `json:"token"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	IssuedAt    string `json:"issued_at"`
}

// NewHandler creates a new authentication handler
func NewHandler(validator *Validator, tokenGenerator *TokenGenerator, serviceName string) *Handler {
	return &Handler{
		validator:      validator,
		tokenGenerator: tokenGenerator,
		serviceName:    serviceName,
	}
}

// ServeAuth handles the /auth endpoint
func (h *Handler) ServeAuth(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	service := r.URL.Query().Get("service")

	scope := r.URL.Query().Get("scope")
	account := r.URL.Query().Get("account")

	log.Printf("auth request: service=%s scope=%s account=%s", service, scope, account)

	// Extract Basic Auth credentials
	username, password, ok := r.BasicAuth()
	if !ok {
		log.Printf("auth: missing or invalid Authorization header")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse all scope parameters to extract repositories and actions
	scopes := r.URL.Query()["scope"]
	if len(scopes) == 0 {
		log.Printf("auth: missing scope parameter")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var accessGrants []AccessEntry
	var authenticatedNamespace string

	// Determine the authenticated namespace from the username
	// The username should match the namespace that owns the credentials
	authenticatedNamespace = username

	log.Printf("auth: authenticated namespace=%s", authenticatedNamespace)

	// Validate credentials against the authenticated namespace
	if !h.validator.ValidateCredentials(r.Context(), authenticatedNamespace, username, password) {
		log.Printf("auth: invalid credentials for namespace=%s", authenticatedNamespace)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Process each scope
	for _, scopeStr := range scopes {
		repo, actions, err := parseScope(scopeStr)
		if err != nil {
			log.Printf("auth: failed to parse scope %s: %v", scopeStr, err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		// Extract namespace from repository path
		repoNamespace, err := extractNamespaceFromRepo(repo)
		if err != nil {
			log.Printf("auth: failed to extract namespace from repo=%s: %v", repo, err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		log.Printf("auth: processing scope=%s repo=%s namespace=%s actions=%v", scopeStr, repo, repoNamespace, actions)

		// Security policy:
		// 1. Full access (read/write) only to authenticated namespace
		// 2. Read-only access allowed to other namespaces (for layer mounting)
		if repoNamespace == authenticatedNamespace {
			// Full access to own namespace
			accessGrants = append(accessGrants, AccessEntry{
				Type:    "repository",
				Name:    repo,
				Actions: actions,
			})
			log.Printf("auth: granted full access to own namespace repo=%s actions=%v", repo, actions)
		} else {
			// Cross-namespace access: only allow read operations
			allowedActions := []string{}
			for _, action := range actions {
				if action == "pull" {
					allowedActions = append(allowedActions, action)
				}
			}

			if len(allowedActions) > 0 {
				accessGrants = append(accessGrants, AccessEntry{
					Type:    "repository",
					Name:    repo,
					Actions: allowedActions,
				})
				log.Printf("auth: granted cross-namespace read access repo=%s actions=%v", repo, allowedActions)
			} else {
				log.Printf("auth: denied cross-namespace write access repo=%s actions=%v", repo, actions)
			}
		}
	}

	// Generate JWT token
	token, expiresAt, err := h.tokenGenerator.GenerateToken(username, h.serviceName, accessGrants)
	if err != nil {
		log.Printf("auth: failed to generate token: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	expiresIn := int(time.Until(expiresAt).Seconds())

	// Return token response
	response := TokenResponse{
		Token:       token,
		AccessToken: token,
		ExpiresIn:   expiresIn,
		IssuedAt:    time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("auth: failed to encode response: %v", err)
		return
	}

	log.Printf("auth: token issued for namespace=%s with %d access grants", authenticatedNamespace, len(accessGrants))
}

// parseScope parses Docker registry scope format: "repository:<name>:<actions>"
// Example: "repository:test-app/myapp:push,pull" => ("test-app/myapp", ["push", "pull"])
func parseScope(scope string) (string, []string, error) {
	parts := strings.SplitN(scope, ":", 3)
	if len(parts) != 3 || parts[0] != "repository" {
		return "", nil, http.ErrAbortHandler
	}

	repo := parts[1]
	actionsStr := parts[2]
	actions := strings.Split(actionsStr, ",")

	return repo, actions, nil
}

// extractNamespaceFromRepo extracts the namespace from a repository path
// Example: "test-build/img-v2" => "test-build"
// Example: "test-build/subdir/img" => "test-build"
func extractNamespaceFromRepo(repo string) (string, error) {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) < 1 || parts[0] == "" {
		return "", http.ErrAbortHandler
	}
	return parts[0], nil
}

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

	// Parse scope to extract repository and actions
	var accessGrants []AccessEntry
	var namespace string

	if scope == "" {
		log.Printf("auth: missing scope parameter")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	repo, actions, err := parseScope(scope)
	if err != nil {
		log.Printf("auth: failed to parse scope: %v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Extract namespace from repository path (e.g., "test-build/img-v2" → "test-build")
	namespace, err = extractNamespaceFromRepo(repo)
	if err != nil {
		log.Printf("auth: failed to extract namespace from repo=%s: %v", repo, err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	log.Printf("auth: extracted namespace=%s from repo=%s", namespace, repo)

	accessGrants = append(accessGrants, AccessEntry{
		Type:    "repository",
		Name:    repo,
		Actions: actions,
	})

	// Validate credentials against <namespace>-registry-credentials secret
	if !h.validator.ValidateCredentials(r.Context(), namespace, username, password) {
		log.Printf("auth: invalid credentials for namespace=%s", namespace)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
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

	log.Printf("auth: token issued for namespace=%s scope=%s", namespace, scope)
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

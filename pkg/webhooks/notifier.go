package webhooks

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	platformv1alpha1 "github.com/kibamail/kibaship-operator/api/v1alpha1"
)

// Notifier defines the interface for sending webhook events.
type Notifier interface {
	NotifyProjectStatusChange(ctx context.Context, evt ProjectStatusEvent) error
	NotifyApplicationStatusChange(ctx context.Context, evt ApplicationStatusEvent) error
	NotifyApplicationDomainStatusChange(ctx context.Context, evt ApplicationDomainStatusEvent) error
	NotifyDeploymentStatusChange(ctx context.Context, evt DeploymentStatusEvent) error
}

// ProjectStatusEvent is the payload for project status change notifications.
type ProjectStatusEvent struct {
	Type          string                   `json:"type"`
	PreviousPhase string                   `json:"previousPhase"`
	NewPhase      string                   `json:"newPhase"`
	Project       platformv1alpha1.Project `json:"project"`
	Timestamp     time.Time                `json:"timestamp"`
}

// ApplicationStatusEvent is the payload for application status change notifications.
type ApplicationStatusEvent struct {
	Type          string                       `json:"type"`
	PreviousPhase string                       `json:"previousPhase"`
	NewPhase      string                       `json:"newPhase"`
	Application   platformv1alpha1.Application `json:"application"`
	Timestamp     time.Time                    `json:"timestamp"`
}

// ApplicationDomainStatusEvent is the payload for application domain status change notifications.
type ApplicationDomainStatusEvent struct {
	Type              string                             `json:"type"`
	PreviousPhase     string                             `json:"previousPhase"`
	NewPhase          string                             `json:"newPhase"`
	ApplicationDomain platformv1alpha1.ApplicationDomain `json:"applicationDomain"`
	Timestamp         time.Time                          `json:"timestamp"`
}

// DeploymentStatusEvent is the payload for deployment status change notifications.
type DeploymentStatusEvent struct {
	Type          string                      `json:"type"`
	PreviousPhase string                      `json:"previousPhase"`
	NewPhase      string                      `json:"newPhase"`
	Deployment    platformv1alpha1.Deployment `json:"deployment"`
	Timestamp     time.Time                   `json:"timestamp"`
}

// NoopNotifier is a drop-in that does nothing.
type NoopNotifier struct{}

func (n NoopNotifier) NotifyProjectStatusChange(ctx context.Context, evt ProjectStatusEvent) error {
	return nil
}
func (n NoopNotifier) NotifyApplicationStatusChange(ctx context.Context, evt ApplicationStatusEvent) error {
	return nil
}
func (n NoopNotifier) NotifyApplicationDomainStatusChange(ctx context.Context, evt ApplicationDomainStatusEvent) error {
	return nil
}
func (n NoopNotifier) NotifyDeploymentStatusChange(ctx context.Context, evt DeploymentStatusEvent) error {
	return nil
}

// HTTPNotifier implements Notifier using retryablehttp and HMAC-SHA256 signing.
type HTTPNotifier struct {
	client     *retryablehttp.Client
	targetURL  string
	signingKey []byte
}

// NewHTTPNotifier constructs an HTTPNotifier with sane defaults.
// Retry policy: retry on 408, 429, and all 5xx; backoff with jitter.
func NewHTTPNotifier(targetURL string, signingKey []byte) *HTTPNotifier {
	c := retryablehttp.NewClient()
	c.RetryMax = 5
	c.RetryWaitMin = 500 * time.Millisecond
	c.RetryWaitMax = 10 * time.Second
	c.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		if err != nil {
			return true, nil
		}
		if resp == nil {
			return true, nil
		}
		code := resp.StatusCode
		if code == http.StatusRequestTimeout || code == http.StatusTooManyRequests {
			return true, nil
		}
		return code >= 500, nil
	}
	c.Logger = nil // keep quiet in tests; rely on our logs
	return &HTTPNotifier{client: c, targetURL: targetURL, signingKey: signingKey}
}

func (n *HTTPNotifier) postSigned(ctx context.Context, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	h := hmac.New(sha256.New, n.signingKey)
	_, _ = h.Write(body)
	sig := hex.EncodeToString(h.Sum(nil))

	req, err := retryablehttp.NewRequest(http.MethodPost, n.targetURL, body)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Kibaship-Signature", sig)
	_, err = n.client.Do(req)
	return err
}

func (n *HTTPNotifier) NotifyProjectStatusChange(ctx context.Context, evt ProjectStatusEvent) error {
	return n.postSigned(ctx, evt)
}

func (n *HTTPNotifier) NotifyApplicationStatusChange(ctx context.Context, evt ApplicationStatusEvent) error {
	return n.postSigned(ctx, evt)
}

func (n *HTTPNotifier) NotifyApplicationDomainStatusChange(ctx context.Context, evt ApplicationDomainStatusEvent) error {
	return n.postSigned(ctx, evt)
}

func (n *HTTPNotifier) NotifyDeploymentStatusChange(ctx context.Context, evt DeploymentStatusEvent) error {
	return n.postSigned(ctx, evt)
}

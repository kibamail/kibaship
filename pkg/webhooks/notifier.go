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
	platformv1alpha1 "github.com/kibamail/kibaship/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Notifier defines the interface for sending webhook events.
type Notifier interface {
	NotifyProjectStatusChange(ctx context.Context, evt ProjectStatusEvent) error
	NotifyEnvironmentStatusChange(ctx context.Context, evt EnvironmentStatusEvent) error
	NotifyApplicationStatusChange(ctx context.Context, evt ApplicationStatusEvent) error
	NotifyApplicationDomainStatusChange(ctx context.Context, evt ApplicationDomainStatusEvent) error
	NotifyDeploymentStatusChange(ctx context.Context, evt DeploymentStatusEvent) error
	// NotifyOptimizedDeploymentStatusChange sends memory-optimized deployment status notifications
	NotifyOptimizedDeploymentStatusChange(ctx context.Context, evt OptimizedDeploymentStatusEvent) error
}

// ProjectStatusEvent is the payload for project status change notifications.
type ProjectStatusEvent struct {
	Type          string                   `json:"type"`
	PreviousPhase string                   `json:"previousPhase"`
	NewPhase      string                   `json:"newPhase"`
	Project       platformv1alpha1.Project `json:"project"`
	Timestamp     time.Time                `json:"timestamp"`
}

// EnvironmentStatusEvent is the payload for environment status change notifications.
type EnvironmentStatusEvent struct {
	Type          string                       `json:"type"`
	PreviousPhase string                       `json:"previousPhase"`
	NewPhase      string                       `json:"newPhase"`
	Environment   platformv1alpha1.Environment `json:"environment"`
	Timestamp     time.Time                    `json:"timestamp"`
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
	// Certificate is optionally included for ApplicationDomain events when a CertificateRef exists
	Certificate any       `json:"certificate,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// DeploymentStatusEvent is the payload for deployment status change notifications.
type DeploymentStatusEvent struct {
	Type          string                      `json:"type"`
	PreviousPhase string                      `json:"previousPhase"`
	NewPhase      string                      `json:"newPhase"`
	Deployment    platformv1alpha1.Deployment `json:"deployment"`
	// PipelineRun is optionally included for Deployment events when a matching PipelineRun exists
	PipelineRun any       `json:"pipelineRun,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// OptimizedDeploymentStatusEvent is a memory-optimized version of DeploymentStatusEvent
// that contains only essential fields to reduce webhook payload size and memory usage.
type OptimizedDeploymentStatusEvent struct {
	Type          string `json:"type"`
	PreviousPhase string `json:"previousPhase"`
	NewPhase      string `json:"newPhase"`
	// Only essential deployment fields
	DeploymentRef struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
		UUID      string `json:"uuid"`
		Phase     string `json:"phase"`
		Slug      string `json:"slug"`
	} `json:"deploymentRef"`
	// Only essential PipelineRun fields
	PipelineRunRef *struct {
		Name   string `json:"name"`
		Status string `json:"status"`
		Reason string `json:"reason"`
	} `json:"pipelineRunRef,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// NoopNotifier is a drop-in that does nothing.
type NoopNotifier struct{}

func (n NoopNotifier) NotifyProjectStatusChange(ctx context.Context, evt ProjectStatusEvent) error {
	return nil
}
func (n NoopNotifier) NotifyEnvironmentStatusChange(ctx context.Context, evt EnvironmentStatusEvent) error {
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
func (n NoopNotifier) NotifyOptimizedDeploymentStatusChange(
	ctx context.Context, evt OptimizedDeploymentStatusEvent,
) error {
	return nil
}

// HTTPNotifier implements Notifier using retryablehttp and HMAC-SHA256 signing.
type HTTPNotifier struct {
	client     *retryablehttp.Client
	targetURL  string
	signingKey []byte
	reader     client.Reader // cache-backed reader for enrichment
}

// NewHTTPNotifier constructs an HTTPNotifier with sane defaults.
// Retry policy: retry on 408, 429, and all 5xx; backoff with jitter.
func NewHTTPNotifier(targetURL string, signingKey []byte, reader client.Reader) *HTTPNotifier {
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
	return &HTTPNotifier{client: c, targetURL: targetURL, signingKey: signingKey, reader: reader}
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

func (n *HTTPNotifier) NotifyEnvironmentStatusChange(ctx context.Context, evt EnvironmentStatusEvent) error {
	return n.postSigned(ctx, evt)
}

func (n *HTTPNotifier) NotifyApplicationStatusChange(ctx context.Context, evt ApplicationStatusEvent) error {
	return n.postSigned(ctx, evt)
}

func (n *HTTPNotifier) NotifyApplicationDomainStatusChange(
	ctx context.Context,
	evt ApplicationDomainStatusEvent,
) error {
	// enrich with Certificate when available
	ref := evt.ApplicationDomain.Status.CertificateRef
	if n.reader != nil && ref != nil && ref.Name != "" && ref.Namespace != "" {
		u := &unstructured.Unstructured{}
		u.SetGroupVersionKind(schema.GroupVersionKind{Group: "cert-manager.io", Version: "v1", Kind: "Certificate"})
		_ = n.reader.Get(ctx, client.ObjectKey{Name: ref.Name, Namespace: ref.Namespace}, u)
		if len(u.Object) > 0 {
			evt.Certificate = u.Object
		}
	}
	return n.postSigned(ctx, evt)
}

func (n *HTTPNotifier) NotifyDeploymentStatusChange(ctx context.Context, evt DeploymentStatusEvent) error {
	// enrich with latest PipelineRun when available and not already provided
	if n.reader != nil && evt.PipelineRun == nil {
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(schema.GroupVersionKind{Group: "tekton.dev", Version: "v1", Kind: "PipelineRunList"})
		_ = n.reader.List(ctx, list,
			client.InNamespace(evt.Deployment.Namespace),
			client.MatchingLabels(map[string]string{"deployment.kibaship.com/name": evt.Deployment.Name}),
		)
		if len(list.Items) > 0 {
			// pick newest by creationTimestamp
			latest := list.Items[0]
			for _, it := range list.Items[1:] {
				if it.GetCreationTimestamp().After(latest.GetCreationTimestamp().Time) {
					latest = it
				}
			}
			evt.PipelineRun = latest.Object
		}
	}
	return n.postSigned(ctx, evt)
}

func (n *HTTPNotifier) NotifyOptimizedDeploymentStatusChange(
	ctx context.Context, evt OptimizedDeploymentStatusEvent,
) error {
	return n.postSigned(ctx, evt)
}

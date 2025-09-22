package main

import (
	"context"
	"fmt"

	v1alpha1 "github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"github.com/cert-manager/cert-manager/pkg/acme/webhook/cmd"
	"github.com/redis/go-redis/v9"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var groupName = "dns.kibaship.com"

func main() {
	fmt.Printf("Starting Kibaship cert-manager DNS01 webhook (group=%s)\n", groupName)
	cmd.RunWebhookServer(groupName, &kibashipSolver{})
}

type kibashipSolver struct{
	kube  kubernetes.Interface
	redis *redis.ClusterClient
}

func (s *kibashipSolver) Name() string { return "kibaship" }

// Initialize sets up K8s clients now; Valkey clients will be created on-demand later.
func (s *kibashipSolver) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	clientset, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		return fmt.Errorf("create k8s client: %w", err)
	}
	s.kube = clientset

	// Establish Valkey connectivity before reporting ready
	ctx := context.Background()
	if err := s.bootValkey(ctx, kubeClientConfig); err != nil {
		return fmt.Errorf("valkey startup: %w", err)
	}
	fmt.Println("kibaship webhook initialized: k8s + valkey ready")
	return nil
}

// Present is called by cert-manager to create the DNS01 TXT record. We'll emit to Valkey later.
func (s *kibashipSolver) Present(ch *v1alpha1.ChallengeRequest) error {
	// Scaffolding: do nothing yet
	fmt.Printf("Present called for fqdn=%s zone=%s\n", ch.ResolvedFQDN, ch.ResolvedZone)
	return nil
}

// CleanUp should remove the TXT record. We'll emit a cleanup event later.
func (s *kibashipSolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	// Scaffolding: do nothing yet
	fmt.Printf("CleanUp called for fqdn=%s zone=%s\n", ch.ResolvedFQDN, ch.ResolvedZone)
	return nil
}


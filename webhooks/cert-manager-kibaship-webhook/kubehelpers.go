package main

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// readSecretValue reads a single key from a Secret. If ref.Namespace is empty,
// defaultNS is used. Returns the value as string.
func (s *kibashipSolver) readSecretValue(ctx context.Context, defaultNS string, ref *SecretRef) (string, error) {
	if ref == nil {
		return "", nil
	}
	if s.kube == nil {
		return "", fmt.Errorf("k8s client not initialized")
	}
	ns := ref.Namespace
	if ns == "" {
		ns = defaultNS
	}
	if ns == "" {
		return "", fmt.Errorf("secret namespace is empty")
	}
	sec, err := s.kube.CoreV1().Secrets(ns).Get(ctx, ref.Name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get secret %s/%s: %w", ns, ref.Name, err)
	}
	b, ok := sec.Data[ref.Key]
	if !ok {
		return "", fmt.Errorf("secret key %q not found in %s/%s", ref.Key, ns, ref.Name)
	}
	return string(b), nil
}


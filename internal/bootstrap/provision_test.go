package bootstrap

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestEnsureGateway(t *testing.T) {
	scheme := runtime.NewScheme()
	ctx := context.Background()

	tests := []struct {
		name       string
		existing   []client.Object
		wantErr    bool
		wantCreate bool
	}{
		{
			name:       "Creates Gateway when it doesn't exist",
			existing:   []client.Object{},
			wantErr:    false,
			wantCreate: true,
		},
		{
			name: "Does not create Gateway when it already exists",
			existing: []client.Object{
				&unstructured.Unstructured{
					Object: map[string]any{
						"apiVersion": "gateway.networking.k8s.io/v1",
						"kind":       "Gateway",
						"metadata": map[string]any{
							"name":      gatewayName,
							"namespace": gatewayAPISystemNS,
						},
						"spec": map[string]any{
							"gatewayClassName": "cilium",
							"listeners":        []any{},
						},
					},
				},
			},
			wantErr:    false,
			wantCreate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.existing...).
				Build()

			err := ensureGateway(ctx, fakeClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("ensureGateway() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Verify Gateway was created or already exists
			obj := &unstructured.Unstructured{}
			obj.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "gateway.networking.k8s.io",
				Version: "v1",
				Kind:    "Gateway",
			})

			err = fakeClient.Get(ctx, client.ObjectKey{
				Namespace: gatewayAPISystemNS,
				Name:      gatewayName,
			}, obj)

			if err != nil {
				if !errors.IsNotFound(err) {
					t.Errorf("Failed to get Gateway: %v", err)
				} else if tt.wantCreate {
					t.Errorf("Expected Gateway to be created, but it doesn't exist")
				}
				return
			}

			// Only verify spec details when we expected creation
			if !tt.wantCreate {
				return
			}

			// Verify Gateway spec
			spec, ok := obj.Object["spec"].(map[string]any)
			if !ok {
				t.Errorf("Gateway spec is not a map")
				return
			}

			// Verify gateway class
			gatewayClass, ok := spec["gatewayClassName"].(string)
			if !ok || gatewayClass != "cilium" {
				t.Errorf("Gateway gatewayClassName = %v, want 'cilium'", gatewayClass)
			}

			// Verify listeners
			listeners, ok := spec["listeners"].([]any)
			if !ok {
				t.Errorf("Gateway listeners is not an array")
				return
			}

			if len(listeners) != 5 {
				t.Errorf("Gateway has %d listeners, want 5", len(listeners))
			}

			// Verify listener names
			expectedListeners := map[string]bool{
				"http":         false,
				"https":        false,
				"mysql-tls":    false,
				"valkey-tls":   false,
				"postgres-tls": false,
			}

			for _, l := range listeners {
				listener, ok := l.(map[string]any)
				if !ok {
					t.Errorf("Listener is not a map")
					continue
				}

				name, ok := listener["name"].(string)
				if !ok {
					t.Errorf("Listener name is not a string")
					continue
				}

				if _, exists := expectedListeners[name]; exists {
					expectedListeners[name] = true
				} else {
					t.Errorf("Unexpected listener: %s", name)
				}
			}

			// Check all expected listeners are present
			for name, found := range expectedListeners {
				if !found {
					t.Errorf("Expected listener %s not found", name)
				}
			}
		})
	}
}

func TestEnsureGatewayReferenceGrant(t *testing.T) {
	scheme := runtime.NewScheme()
	ctx := context.Background()

	tests := []struct {
		name       string
		existing   []client.Object
		wantErr    bool
		wantCreate bool
	}{
		{
			name:       "Creates ReferenceGrant when it doesn't exist",
			existing:   []client.Object{},
			wantErr:    false,
			wantCreate: true,
		},
		{
			name: "Does not create ReferenceGrant when it already exists",
			existing: []client.Object{
				&unstructured.Unstructured{
					Object: map[string]any{
						"apiVersion": "gateway.networking.k8s.io/v1beta1",
						"kind":       "ReferenceGrant",
						"metadata": map[string]any{
							"name":      "gateway-to-certificates",
							"namespace": certificatesNS,
						},
						"spec": map[string]any{
							"from": []any{},
							"to":   []any{},
						},
					},
				},
			},
			wantErr:    false,
			wantCreate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.existing...).
				Build()

			err := ensureGatewayReferenceGrant(ctx, fakeClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("ensureGatewayReferenceGrant() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Verify ReferenceGrant was created or already exists
			obj := &unstructured.Unstructured{}
			obj.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "gateway.networking.k8s.io",
				Version: "v1beta1",
				Kind:    "ReferenceGrant",
			})

			err = fakeClient.Get(ctx, client.ObjectKey{
				Namespace: certificatesNS,
				Name:      "gateway-to-certificates",
			}, obj)

			if err != nil {
				if !errors.IsNotFound(err) {
					t.Errorf("Failed to get ReferenceGrant: %v", err)
				} else if tt.wantCreate {
					t.Errorf("Expected ReferenceGrant to be created, but it doesn't exist")
				}
				return
			}

			// Only verify spec details when we expected creation
			if !tt.wantCreate {
				return
			}

			// Verify ReferenceGrant spec
			spec, ok := obj.Object["spec"].(map[string]any)
			if !ok {
				t.Errorf("ReferenceGrant spec is not a map")
				return
			}

			// Verify from field
			from, ok := spec["from"].([]any)
			if !ok || len(from) == 0 {
				t.Errorf("ReferenceGrant from field is invalid")
				return
			}

			fromEntry, ok := from[0].(map[string]any)
			if !ok {
				t.Errorf("ReferenceGrant from entry is not a map")
				return
			}

			if fromEntry["namespace"] != gatewayAPISystemNS {
				t.Errorf("ReferenceGrant from.namespace = %v, want %s", fromEntry["namespace"], gatewayAPISystemNS)
			}

			if fromEntry["kind"] != "Gateway" {
				t.Errorf("ReferenceGrant from.kind = %v, want 'Gateway'", fromEntry["kind"])
			}

			// Verify to field
			to, ok := spec["to"].([]any)
			if !ok || len(to) == 0 {
				t.Errorf("ReferenceGrant to field is invalid")
				return
			}

			toEntry, ok := to[0].(map[string]any)
			if !ok {
				t.Errorf("ReferenceGrant to entry is not a map")
				return
			}

			if toEntry["kind"] != "Secret" {
				t.Errorf("ReferenceGrant to.kind = %v, want 'Secret'", toEntry["kind"])
			}
		})
	}
}

func TestEnsureWildcardCertificate(t *testing.T) {
	scheme := runtime.NewScheme()
	ctx := context.Background()
	baseDomain := "example.com"

	tests := []struct {
		name       string
		existing   []client.Object
		wantErr    bool
		wantCreate bool
	}{
		{
			name:       "Creates wildcard certificate when it doesn't exist",
			existing:   []client.Object{},
			wantErr:    false,
			wantCreate: true,
		},
		{
			name: "Does not create certificate when it already exists",
			existing: []client.Object{
				&unstructured.Unstructured{
					Object: map[string]any{
						"apiVersion": "cert-manager.io/v1",
						"kind":       "Certificate",
						"metadata": map[string]any{
							"name":      wildcardCertName,
							"namespace": certificatesNS,
						},
						"spec": map[string]any{
							"secretName": wildcardCertName,
							"issuerRef":  map[string]any{"name": issuerName, "kind": "ClusterIssuer"},
							"dnsNames":   []any{},
						},
					},
				},
			},
			wantErr:    false,
			wantCreate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.existing...).
				Build()

			err := ensureWildcardCertificate(ctx, fakeClient, baseDomain)
			if (err != nil) != tt.wantErr {
				t.Errorf("ensureWildcardCertificate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Verify Certificate was created or already exists
			obj := &unstructured.Unstructured{}
			obj.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "cert-manager.io",
				Version: "v1",
				Kind:    "Certificate",
			})

			err = fakeClient.Get(ctx, client.ObjectKey{
				Namespace: certificatesNS,
				Name:      wildcardCertName,
			}, obj)

			if err != nil {
				if !errors.IsNotFound(err) {
					t.Errorf("Failed to get Certificate: %v", err)
				} else if tt.wantCreate {
					t.Errorf("Expected Certificate to be created, but it doesn't exist")
				}
				return
			}

			// Only verify spec details when we expected creation
			if !tt.wantCreate {
				return
			}

			// Verify Certificate spec
			spec, ok := obj.Object["spec"].(map[string]any)
			if !ok {
				t.Errorf("Certificate spec is not a map")
				return
			}

			// Verify dnsNames includes *.apps.example.com
			dnsNames, ok := spec["dnsNames"].([]any)
			if !ok || len(dnsNames) == 0 {
				t.Errorf("Certificate dnsNames is invalid")
				return
			}

			expectedDNS := "*.apps." + baseDomain
			if dnsNames[0] != expectedDNS {
				t.Errorf("Certificate dnsNames[0] = %v, want %s", dnsNames[0], expectedDNS)
			}
		})
	}
}

func TestEnsureClusterIssuer(t *testing.T) {
	scheme := runtime.NewScheme()
	ctx := context.Background()
	email := "test@example.com"

	tests := []struct {
		name       string
		existing   []client.Object
		wantErr    bool
		wantCreate bool
	}{
		{
			name:       "Creates ClusterIssuer when it doesn't exist",
			existing:   []client.Object{},
			wantErr:    false,
			wantCreate: true,
		},
		{
			name: "Does not create ClusterIssuer when it already exists",
			existing: []client.Object{
				&unstructured.Unstructured{
					Object: map[string]any{
						"apiVersion": "cert-manager.io/v1",
						"kind":       "ClusterIssuer",
						"metadata": map[string]any{
							"name": issuerName,
						},
						"spec": map[string]any{
							"acme": map[string]any{
								"email":   email,
								"server":  "https://acme-v02.api.letsencrypt.org/directory",
								"solvers": []any{},
							},
						},
					},
				},
			},
			wantErr:    false,
			wantCreate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.existing...).
				Build()

			err := ensureClusterIssuer(ctx, fakeClient, email)
			if (err != nil) != tt.wantErr {
				t.Errorf("ensureClusterIssuer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Verify ClusterIssuer was created or already exists
			obj := &unstructured.Unstructured{}
			obj.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "cert-manager.io",
				Version: "v1",
				Kind:    "ClusterIssuer",
			})

			err = fakeClient.Get(ctx, client.ObjectKey{
				Name: issuerName,
			}, obj)

			if err != nil {
				if !errors.IsNotFound(err) {
					t.Errorf("Failed to get ClusterIssuer: %v", err)
				} else if tt.wantCreate {
					t.Errorf("Expected ClusterIssuer to be created, but it doesn't exist")
				}
				return
			}

			// Only verify spec details when we expected creation
			if !tt.wantCreate {
				return
			}

			// Verify ClusterIssuer spec has ACME configuration
			spec, ok := obj.Object["spec"].(map[string]any)
			if !ok {
				t.Errorf("ClusterIssuer spec is not a map")
				return
			}

			acme, ok := spec["acme"].(map[string]any)
			if !ok {
				t.Errorf("ClusterIssuer spec.acme is not a map")
				return
			}

			// Verify email
			if acme["email"] != email {
				t.Errorf("ClusterIssuer email = %v, want %s", acme["email"], email)
			}

			// Verify acmeDNS solver
			solvers, ok := acme["solvers"].([]any)
			if !ok || len(solvers) == 0 {
				t.Errorf("ClusterIssuer solvers is invalid")
				return
			}

			solver, ok := solvers[0].(map[string]any)
			if !ok {
				t.Errorf("ClusterIssuer solver is not a map")
				return
			}

			dns01, ok := solver["dns01"].(map[string]any)
			if !ok {
				t.Errorf("ClusterIssuer solver dns01 is not a map")
				return
			}

		acmeDNS, ok := dns01["acmeDNS"].(map[string]any)
		if !ok {
			t.Errorf("ClusterIssuer solver dns01.acmeDNS is not a map")
			return
		}

		if acmeDNS["host"] != "http://acme-dns.kibaship.svc.cluster.local" {
			t.Errorf("ClusterIssuer acmeDNS host = %v, want 'http://acme-dns.kibaship.svc.cluster.local'", acmeDNS["host"])
		}

		accountSecretRef, ok := acmeDNS["accountSecretRef"].(map[string]any)
		if !ok {
			t.Errorf("ClusterIssuer acmeDNS accountSecretRef is not a map")
			return
		}

		if accountSecretRef["name"] != "acme-dns-account" {
			t.Errorf("ClusterIssuer accountSecretRef name = %v, want 'acme-dns-account'", accountSecretRef["name"])
		}

		if accountSecretRef["key"] != "acmedns.json" {
			t.Errorf("ClusterIssuer accountSecretRef key = %v, want 'acmedns.json'", accountSecretRef["key"])
		}
		})
	}
}

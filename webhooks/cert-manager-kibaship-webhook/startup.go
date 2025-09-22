package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

const (
	defaultValkeyName    = "kibaship-valkey-cluster-kibaship-com"
	defaultValkeyPort    = 6379
	saNamespaceFile      = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
)

// bootValkey waits for the Valkey CR to be ready, reads the password secret,
// and establishes a Cluster client connection using a seed address.
func (s *kibashipSolver) bootValkey(ctx context.Context, cfg *rest.Config) error {
	ns, err := detectNamespace()
	if err != nil {
		return fmt.Errorf("detect namespace: %w", err)
	}

	// 1) Wait for Valkey readiness via CR status.ready
	if err := waitForValkeyReady(ctx, cfg, ns, defaultValkeyName); err != nil {
		return err
	}

	// 2) Read password from Secret (single-key secret)
	password, err := readSingleValueSecret(ctx, s, ns, defaultValkeyName)
	if err != nil {
		return err
	}

	// 3) Connect to cluster using seed address
	seed := fmt.Sprintf("%s.%s.svc.cluster.local:%d", defaultValkeyName, ns, defaultValkeyPort)
	rc := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:        []string{seed},
		Password:     password,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})
	if err := rc.Ping(ctx).Err(); err != nil {
		_ = rc.Close()
		return fmt.Errorf("valkey ping failed: %w", err)
	}

	s.redis = rc
	fmt.Println("Valkey cluster connection established")
	return nil
}

func detectNamespace() (string, error) {
	b, err := ioutil.ReadFile(saNamespaceFile)
	if err == nil {
		ns := string(b)
		if ns != "" {
			return ns, nil
		}
	}
	if env := os.Getenv("POD_NAMESPACE"); env != "" {
		return env, nil
	}
	return "", errors.New("unable to determine namespace")
}

func waitForValkeyReady(ctx context.Context, cfg *rest.Config, ns, name string) error {
	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return fmt.Errorf("discovery client: %w", err)
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))
	gv, err := schema.ParseGroupVersion("hyperspike.io/v1")
	if err != nil { return err }
	m, err := mapper.RESTMapping(gv.WithKind("Valkey").GroupKind(), gv.Version)
	if err != nil { return fmt.Errorf("map Valkey GVK: %w", err) }

	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil { return fmt.Errorf("dynamic client: %w", err) }
	res := dyn.Resource(m.Resource).Namespace(ns)

	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	tick := time.NewTicker(20 * time.Second)
	defer tick.Stop()

	check := func() (bool, error) {
		obj, err := res.Get(ctxTimeout, name, metav1.GetOptions{})
		if err != nil { return false, nil } // treat errors as not ready yet
		ready, found, _ := unstructured.NestedBool(obj.Object, "status", "ready")
		return found && ready, nil
	}

	if ok, _ := check(); ok { return nil }
	for {
		select {
		case <-ctxTimeout.Done():
			return fmt.Errorf("timeout waiting for Valkey %s/%s ready", ns, name)
		case <-tick.C:
			if ok, _ := check(); ok { return nil }
		}
	}
}


// readSingleValueSecret fetches secret name in ns and returns its single data value as string.
func readSingleValueSecret(ctx context.Context, s *kibashipSolver, ns, name string) (string, error) {
	sec, err := s.kube.CoreV1().Secrets(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil { return "", fmt.Errorf("get secret %s/%s: %w", ns, name, err) }
	if len(sec.Data) != 1 { return "", fmt.Errorf("expected single field in secret %s/%s", ns, name) }
	for _, b := range sec.Data { return string(b), nil }
	return "", fmt.Errorf("secret %s/%s empty", ns, name)
}


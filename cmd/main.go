/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	platformv1alpha1 "github.com/kibamail/kibaship-operator/api/v1alpha1"
	"github.com/kibamail/kibaship-operator/internal/controller"
	"github.com/kibamail/kibaship-operator/pkg/streaming"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(platformv1alpha1.AddToScheme(scheme))
	utilruntime.Must(tektonv1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

// nolint:gocyclo
func main() {
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "d3e53d55.operator.kibaship.com",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	// Provision system Valkey cluster
	valkeyProvisioner := controller.NewValkeyProvisioner(mgr.GetClient())
	if err := valkeyProvisioner.ProvisionSystemValkeyCluster(context.Background()); err != nil {
		setupLog.Error(err, "Failed to provision system Valkey cluster")
		os.Exit(1)
	}

	// Get the operator namespace (assume same namespace as manager)
	operatorNamespace := os.Getenv("NAMESPACE")
	if operatorNamespace == "" {
		// Fallback to reading from service account if NAMESPACE env var not set
		operatorNamespace = "kibaship-operator" // Default namespace
	}

	// Create streaming configuration
	streamingConfig := streaming.DefaultConfig(operatorNamespace)

	// Create uncached Kubernetes client for startup initialization
	// This is needed because mgr.GetClient() requires the manager cache to be started
	uncachedClient, err := client.New(mgr.GetConfig(), client.Options{Scheme: mgr.GetScheme()})
	if err != nil {
		setupLog.Error(err, "Failed to create uncached client for streaming initialization")
		os.Exit(1)
	}
	k8sClient := streaming.NewKubernetesClientAdapter(uncachedClient)

	// BLOCKING STEP: Wait for Valkey cluster to become ready using simple polling
	setupLog.Info("Waiting for Valkey cluster to become ready - this will block until ready or timeout",
		"timeout", "5m", "checkInterval", "20s", "resource", streamingConfig.ValkeyServiceName)

	valkeyGate := streaming.NewValkeyReadyGate(k8sClient, streamingConfig)
	if err := valkeyGate.WaitForReady(context.Background()); err != nil {
		setupLog.Error(err, "Valkey cluster failed to become ready within timeout - crashing pod")
		os.Exit(1)
	}

	setupLog.Info("✅ Valkey cluster is ready - proceeding with streaming initialization")

	// Initialize streaming components now that Valkey is ready
	timeProvider := streaming.NewRealTimeProvider()
	secretManager := streaming.NewSecretManager(k8sClient, streamingConfig)
	connectionManager := streaming.NewConnectionManager(streamingConfig)

	// Initialize cluster connection with auto-discovery
	setupLog.Info("Establishing cluster connection to Valkey with auto-discovery")
	password, err := secretManager.GetValkeyPassword(context.Background())
	if err != nil {
		setupLog.Error(err, "Failed to get Valkey password")
		os.Exit(1)
	}

	// Build seed address for cluster discovery
	seedAddress := fmt.Sprintf("%s.%s.svc.cluster.local",
		streamingConfig.ValkeyServiceName,
		streamingConfig.Namespace)

	if err := connectionManager.InitializeCluster(context.Background(), seedAddress, password); err != nil {
		setupLog.Error(err, "Failed to initialize Valkey cluster connection")
		os.Exit(1)
	}

	setupLog.Info("✅ Valkey streaming initialized successfully")

	// Create streaming publisher for controllers
	streamPublisher := streaming.NewProjectStreamPublisher(
		connectionManager,
		timeProvider,
		streamingConfig,
	)

	// Now set up controllers with streaming publisher
	if err := controller.NewProjectReconciler(
		mgr.GetClient(),
		mgr.GetScheme(),
		streamPublisher,
	).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Project")
		os.Exit(1)
	}
	if err := (&platformv1alpha1.Project{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Project")
		os.Exit(1)
	}
	// Register Application webhook
	if err := (&platformv1alpha1.Application{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Application")
		os.Exit(1)
	}
	// Register Deployment webhook
	if err := (&platformv1alpha1.Deployment{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Deployment")
		os.Exit(1)
	}
	if err := (&controller.ApplicationReconciler{
		Client:          mgr.GetClient(),
		Scheme:          mgr.GetScheme(),
		StreamPublisher: streamPublisher,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Application")
		os.Exit(1)
	}
	if err := (&controller.DeploymentReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		NamespaceManager: controller.NewNamespaceManager(mgr.GetClient()),
		StreamPublisher:  streamPublisher,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Deployment")
		os.Exit(1)
	}
	if err := (&controller.ApplicationDomainReconciler{
		Client:          mgr.GetClient(),
		Scheme:          mgr.GetScheme(),
		StreamPublisher: streamPublisher,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ApplicationDomain")
		os.Exit(1)
	}
	if err := (&platformv1alpha1.ApplicationDomain{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "ApplicationDomain")
		os.Exit(1)
	}

	setupLog.Info("All controllers initialized with streaming publisher")

	// Set up graceful shutdown for streaming components
	ctx := ctrl.SetupSignalHandler()
	defer func() {
		setupLog.Info("Shutting down Valkey streaming components")
		if err := connectionManager.Close(); err != nil {
			setupLog.Error(err, "Error during streaming shutdown")
		}
	}()

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

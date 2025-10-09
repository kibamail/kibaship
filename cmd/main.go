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
	"crypto/rand"
	"flag"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	platformv1alpha1 "github.com/kibamail/kibaship-operator/api/v1alpha1"
	"github.com/kibamail/kibaship-operator/internal/bootstrap"
	"github.com/kibamail/kibaship-operator/internal/controller"
	"github.com/kibamail/kibaship-operator/pkg/config"
	"github.com/kibamail/kibaship-operator/pkg/webhooks"
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

	// Create uncached Kubernetes client for bootstrap operations
	uncachedClient, err := client.New(mgr.GetConfig(), client.Options{Scheme: mgr.GetScheme()})
	if err != nil {
		setupLog.Error(err, "Failed to create uncached client")
		os.Exit(1)
	}

	// Load operator configuration from ConfigMap
	setupLog.Info("Loading operator configuration from ConfigMap")
	opConfig, err := config.LoadConfigFromConfigMap(context.Background(), mgr.GetConfig())
	if err != nil {
		setupLog.Error(err, "failed to load operator configuration from ConfigMap")
		os.Exit(1)
	}
	setupLog.Info("Operator configuration loaded successfully",
		"domain", opConfig.Domain,
		"webhookURL", opConfig.WebhookURL,
		"acmeEmail", opConfig.ACMEEmail)

	// Set the global operator configuration
	if err := controller.SetOperatorConfig(opConfig.Domain); err != nil {
		setupLog.Error(err, "failed to set operator configuration")
		os.Exit(1)
	}

	// Bootstrap: ensure storage classes first, then provision dynamic ingress/cert-manager resources
	setupLog.Info("Starting bootstrap process")
	setupLog.Info("Bootstrap step 1: Ensuring storage classes")
	if err := bootstrap.EnsureStorageClasses(context.Background(), uncachedClient); err != nil {
		setupLog.Error(err, "bootstrap storage classes failed (continuing)")
	} else {
		setupLog.Info("Bootstrap step 1: Storage classes completed successfully")
	}

	acmeEmail := opConfig.ACMEEmail
	baseDomain := opConfig.Domain
	setupLog.Info("Bootstrap step 2: Provisioning ingress and certificates", "domain", baseDomain, "acmeEmail", acmeEmail)
	if err := bootstrap.ProvisionIngressAndCertificates(
		context.Background(),
		uncachedClient,
		baseDomain,
		acmeEmail,
	); err != nil {
		setupLog.Error(err, "bootstrap provisioning failed (continuing)")
	} else {
		setupLog.Info("Bootstrap step 2: Ingress and certificates completed successfully")
	}

	// Bootstrap: ensure registry credentials are provisioned
	setupLog.Info("Bootstrap step 3: Ensuring registry credentials")
	if err := bootstrap.EnsureRegistryCredentials(context.Background(), uncachedClient); err != nil {
		setupLog.Error(err, "bootstrap registry credentials failed (continuing)")
	} else {
		setupLog.Info("Bootstrap step 3: Registry credentials completed successfully")
	}

	// Bootstrap: ensure registry JWKS secret is provisioned
	setupLog.Info("Bootstrap step 4: Ensuring registry JWKS secret")
	if err := bootstrap.EnsureRegistryJWKS(context.Background(), uncachedClient); err != nil {
		setupLog.Error(err, "bootstrap registry JWKS failed (continuing)")
	} else {
		setupLog.Info("Bootstrap step 4: Registry JWKS completed successfully")
	}

	// Bootstrap: copy registry CA certificate to buildkit namespace
	setupLog.Info("Bootstrap step 5: Ensuring registry CA certificate in buildkit namespace")
	if err := bootstrap.EnsureRegistryCACertificateInBuildkit(context.Background(), uncachedClient); err != nil {
		setupLog.Error(err, "bootstrap registry CA certificate in buildkit failed (continuing)")
	} else {
		setupLog.Info("Bootstrap step 5: Registry CA certificate in buildkit completed successfully")
	}

	setupLog.Info("Bootstrap process completed")

	// Webhook configuration: ensure signing Secret exists
	webhookURL := opConfig.WebhookURL
	kcs, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		setupLog.Error(err, "failed to build clientset")
		os.Exit(1)
	}
	var signingKey []byte
	secret, err := kcs.CoreV1().Secrets(config.OperatorNamespace).Get(
		context.Background(),
		config.WebhookSecretName,
		metav1.GetOptions{},
	)
	if apierrors.IsNotFound(err) {
		buf := make([]byte, 32)
		if _, err := rand.Read(buf); err != nil {
			setupLog.Error(err, "failed to generate webhook signing key")
			os.Exit(1)
		}
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      config.WebhookSecretName,
				Namespace: config.OperatorNamespace,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{config.WebhookSecretKey: buf},
		}
		if _, err := kcs.CoreV1().Secrets(config.OperatorNamespace).Create(
			context.Background(),
			secret,
			metav1.CreateOptions{},
		); err != nil {
			setupLog.Error(err, "failed to create webhook signing secret")
			os.Exit(1)
		}
		signingKey = buf
	} else if err != nil {
		setupLog.Error(err, "failed to read webhook signing secret")
		os.Exit(1)
	} else {
		b, ok := secret.Data[config.WebhookSecretKey]
		if !ok || len(b) == 0 {
			buf := make([]byte, 32)
			if _, err := rand.Read(buf); err != nil {
				setupLog.Error(err, "failed to generate webhook signing key")
				os.Exit(1)
			}
			if secret.Data == nil {
				secret.Data = map[string][]byte{}
			}
			secret.Data[config.WebhookSecretKey] = buf
			if _, err := kcs.CoreV1().Secrets(config.OperatorNamespace).Update(
				context.Background(),
				secret,
				metav1.UpdateOptions{},
			); err != nil {
				setupLog.Error(err, "failed to update webhook signing secret")
				os.Exit(1)
			}
			signingKey = buf
		} else {
			signingKey = b
		}
	}

	// Build notifier (inject cache-backed reader for enrichment)
	n := webhooks.NewHTTPNotifier(webhookURL, signingKey, mgr.GetClient())

	// Now set up controllers
	if err := (&controller.ProjectReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		NamespaceManager: controller.NewNamespaceManager(mgr.GetClient()),
		Validator:        controller.NewProjectValidator(mgr.GetClient()),
		Notifier:         n,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Project")
		os.Exit(1)
	}
	if err := (&platformv1alpha1.Project{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Project")
		os.Exit(1)
	}
	// Register Environment controller
	if err := (&controller.EnvironmentReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Notifier: n,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Environment")
		os.Exit(1)
	}
	// Register Environment webhook
	if err := (&platformv1alpha1.Environment{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Environment")
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
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Notifier: n,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Application")
		os.Exit(1)
	}
	if err := (&controller.DeploymentReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		NamespaceManager: controller.NewNamespaceManager(mgr.GetClient()),
		Notifier:         n,
		Recorder:         mgr.GetEventRecorderFor("deployment-controller"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Deployment")
		os.Exit(1)
	}
	if err := (&controller.ApplicationDomainReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Notifier: n,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ApplicationDomain")
		os.Exit(1)
	}
	// Watch cert-manager Certificates and mirror status to ApplicationDomains
	if err := (&controller.CertificateWatcherReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Notifier: n,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CertificateWatcher")
		os.Exit(1)
	}
	// Watch Tekton PipelineRuns and mirror status to Deployments
	if err := (&controller.PipelineRunWatcherReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Notifier: n,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "PipelineRunWatcher")
		os.Exit(1)
	}

	if err := (&platformv1alpha1.ApplicationDomain{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "ApplicationDomain")
		os.Exit(1)
	}

	setupLog.Info("All controllers initialized")

	ctx := ctrl.SetupSignalHandler()

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

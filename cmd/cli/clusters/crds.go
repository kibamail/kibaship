package clusters

// CRDFile represents a CRD file with name and URL
type CRDFile struct {
	Name string
	URL  string
}

const (
	// Base URLs for CRD files
	gatewayBaseURL = "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/gateway/v1.3.0/"
	tektonBaseURL  = "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/"
	valkeyBaseURL  = "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/valkey/v0.0.59/"
)

// GatewayCRDFiles contains the custom Gateway API CRDs v1.3.0
var GatewayCRDFiles = []CRDFile{
	{
		Name: "01-backendtlspolicies",
		URL:  gatewayBaseURL + "01-backendtlspolicies.gateway.networking.k8s.io-custom-resource-definition.yaml",
	},
	{
		Name: "02-gatewayclasses",
		URL:  gatewayBaseURL + "02-gatewayclasses.gateway.networking.k8s.io-custom-resource-definition.yaml",
	},
	{
		Name: "03-gateways",
		URL:  gatewayBaseURL + "03-gateways.gateway.networking.k8s.io-custom-resource-definition.yaml",
	},
	{
		Name: "04-grpcroutes",
		URL:  gatewayBaseURL + "04-grpcroutes.gateway.networking.k8s.io-custom-resource-definition.yaml",
	},
	{
		Name: "05-httproutes",
		URL:  gatewayBaseURL + "05-httproutes.gateway.networking.k8s.io-custom-resource-definition.yaml",
	},
	{
		Name: "06-referencegrants",
		URL:  gatewayBaseURL + "06-referencegrants.gateway.networking.k8s.io-custom-resource-definition.yaml",
	},
	{
		Name: "07-tcproutes",
		URL:  gatewayBaseURL + "07-tcproutes.gateway.networking.k8s.io-custom-resource-definition.yaml",
	},
	{
		Name: "08-tlsroutes",
		URL:  gatewayBaseURL + "08-tlsroutes.gateway.networking.k8s.io-custom-resource-definition.yaml",
	},
	{
		Name: "09-udproutes",
		URL:  gatewayBaseURL + "09-udproutes.gateway.networking.k8s.io-custom-resource-definition.yaml",
	},
}

// GatewayCRDNames contains the names of Gateway API CRDs for waiting
var GatewayCRDNames = []string{
	"backendtlspolicies.gateway.networking.k8s.io",
	"gatewayclasses.gateway.networking.k8s.io",
	"gateways.gateway.networking.k8s.io",
	"grpcroutes.gateway.networking.k8s.io",
	"httproutes.gateway.networking.k8s.io",
	"referencegrants.gateway.networking.k8s.io",
	"tcproutes.gateway.networking.k8s.io",
	"tlsroutes.gateway.networking.k8s.io",
	"udproutes.gateway.networking.k8s.io",
}

// TektonCRDFiles contains the Tekton Pipelines CRDs v1.4.0
var TektonCRDFiles = []CRDFile{
	{
		Name: "01_tekton_pipelines_namespace",
		URL:  tektonBaseURL + "01_tekton_pipelines_namespace.yaml",
	},
	{
		Name: "02_tekton_pipelines_controller_cluster_access_cluster_role",
		URL:  tektonBaseURL + "02_tekton_pipelines_controller_cluster_access_cluster_role.yaml",
	},
	{
		Name: "03_tekton_pipelines_controller_tenant_access_cluster_role",
		URL:  tektonBaseURL + "03_tekton_pipelines_controller_tenant_access_cluster_role.yaml",
	},
	{
		Name: "04_tekton_pipelines_webhook_cluster_access_cluster_role",
		URL:  tektonBaseURL + "04_tekton_pipelines_webhook_cluster_access_cluster_role.yaml",
	},
	{
		Name: "05_tekton_events_controller_cluster_access_cluster_role",
		URL:  tektonBaseURL + "05_tekton_events_controller_cluster_access_cluster_role.yaml",
	},
	{
		Name: "06_tekton_pipelines_controller_role",
		URL:  tektonBaseURL + "06_tekton_pipelines_controller_role.yaml",
	},
	{
		Name: "07_tekton_pipelines_webhook_role",
		URL:  tektonBaseURL + "07_tekton_pipelines_webhook_role.yaml",
	},
	{
		Name: "08_tekton_pipelines_events_controller_role",
		URL:  tektonBaseURL + "08_tekton_pipelines_events_controller_role.yaml",
	},
	{
		Name: "09_tekton_pipelines_leader_election_role",
		URL:  tektonBaseURL + "09_tekton_pipelines_leader_election_role.yaml",
	},
	{
		Name: "10_tekton_pipelines_info_role",
		URL:  tektonBaseURL + "10_tekton_pipelines_info_role.yaml",
	},
	{
		Name: "11_tekton_pipelines_controller_service_account",
		URL:  tektonBaseURL + "11_tekton_pipelines_controller_service_account.yaml",
	},
	{
		Name: "12_tekton_pipelines_webhook_service_account",
		URL:  tektonBaseURL + "12_tekton_pipelines_webhook_service_account.yaml",
	},
	{
		Name: "13_tekton_events_controller_service_account",
		URL:  tektonBaseURL + "13_tekton_events_controller_service_account.yaml",
	},
	{
		Name: "14_tekton_pipelines_controller_cluster_access_cluster_role_binding",
		URL:  tektonBaseURL + "14_tekton_pipelines_controller_cluster_access_cluster_role_binding.yaml",
	},
	{
		Name: "15_tekton_pipelines_controller_tenant_access_cluster_role_binding",
		URL:  tektonBaseURL + "15_tekton_pipelines_controller_tenant_access_cluster_role_binding.yaml",
	},
	{
		Name: "16_tekton_pipelines_webhook_cluster_access_cluster_role_binding",
		URL:  tektonBaseURL + "16_tekton_pipelines_webhook_cluster_access_cluster_role_binding.yaml",
	},
	{
		Name: "17_tekton_events_controller_cluster_access_cluster_role_binding",
		URL:  tektonBaseURL + "17_tekton_events_controller_cluster_access_cluster_role_binding.yaml",
	},
	{
		Name: "18_tekton_pipelines_controller_role_binding",
		URL:  tektonBaseURL + "18_tekton_pipelines_controller_role_binding.yaml",
	},
	{
		Name: "19_tekton_pipelines_webhook_role_binding",
		URL:  tektonBaseURL + "19_tekton_pipelines_webhook_role_binding.yaml",
	},
	{
		Name: "20_tekton_pipelines_controller_leaderelection_role_binding",
		URL:  tektonBaseURL + "20_tekton_pipelines_controller_leaderelection_role_binding.yaml",
	},
	{
		Name: "21_tekton_pipelines_webhook_leaderelection_role_binding",
		URL:  tektonBaseURL + "21_tekton_pipelines_webhook_leaderelection_role_binding.yaml",
	},
	{
		Name: "22_tekton_pipelines_info_role_binding",
		URL:  tektonBaseURL + "22_tekton_pipelines_info_role_binding.yaml",
	},
	{
		Name: "23_tekton_pipelines_events_controller_role_binding",
		URL:  tektonBaseURL + "23_tekton_pipelines_events_controller_role_binding.yaml",
	},
	{
		Name: "24_tekton_events_controller_leaderelection_role_binding",
		URL:  tektonBaseURL + "24_tekton_events_controller_leaderelection_role_binding.yaml",
	},
	{
		Name: "25_customruns_tekton_dev_custom_resource_definition",
		URL:  tektonBaseURL + "25_customruns_tekton_dev_custom_resource_definition.yaml",
	},
	{
		Name: "26_pipelines_tekton_dev_custom_resource_definition",
		URL:  tektonBaseURL + "26_pipelines_tekton_dev_custom_resource_definition.yaml",
	},
	{
		Name: "27_pipelineruns_tekton_dev_custom_resource_definition",
		URL:  tektonBaseURL + "27_pipelineruns_tekton_dev_custom_resource_definition.yaml",
	},
	{
		Name: "28_resolutionrequests_resolution_tekton_dev_custom_resource_definition",
		URL:  tektonBaseURL + "28_resolutionrequests_resolution_tekton_dev_custom_resource_definition.yaml",
	},
	{
		Name: "29_stepactions_tekton_dev_custom_resource_definition",
		URL:  tektonBaseURL + "29_stepactions_tekton_dev_custom_resource_definition.yaml",
	},
	{
		Name: "30_tasks_tekton_dev_custom_resource_definition",
		URL:  tektonBaseURL + "30_tasks_tekton_dev_custom_resource_definition.yaml",
	},
	{
		Name: "31_taskruns_tekton_dev_custom_resource_definition",
		URL:  tektonBaseURL + "31_taskruns_tekton_dev_custom_resource_definition.yaml",
	},
	{
		Name: "32_verificationpolicies_tekton_dev_custom_resource_definition",
		URL:  tektonBaseURL + "32_verificationpolicies_tekton_dev_custom_resource_definition.yaml",
	},
	{
		Name: "33_webhook_certs_secret",
		URL:  tektonBaseURL + "33_webhook_certs_secret.yaml",
	},
	{
		Name: "34_validation_webhook_pipeline_tekton_dev_validating_webhook_configuration",
		URL:  tektonBaseURL + "34_validation_webhook_pipeline_tekton_dev_validating_webhook_configuration.yaml",
	},
	{
		Name: "35_webhook_pipeline_tekton_dev_mutating_webhook_configuration",
		URL:  tektonBaseURL + "35_webhook_pipeline_tekton_dev_mutating_webhook_configuration.yaml",
	},
	// Continue with remaining Tekton files (36-75)
	{
		Name: "36_config_webhook_pipeline_tekton_dev_validating_webhook_configuration",
		URL:  tektonBaseURL + "36_config_webhook_pipeline_tekton_dev_validating_webhook_configuration.yaml",
	},
	{
		Name: "37_tekton_aggregate_edit_cluster_role",
		URL:  tektonBaseURL + "37_tekton_aggregate_edit_cluster_role.yaml",
	},
	{
		Name: "38_tekton_aggregate_view_cluster_role",
		URL:  tektonBaseURL + "38_tekton_aggregate_view_cluster_role.yaml",
	},
	{
		Name: "39_config_defaults_config_map",
		URL:  tektonBaseURL + "39_config_defaults_config_map.yaml",
	},
	{
		Name: "40_config_events_config_map",
		URL:  tektonBaseURL + "40_config_events_config_map.yaml",
	},
	{
		Name: "41_feature_flags_config_map",
		URL:  tektonBaseURL + "41_feature_flags_config_map.yaml",
	},
	{
		Name: "42_pipelines_info_config_map",
		URL:  tektonBaseURL + "42_pipelines_info_config_map.yaml",
	},
	{
		Name: "43_config_leader_election_controller_config_map",
		URL:  tektonBaseURL + "43_config_leader_election_controller_config_map.yaml",
	},
	{
		Name: "44_config_leader_election_events_config_map",
		URL:  tektonBaseURL + "44_config_leader_election_events_config_map.yaml",
	},
	{
		Name: "45_config_leader_election_webhook_config_map",
		URL:  tektonBaseURL + "45_config_leader_election_webhook_config_map.yaml",
	},
	{
		Name: "46_config_logging_config_map",
		URL:  tektonBaseURL + "46_config_logging_config_map.yaml",
	},
	{
		Name: "47_config_observability_config_map",
		URL:  tektonBaseURL + "47_config_observability_config_map.yaml",
	},
	{
		Name: "48_config_registry_cert_config_map",
		URL:  tektonBaseURL + "48_config_registry_cert_config_map.yaml",
	},
	{
		Name: "49_config_spire_config_map",
		URL:  tektonBaseURL + "49_config_spire_config_map.yaml",
	},
	{
		Name: "50_config_tracing_config_map",
		URL:  tektonBaseURL + "50_config_tracing_config_map.yaml",
	},
	{
		Name: "51_config_wait_exponential_backoff_config_map",
		URL:  tektonBaseURL + "51_config_wait_exponential_backoff_config_map.yaml",
	},
	{
		Name: "52_tekton_pipelines_controller_deployment",
		URL:  tektonBaseURL + "52_tekton_pipelines_controller_deployment.yaml",
	},
	{
		Name: "53_tekton_pipelines_controller_service",
		URL:  tektonBaseURL + "53_tekton_pipelines_controller_service.yaml",
	},
	{
		Name: "54_tekton_events_controller_deployment",
		URL:  tektonBaseURL + "54_tekton_events_controller_deployment.yaml",
	},
	{
		Name: "55_tekton_events_controller_service",
		URL:  tektonBaseURL + "55_tekton_events_controller_service.yaml",
	},
	// Remaining Tekton files (56-75)
	{
		Name: "56_tekton_pipelines_resolvers_namespace",
		URL:  tektonBaseURL + "56_tekton_pipelines_resolvers_namespace.yaml",
	},
	{
		Name: "57_tekton_pipelines_resolvers_resolution_request_updates_cluster_role",
		URL:  tektonBaseURL + "57_tekton_pipelines_resolvers_resolution_request_updates_cluster_role.yaml",
	},
	{
		Name: "58_tekton_pipelines_resolvers_namespace_rbac_role",
		URL:  tektonBaseURL + "58_tekton_pipelines_resolvers_namespace_rbac_role.yaml",
	},
	{
		Name: "59_tekton_pipelines_resolvers_service_account",
		URL:  tektonBaseURL + "59_tekton_pipelines_resolvers_service_account.yaml",
	},
	{
		Name: "60_tekton_pipelines_resolvers_cluster_role_binding",
		URL:  tektonBaseURL + "60_tekton_pipelines_resolvers_cluster_role_binding.yaml",
	},
	{
		Name: "61_tekton_pipelines_resolvers_namespace_rbac_role_binding",
		URL:  tektonBaseURL + "61_tekton_pipelines_resolvers_namespace_rbac_role_binding.yaml",
	},
	{
		Name: "62_bundleresolver_config_config_map",
		URL:  tektonBaseURL + "62_bundleresolver_config_config_map.yaml",
	},
	{
		Name: "63_cluster_resolver_config_config_map",
		URL:  tektonBaseURL + "63_cluster_resolver_config_config_map.yaml",
	},
	{
		Name: "64_resolvers_feature_flags_config_map",
		URL:  tektonBaseURL + "64_resolvers_feature_flags_config_map.yaml",
	},
	{
		Name: "65_config_leader_election_resolvers_config_map",
		URL:  tektonBaseURL + "65_config_leader_election_resolvers_config_map.yaml",
	},
	{
		Name: "66_config_logging_config_map",
		URL:  tektonBaseURL + "66_config_logging_config_map.yaml",
	},
	{
		Name: "67_config_observability_config_map",
		URL:  tektonBaseURL + "67_config_observability_config_map.yaml",
	},
	{
		Name: "68_git_resolver_config_config_map",
		URL:  tektonBaseURL + "68_git_resolver_config_config_map.yaml",
	},
	{
		Name: "69_http_resolver_config_config_map",
		URL:  tektonBaseURL + "69_http_resolver_config_config_map.yaml",
	},
	{
		Name: "70_hubresolver_config_config_map",
		URL:  tektonBaseURL + "70_hubresolver_config_config_map.yaml",
	},
	{
		Name: "71_tekton_pipelines_remote_resolvers_deployment",
		URL:  tektonBaseURL + "71_tekton_pipelines_remote_resolvers_deployment.yaml",
	},
	{
		Name: "72_tekton_pipelines_remote_resolvers_service",
		URL:  tektonBaseURL + "72_tekton_pipelines_remote_resolvers_service.yaml",
	},
	{
		Name: "73_tekton_pipelines_webhook_horizontal_pod_autoscaler",
		URL:  tektonBaseURL + "73_tekton_pipelines_webhook_horizontal_pod_autoscaler.yaml",
	},
	{
		Name: "74_tekton_pipelines_webhook_deployment",
		URL:  tektonBaseURL + "74_tekton_pipelines_webhook_deployment.yaml",
	},
	{
		Name: "75_tekton_pipelines_webhook_service",
		URL:  tektonBaseURL + "75_tekton_pipelines_webhook_service.yaml",
	},
}

// TektonCRDNames contains the names of Tekton CRDs for waiting
var TektonCRDNames = []string{
	"customruns.tekton.dev",
	"pipelines.tekton.dev",
	"pipelineruns.tekton.dev",
	"resolutionrequests.resolution.tekton.dev",
	"stepactions.tekton.dev",
	"tasks.tekton.dev",
	"taskruns.tekton.dev",
	"verificationpolicies.tekton.dev",
}

// ValkeyCRDFiles contains the Valkey Operator CRDs v0.0.59
var ValkeyCRDFiles = []CRDFile{
	{
		Name: "01-valkey-operator-system-namespace",
		URL:  valkeyBaseURL + "01-valkey-operator-system-namespace.yaml",
	},
	{
		Name: "02-valkeys.hyperspike.io-custom-resource-definition",
		URL:  valkeyBaseURL + "02-valkeys.hyperspike.io-custom-resource-definition.yaml",
	},
	{
		Name: "03-valkey-operator-controller-manager-service-account",
		URL:  valkeyBaseURL + "03-valkey-operator-controller-manager-service-account.yaml",
	},
	{
		Name: "04-valkey-operator-leader-election-role-role",
		URL:  valkeyBaseURL + "04-valkey-operator-leader-election-role-role.yaml",
	},
	{
		Name: "05-valkey-operator-manager-role-cluster-role",
		URL:  valkeyBaseURL + "05-valkey-operator-manager-role-cluster-role.yaml",
	},
	{
		Name: "06-valkey-operator-valkey-editor-role-cluster-role",
		URL:  valkeyBaseURL + "06-valkey-operator-valkey-editor-role-cluster-role.yaml",
	},
	{
		Name: "07-valkey-operator-valkey-viewer-role-cluster-role",
		URL:  valkeyBaseURL + "07-valkey-operator-valkey-viewer-role-cluster-role.yaml",
	},
	{
		Name: "08-valkey-operator-leader-election-rolebinding-role-binding",
		URL:  valkeyBaseURL + "08-valkey-operator-leader-election-rolebinding-role-binding.yaml",
	},
	{
		Name: "09-valkey-operator-manager-rolebinding-cluster-role-binding",
		URL:  valkeyBaseURL + "09-valkey-operator-manager-rolebinding-cluster-role-binding.yaml",
	},
	{
		Name: "10-valkey-operator-config-config-map",
		URL:  valkeyBaseURL + "10-valkey-operator-config-config-map.yaml",
	},
	{
		Name: "11-valkey-operator-controller-manager-deployment",
		URL:  valkeyBaseURL + "11-valkey-operator-controller-manager-deployment.yaml",
	},
}

// ValkeyCRDNames contains the names of Valkey CRDs for waiting
var ValkeyCRDNames = []string{
	"valkeys.hyperspike.io",
}

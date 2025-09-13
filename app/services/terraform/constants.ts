export const GATEWAY_CRD_FILES = [
  {
    name: "01-backendtlspolicies",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/gateway/v1.3.0/01-backendtlspolicies.gateway.networking.k8s.io-custom-resource-definition.yaml"
  },
  {
    name: "02-gatewayclasses",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/gateway/v1.3.0/02-gatewayclasses.gateway.networking.k8s.io-custom-resource-definition.yaml"
  },
  {
    name: "03-gateways",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/gateway/v1.3.0/03-gateways.gateway.networking.k8s.io-custom-resource-definition.yaml"
  },
  {
    name: "04-grpcroutes",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/gateway/v1.3.0/04-grpcroutes.gateway.networking.k8s.io-custom-resource-definition.yaml"
  },
  {
    name: "05-httproutes",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/gateway/v1.3.0/05-httproutes.gateway.networking.k8s.io-custom-resource-definition.yaml"
  },
  {
    name: "06-referencegrants",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/gateway/v1.3.0/06-referencegrants.gateway.networking.k8s.io-custom-resource-definition.yaml"
  },
  {
    name: "07-tcproutes",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/gateway/v1.3.0/07-tcproutes.gateway.networking.k8s.io-custom-resource-definition.yaml"
  },
  {
    name: "08-tlsroutes",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/gateway/v1.3.0/08-tlsroutes.gateway.networking.k8s.io-custom-resource-definition.yaml"
  },
  {
    name: "09-udproutes",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/gateway/v1.3.0/09-udproutes.gateway.networking.k8s.io-custom-resource-definition.yaml"
  }
] as const

export const TEKTON_CRD_FILES = [
  {
    name: "01_tekton_pipelines_namespace",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/01_tekton_pipelines_namespace.yaml"
  },
  {
    name: "02_tekton_pipelines_controller_cluster_access_cluster_role",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/02_tekton_pipelines_controller_cluster_access_cluster_role.yaml"
  },
  {
    name: "03_tekton_pipelines_controller_tenant_access_cluster_role",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/03_tekton_pipelines_controller_tenant_access_cluster_role.yaml"
  },
  {
    name: "04_tekton_pipelines_webhook_cluster_access_cluster_role",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/04_tekton_pipelines_webhook_cluster_access_cluster_role.yaml"
  },
  {
    name: "05_tekton_events_controller_cluster_access_cluster_role",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/05_tekton_events_controller_cluster_access_cluster_role.yaml"
  },
  {
    name: "06_tekton_pipelines_controller_role",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/06_tekton_pipelines_controller_role.yaml"
  },
  {
    name: "07_tekton_pipelines_webhook_role",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/07_tekton_pipelines_webhook_role.yaml"
  },
  {
    name: "08_tekton_pipelines_events_controller_role",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/08_tekton_pipelines_events_controller_role.yaml"
  },
  {
    name: "09_tekton_pipelines_leader_election_role",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/09_tekton_pipelines_leader_election_role.yaml"
  },
  {
    name: "10_tekton_pipelines_info_role",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/10_tekton_pipelines_info_role.yaml"
  },
  {
    name: "11_tekton_pipelines_controller_service_account",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/11_tekton_pipelines_controller_service_account.yaml"
  },
  {
    name: "12_tekton_pipelines_webhook_service_account",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/12_tekton_pipelines_webhook_service_account.yaml"
  },
  {
    name: "13_tekton_events_controller_service_account",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/13_tekton_events_controller_service_account.yaml"
  },
  {
    name: "14_tekton_pipelines_controller_cluster_access_cluster_role_binding",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/14_tekton_pipelines_controller_cluster_access_cluster_role_binding.yaml"
  },
  {
    name: "15_tekton_pipelines_controller_tenant_access_cluster_role_binding",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/15_tekton_pipelines_controller_tenant_access_cluster_role_binding.yaml"
  },
  {
    name: "16_tekton_pipelines_webhook_cluster_access_cluster_role_binding",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/16_tekton_pipelines_webhook_cluster_access_cluster_role_binding.yaml"
  },
  {
    name: "17_tekton_events_controller_cluster_access_cluster_role_binding",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/17_tekton_events_controller_cluster_access_cluster_role_binding.yaml"
  },
  {
    name: "18_tekton_pipelines_controller_role_binding",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/18_tekton_pipelines_controller_role_binding.yaml"
  },
  {
    name: "19_tekton_pipelines_webhook_role_binding",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/19_tekton_pipelines_webhook_role_binding.yaml"
  },
  {
    name: "20_tekton_pipelines_controller_leaderelection_role_binding",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/20_tekton_pipelines_controller_leaderelection_role_binding.yaml"
  },
  {
    name: "21_tekton_pipelines_webhook_leaderelection_role_binding",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/21_tekton_pipelines_webhook_leaderelection_role_binding.yaml"
  },
  {
    name: "22_tekton_pipelines_info_role_binding",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/22_tekton_pipelines_info_role_binding.yaml"
  },
  {
    name: "23_tekton_pipelines_events_controller_role_binding",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/23_tekton_pipelines_events_controller_role_binding.yaml"
  },
  {
    name: "24_tekton_events_controller_leaderelection_role_binding",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/24_tekton_events_controller_leaderelection_role_binding.yaml"
  },
  {
    name: "25_customruns_tekton_dev_custom_resource_definition",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/25_customruns_tekton_dev_custom_resource_definition.yaml"
  },
  {
    name: "26_pipelines_tekton_dev_custom_resource_definition",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/26_pipelines_tekton_dev_custom_resource_definition.yaml"
  },
  {
    name: "27_pipelineruns_tekton_dev_custom_resource_definition",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/27_pipelineruns_tekton_dev_custom_resource_definition.yaml"
  },
  {
    name: "28_resolutionrequests_resolution_tekton_dev_custom_resource_definition",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/28_resolutionrequests_resolution_tekton_dev_custom_resource_definition.yaml"
  },
  {
    name: "29_stepactions_tekton_dev_custom_resource_definition",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/29_stepactions_tekton_dev_custom_resource_definition.yaml"
  },
  {
    name: "30_tasks_tekton_dev_custom_resource_definition",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/30_tasks_tekton_dev_custom_resource_definition.yaml"
  },
  {
    name: "31_taskruns_tekton_dev_custom_resource_definition",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/31_taskruns_tekton_dev_custom_resource_definition.yaml"
  },
  {
    name: "32_verificationpolicies_tekton_dev_custom_resource_definition",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/32_verificationpolicies_tekton_dev_custom_resource_definition.yaml"
  },
  {
    name: "33_webhook_certs_secret",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/33_webhook_certs_secret.yaml"
  },
  {
    name: "34_validation_webhook_pipeline_tekton_dev_validating_webhook_configuration",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/34_validation_webhook_pipeline_tekton_dev_validating_webhook_configuration.yaml"
  },
  {
    name: "35_webhook_pipeline_tekton_dev_mutating_webhook_configuration",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/35_webhook_pipeline_tekton_dev_mutating_webhook_configuration.yaml"
  },
  {
    name: "36_config_webhook_pipeline_tekton_dev_validating_webhook_configuration",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/36_config_webhook_pipeline_tekton_dev_validating_webhook_configuration.yaml"
  },
  {
    name: "37_tekton_aggregate_edit_cluster_role",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/37_tekton_aggregate_edit_cluster_role.yaml"
  },
  {
    name: "38_tekton_aggregate_view_cluster_role",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/38_tekton_aggregate_view_cluster_role.yaml"
  },
  {
    name: "39_config_defaults_config_map",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/39_config_defaults_config_map.yaml"
  },
  {
    name: "40_config_events_config_map",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/40_config_events_config_map.yaml"
  },
  {
    name: "41_feature_flags_config_map",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/41_feature_flags_config_map.yaml"
  },
  {
    name: "42_pipelines_info_config_map",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/42_pipelines_info_config_map.yaml"
  },
  {
    name: "43_config_leader_election_controller_config_map",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/43_config_leader_election_controller_config_map.yaml"
  },
  {
    name: "44_config_leader_election_events_config_map",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/44_config_leader_election_events_config_map.yaml"
  },
  {
    name: "45_config_leader_election_webhook_config_map",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/45_config_leader_election_webhook_config_map.yaml"
  },
  {
    name: "46_config_logging_config_map",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/46_config_logging_config_map.yaml"
  },
  {
    name: "47_config_observability_config_map",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/47_config_observability_config_map.yaml"
  },
  {
    name: "48_config_registry_cert_config_map",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/48_config_registry_cert_config_map.yaml"
  },
  {
    name: "49_config_spire_config_map",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/49_config_spire_config_map.yaml"
  },
  {
    name: "50_config_tracing_config_map",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/50_config_tracing_config_map.yaml"
  },
  {
    name: "51_config_wait_exponential_backoff_config_map",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/51_config_wait_exponential_backoff_config_map.yaml"
  },
  {
    name: "52_tekton_pipelines_controller_deployment",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/52_tekton_pipelines_controller_deployment.yaml"
  },
  {
    name: "53_tekton_pipelines_controller_service",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/53_tekton_pipelines_controller_service.yaml"
  },
  {
    name: "54_tekton_events_controller_deployment",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/54_tekton_events_controller_deployment.yaml"
  },
  {
    name: "55_tekton_events_controller_service",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/55_tekton_events_controller_service.yaml"
  },
  {
    name: "56_tekton_pipelines_resolvers_namespace",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/56_tekton_pipelines_resolvers_namespace.yaml"
  },
  {
    name: "57_tekton_pipelines_resolvers_resolution_request_updates_cluster_role",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/57_tekton_pipelines_resolvers_resolution_request_updates_cluster_role.yaml"
  },
  {
    name: "58_tekton_pipelines_resolvers_namespace_rbac_role",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/58_tekton_pipelines_resolvers_namespace_rbac_role.yaml"
  },
  {
    name: "59_tekton_pipelines_resolvers_service_account",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/59_tekton_pipelines_resolvers_service_account.yaml"
  },
  {
    name: "60_tekton_pipelines_resolvers_cluster_role_binding",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/60_tekton_pipelines_resolvers_cluster_role_binding.yaml"
  },
  {
    name: "61_tekton_pipelines_resolvers_namespace_rbac_role_binding",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/61_tekton_pipelines_resolvers_namespace_rbac_role_binding.yaml"
  },
  {
    name: "62_bundleresolver_config_config_map",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/62_bundleresolver_config_config_map.yaml"
  },
  {
    name: "63_cluster_resolver_config_config_map",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/63_cluster_resolver_config_config_map.yaml"
  },
  {
    name: "64_resolvers_feature_flags_config_map",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/64_resolvers_feature_flags_config_map.yaml"
  },
  {
    name: "65_config_leader_election_resolvers_config_map",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/65_config_leader_election_resolvers_config_map.yaml"
  },
  {
    name: "66_config_logging_config_map",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/66_config_logging_config_map.yaml"
  },
  {
    name: "67_config_observability_config_map",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/67_config_observability_config_map.yaml"
  },
  {
    name: "68_git_resolver_config_config_map",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/68_git_resolver_config_config_map.yaml"
  },
  {
    name: "69_http_resolver_config_config_map",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/69_http_resolver_config_config_map.yaml"
  },
  {
    name: "70_hubresolver_config_config_map",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/70_hubresolver_config_config_map.yaml"
  },
  {
    name: "71_tekton_pipelines_remote_resolvers_deployment",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/71_tekton_pipelines_remote_resolvers_deployment.yaml"
  },
  {
    name: "72_tekton_pipelines_remote_resolvers_service",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/72_tekton_pipelines_remote_resolvers_service.yaml"
  },
  {
    name: "73_tekton_pipelines_webhook_horizontal_pod_autoscaler",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/73_tekton_pipelines_webhook_horizontal_pod_autoscaler.yaml"
  },
  {
    name: "74_tekton_pipelines_webhook_deployment",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/74_tekton_pipelines_webhook_deployment.yaml"
  },
  {
    name: "75_tekton_pipelines_webhook_service",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.4.0/75_tekton_pipelines_webhook_service.yaml"
  }
] as const

export const VALKEY_CRD_FILES = [
  {
    name: "01-valkey-operator-system-namespace",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/valkey/v0.0.59/01-valkey-operator-system-namespace.yaml"
  },
  {
    name: "02-valkeys.hyperspike.io-custom-resource-definition",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/valkey/v0.0.59/02-valkeys.hyperspike.io-custom-resource-definition.yaml"
  },
  {
    name: "03-valkey-operator-controller-manager-service-account",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/valkey/v0.0.59/03-valkey-operator-controller-manager-service-account.yaml"
  },
  {
    name: "04-valkey-operator-leader-election-role-role",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/valkey/v0.0.59/04-valkey-operator-leader-election-role-role.yaml"
  },
  {
    name: "05-valkey-operator-manager-role-cluster-role",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/valkey/v0.0.59/05-valkey-operator-manager-role-cluster-role.yaml"
  },
  {
    name: "06-valkey-operator-valkey-editor-role-cluster-role",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/valkey/v0.0.59/06-valkey-operator-valkey-editor-role-cluster-role.yaml"
  },
  {
    name: "07-valkey-operator-valkey-viewer-role-cluster-role",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/valkey/v0.0.59/07-valkey-operator-valkey-viewer-role-cluster-role.yaml"
  },
  {
    name: "08-valkey-operator-leader-election-rolebinding-role-binding",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/valkey/v0.0.59/08-valkey-operator-leader-election-rolebinding-role-binding.yaml"
  },
  {
    name: "09-valkey-operator-manager-rolebinding-cluster-role-binding",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/valkey/v0.0.59/09-valkey-operator-manager-rolebinding-cluster-role-binding.yaml"
  },
  {
    name: "10-valkey-operator-config-config-map",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/valkey/v0.0.59/10-valkey-operator-config-config-map.yaml"
  },
  {
    name: "11-valkey-operator-controller-manager-deployment",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/valkey/v0.0.59/11-valkey-operator-controller-manager-deployment.yaml"
  }
] as const

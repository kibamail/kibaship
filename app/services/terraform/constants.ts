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

export const LINSTOR_CRD_FILES = [
  {
    name: "01_piraeus_datastore_namespace",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/01_piraeus_datastore_namespace.yaml"
  },
  {
    name: "02_linstorclusters_piraeus_io_custom_resource_definition",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/02_linstorclusters_piraeus_io_custom_resource_definition.yaml"
  },
  {
    name: "03_linstornodeconnections_piraeus_io_custom_resource_definition",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/03_linstornodeconnections_piraeus_io_custom_resource_definition.yaml"
  },
  {
    name: "04_linstorsatelliteconfigurations_piraeus_io_custom_resource_definition",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/04_linstorsatelliteconfigurations_piraeus_io_custom_resource_definition.yaml"
  },
  {
    name: "05_linstorsatellites_piraeus_io_custom_resource_definition",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/05_linstorsatellites_piraeus_io_custom_resource_definition.yaml"
  },
  {
    name: "06_piraeus_operator_controller_manager_service_account",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/06_piraeus_operator_controller_manager_service_account.yaml"
  },
  {
    name: "07_piraeus_operator_gencert_service_account",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/07_piraeus_operator_gencert_service_account.yaml"
  },
  {
    name: "08_piraeus_operator_gencert_role",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/08_piraeus_operator_gencert_role.yaml"
  },
  {
    name: "09_piraeus_operator_leader_election_role_role",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/09_piraeus_operator_leader_election_role_role.yaml"
  },
  {
    name: "10_piraeus_operator_controller_manager_cluster_role",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/10_piraeus_operator_controller_manager_cluster_role.yaml"
  },
  {
    name: "11_piraeus_operator_gencert_cluster_role",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/11_piraeus_operator_gencert_cluster_role.yaml"
  },
  {
    name: "12_piraeus_operator_gencert_role_binding",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/12_piraeus_operator_gencert_role_binding.yaml"
  },
  {
    name: "13_piraeus_operator_leader_election_rolebinding_role_binding",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/13_piraeus_operator_leader_election_rolebinding_role_binding.yaml"
  },
  {
    name: "14_piraeus_operator_gencert_cluster_role_binding",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/14_piraeus_operator_gencert_cluster_role_binding.yaml"
  },
  {
    name: "15_piraeus_operator_manager_rolebinding_cluster_role_binding",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/15_piraeus_operator_manager_rolebinding_cluster_role_binding.yaml"
  },
  {
    name: "16_piraeus_operator_image_config_config_map",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/16_piraeus_operator_image_config_config_map.yaml"
  },
  {
    name: "17_piraeus_operator_webhook_service_service",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/17_piraeus_operator_webhook_service_service.yaml"
  },
  {
    name: "18_piraeus_operator_controller_manager_deployment",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/18_piraeus_operator_controller_manager_deployment.yaml"
  },
  {
    name: "19_piraeus_operator_gencert_deployment",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/19_piraeus_operator_gencert_deployment.yaml"
  },
  {
    name: "20_piraeus_operator_validating_webhook_configuration_validating_webhook_configuration",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/20_piraeus_operator_validating_webhook_configuration_validating_webhook_configuration.yaml"
  }
] as const

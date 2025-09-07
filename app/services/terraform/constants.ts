export const TEKTON_CRD_FILES = [
  {
    name: '01-tekton-pipelines-namespace',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/01-tekton-pipelines-namespace.yaml'
  },
  {
    name: '02-tekton-pipelines-controller-cluster-access-cluster-role',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/02-tekton-pipelines-controller-cluster-access-cluster-role.yaml'
  },
  {
    name: '03-tekton-pipelines-controller-tenant-access-cluster-role',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/03-tekton-pipelines-controller-tenant-access-cluster-role.yaml'
  },
  {
    name: '04-tekton-pipelines-webhook-cluster-access-cluster-role',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/04-tekton-pipelines-webhook-cluster-access-cluster-role.yaml'
  },
  {
    name: '05-tekton-events-controller-cluster-access-cluster-role',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/05-tekton-events-controller-cluster-access-cluster-role.yaml'
  },
  {
    name: '06-tekton-pipelines-controller-role',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/06-tekton-pipelines-controller-role.yaml'
  },
  {
    name: '07-tekton-pipelines-webhook-role',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/07-tekton-pipelines-webhook-role.yaml'
  },
  {
    name: '08-tekton-pipelines-events-controller-role',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/08-tekton-pipelines-events-controller-role.yaml'
  },
  {
    name: '09-tekton-pipelines-leader-election-role',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/09-tekton-pipelines-leader-election-role.yaml'
  },
  {
    name: '10-tekton-pipelines-info-role',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/10-tekton-pipelines-info-role.yaml'
  },
  {
    name: '11-tekton-pipelines-controller-service-account',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/11-tekton-pipelines-controller-service-account.yaml'
  },
  {
    name: '12-tekton-pipelines-webhook-service-account',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/12-tekton-pipelines-webhook-service-account.yaml'
  },
  {
    name: '13-tekton-events-controller-service-account',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/13-tekton-events-controller-service-account.yaml'
  },
  {
    name: '14-tekton-pipelines-controller-cluster-access-cluster-role-binding',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/14-tekton-pipelines-controller-cluster-access-cluster-role-binding.yaml'
  },
  {
    name: '15-tekton-pipelines-controller-tenant-access-cluster-role-binding',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/15-tekton-pipelines-controller-tenant-access-cluster-role-binding.yaml'
  },
  {
    name: '16-tekton-pipelines-webhook-cluster-access-cluster-role-binding',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/16-tekton-pipelines-webhook-cluster-access-cluster-role-binding.yaml'
  },
  {
    name: '17-tekton-events-controller-cluster-access-cluster-role-binding',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/17-tekton-events-controller-cluster-access-cluster-role-binding.yaml'
  },
  {
    name: '18-tekton-pipelines-controller-role-binding',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/18-tekton-pipelines-controller-role-binding.yaml'
  },
  {
    name: '19-tekton-pipelines-webhook-role-binding',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/19-tekton-pipelines-webhook-role-binding.yaml'
  },
  {
    name: '20-tekton-pipelines-controller-leaderelection-role-binding',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/20-tekton-pipelines-controller-leaderelection-role-binding.yaml'
  },
  {
    name: '21-tekton-pipelines-webhook-leaderelection-role-binding',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/21-tekton-pipelines-webhook-leaderelection-role-binding.yaml'
  },
  {
    name: '22-tekton-pipelines-info-role-binding',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/22-tekton-pipelines-info-role-binding.yaml'
  },
  {
    name: '23-tekton-pipelines-events-controller-role-binding',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/23-tekton-pipelines-events-controller-role-binding.yaml'
  },
  {
    name: '24-tekton-events-controller-leaderelection-role-binding',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/24-tekton-events-controller-leaderelection-role-binding.yaml'
  },
  {
    name: '25-customruns.tekton.dev-custom-resource-definition',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/25-customruns.tekton.dev-custom-resource-definition.yaml'
  },
  {
    name: '26-pipelines.tekton.dev-custom-resource-definition',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/26-pipelines.tekton.dev-custom-resource-definition.yaml'
  },
  {
    name: '27-pipelineruns.tekton.dev-custom-resource-definition',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/27-pipelineruns.tekton.dev-custom-resource-definition.yaml'
  },
  {
    name: '28-resolutionrequests.resolution.tekton.dev-custom-resource-definition',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/28-resolutionrequests.resolution.tekton.dev-custom-resource-definition.yaml'
  },
  {
    name: '29-stepactions.tekton.dev-custom-resource-definition',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/29-stepactions.tekton.dev-custom-resource-definition.yaml'
  },
  {
    name: '30-tasks.tekton.dev-custom-resource-definition',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/30-tasks.tekton.dev-custom-resource-definition.yaml'
  },
  {
    name: '31-taskruns.tekton.dev-custom-resource-definition',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/31-taskruns.tekton.dev-custom-resource-definition.yaml'
  },
  {
    name: '32-verificationpolicies.tekton.dev-custom-resource-definition',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/32-verificationpolicies.tekton.dev-custom-resource-definition.yaml'
  },
  {
    name: '33-webhook-certs-secret',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/33-webhook-certs-secret.yaml'
  },
  {
    name: '34-validation.webhook.pipeline.tekton.dev-validating-webhook-configuration',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/34-validation.webhook.pipeline.tekton.dev-validating-webhook-configuration.yaml'
  },
  {
    name: '35-webhook.pipeline.tekton.dev-mutating-webhook-configuration',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/35-webhook.pipeline.tekton.dev-mutating-webhook-configuration.yaml'
  },
  {
    name: '36-config.webhook.pipeline.tekton.dev-validating-webhook-configuration',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/36-config.webhook.pipeline.tekton.dev-validating-webhook-configuration.yaml'
  },
  {
    name: '37-tekton-aggregate-edit-cluster-role',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/37-tekton-aggregate-edit-cluster-role.yaml'
  },
  {
    name: '38-tekton-aggregate-view-cluster-role',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/38-tekton-aggregate-view-cluster-role.yaml'
  },
  {
    name: '39-config-defaults-config-map',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/39-config-defaults-config-map.yaml'
  },
  {
    name: '40-config-events-config-map',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/40-config-events-config-map.yaml'
  },
  {
    name: '41-feature-flags-config-map',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/41-feature-flags-config-map.yaml'
  },
  {
    name: '42-pipelines-info-config-map',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/42-pipelines-info-config-map.yaml'
  },
  {
    name: '43-config-leader-election-controller-config-map',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/43-config-leader-election-controller-config-map.yaml'
  },
  {
    name: '44-config-leader-election-events-config-map',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/44-config-leader-election-events-config-map.yaml'
  },
  {
    name: '45-config-leader-election-webhook-config-map',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/45-config-leader-election-webhook-config-map.yaml'
  },
  {
    name: '46-config-logging-config-map',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/46-config-logging-config-map.yaml'
  },
  {
    name: '47-config-observability-config-map',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/47-config-observability-config-map.yaml'
  },
  {
    name: '48-config-registry-cert-config-map',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/48-config-registry-cert-config-map.yaml'
  },
  {
    name: '49-config-spire-config-map',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/49-config-spire-config-map.yaml'
  },
  {
    name: '50-config-tracing-config-map',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/50-config-tracing-config-map.yaml'
  },
  {
    name: '51-config-wait-exponential-backoff-config-map',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/51-config-wait-exponential-backoff-config-map.yaml'
  },
  {
    name: '52-tekton-pipelines-controller-deployment',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/52-tekton-pipelines-controller-deployment.yaml'
  },
  {
    name: '53-tekton-pipelines-controller-service',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/53-tekton-pipelines-controller-service.yaml'
  },
  {
    name: '54-tekton-events-controller-deployment',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/54-tekton-events-controller-deployment.yaml'
  },
  {
    name: '55-tekton-events-controller-service',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/55-tekton-events-controller-service.yaml'
  },
  {
    name: '56-tekton-pipelines-resolvers-namespace',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/56-tekton-pipelines-resolvers-namespace.yaml'
  },
  {
    name: '57-tekton-pipelines-resolvers-resolution-request-updates-cluster-role',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/57-tekton-pipelines-resolvers-resolution-request-updates-cluster-role.yaml'
  },
  {
    name: '58-tekton-pipelines-resolvers-namespace-rbac-role',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/58-tekton-pipelines-resolvers-namespace-rbac-role.yaml'
  },
  {
    name: '59-tekton-pipelines-resolvers-service-account',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/59-tekton-pipelines-resolvers-service-account.yaml'
  },
  {
    name: '60-tekton-pipelines-resolvers-cluster-role-binding',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/60-tekton-pipelines-resolvers-cluster-role-binding.yaml'
  },
  {
    name: '61-tekton-pipelines-resolvers-namespace-rbac-role-binding',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/61-tekton-pipelines-resolvers-namespace-rbac-role-binding.yaml'
  },
  {
    name: '62-bundleresolver-config-config-map',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/62-bundleresolver-config-config-map.yaml'
  },
  {
    name: '63-cluster-resolver-config-config-map',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/63-cluster-resolver-config-config-map.yaml'
  },
  {
    name: '64-resolvers-feature-flags-config-map',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/64-resolvers-feature-flags-config-map.yaml'
  },
  {
    name: '65-config-leader-election-resolvers-config-map',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/65-config-leader-election-resolvers-config-map.yaml'
  },
  {
    name: '66-config-logging-config-map',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/66-config-logging-config-map.yaml'
  },
  {
    name: '67-config-observability-config-map',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/67-config-observability-config-map.yaml'
  },
  {
    name: '68-git-resolver-config-config-map',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/68-git-resolver-config-config-map.yaml'
  },
  {
    name: '69-http-resolver-config-config-map',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/69-http-resolver-config-config-map.yaml'
  },
  {
    name: '70-hubresolver-config-config-map',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/70-hubresolver-config-config-map.yaml'
  },
  {
    name: '71-tekton-pipelines-remote-resolvers-deployment',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/71-tekton-pipelines-remote-resolvers-deployment.yaml'
  },
  {
    name: '72-tekton-pipelines-remote-resolvers-service',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/72-tekton-pipelines-remote-resolvers-service.yaml'
  },
  {
    name: '73-tekton-pipelines-webhook-horizontal-pod-autoscaler',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/73-tekton-pipelines-webhook-horizontal-pod-autoscaler.yaml'
  },
  {
    name: '74-tekton-pipelines-webhook-deployment',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/74-tekton-pipelines-webhook-deployment.yaml'
  },
  {
    name: '75-tekton-pipelines-webhook-service',
    url: 'https://raw.githubusercontent.com/kibamail/kibaship/main/crds/tekton/v1.3.1/75-tekton-pipelines-webhook-service.yaml'
  }
] as const

export const PIRAEUS_CRD_FILES = [
  {
    name: "01-namespace",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/01-namespace.yaml"
  },
  {
    name: "02-crd-linstor-clusters",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/02-crd-linstor-clusters.yaml"
  },
  {
    name: "03-crd-linstor-node-connections",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/03-crd-linstor-node-connections.yaml"
  },
  {
    name: "04-crd-linstor-satellite-configurations",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/04-crd-linstor-satellite-configurations.yaml"
  },
  {
    name: "05-crd-linstor-satellites",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/05-crd-linstor-satellites.yaml"
  },
  {
    name: "06-service-account-piraeus-datastore",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/06-service-account-piraeus-datastore.yaml"
  },
  {
    name: "07-service-account-piraeus-operator-gencert",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/07-service-account-piraeus-operator-gencert.yaml"
  },
  {
    name: "08-role-piraeus-operator-gencert",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/08-role-piraeus-operator-gencert.yaml"
  },
  {
    name: "09-role-piraeus-operator-leader-election-role",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/09-role-piraeus-operator-leader-election-role.yaml"
  },
  {
    name: "10-cluster-role-piraeus-operator-controller-manager",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/10-cluster-role-piraeus-operator-controller-manager.yaml"
  },
  {
    name: "11-cluster-role-piraeus-operator-gencert",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/11-cluster-role-piraeus-operator-gencert.yaml"
  },
  {
    name: "12-role-binding-piraeus-operator-gencert",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/12-role-binding-piraeus-operator-gencert.yaml"
  },
  {
    name: "13-role-binding-piraeus-operator-leader-election-rolebinding",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/13-role-binding-piraeus-operator-leader-election-rolebinding.yaml"
  },
  {
    name: "14-cluster-role-binding-piraeus-operator-gencert",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/14-cluster-role-binding-piraeus-operator-gencert.yaml"
  },
  {
    name: "15-cluster-role-binding-piraeus-operator-manager-rolebinding",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/15-cluster-role-binding-piraeus-operator-manager-rolebinding.yaml"
  },
  {
    name: "16-config-map-piraeus-operator-image-config",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/16-config-map-piraeus-operator-image-config.yaml"
  },
  {
    name: "17-piraeus-operator-webhook-service",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/17-piraeus-operator-webhook-service.yaml"
  },
  {
    name: "18-deployment-piraeus-operator-controller-manager",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/18-deployment-piraeus-operator-controller-manager.yaml"
  },
  {
    name: "19-piraeus-operator-gencert",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/19-piraeus-operator-gencert.yaml"
  },
  {
    name: "20-validating-webhook-configuration-piraeus-operator-validating-webhook-configuration",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/linstor/v2.9.0/20-validating-webhook-configuration-piraeus-operator-validating-webhook-configuration.yaml"
  }
] as const

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
  },
  {
    name: "10-xbackendtrafficpolicies",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/gateway/v1.3.0/10-xbackendtrafficpolicies.gateway.networking.x-k8s.io-custom-resource-definition.yaml"
  },
  {
    name: "11-xlistenersets",
    url: "https://raw.githubusercontent.com/kibamail/kibaship/main/crds/gateway/v1.3.0/11-xlistenersets.gateway.networking.x-k8s.io-custom-resource-definition.yaml"
  }
] as const

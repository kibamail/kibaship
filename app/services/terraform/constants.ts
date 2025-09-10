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

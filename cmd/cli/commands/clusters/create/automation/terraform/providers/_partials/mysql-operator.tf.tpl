{{- define "mysql-operator" -}}
resource "helm_release" "mysql_operator" {
  name             = "mysql-operator"
  repository       = "https://mysql.github.io/mysql-operator/"
  chart            = "mysql-operator"
  version          = "2.2.5"
  namespace        = "mysql-operator"
  create_namespace = true
  cleanup_on_fail  = true
  replace          = true

  set = [
    {
      name  = "replicas"
      value = "3"
    },
    {
      name  = "envs.k8sClusterDomain"
      value = "cluster.local"
    }
  ]

  depends_on = [helm_release.cilium]
}
{{- end -}}

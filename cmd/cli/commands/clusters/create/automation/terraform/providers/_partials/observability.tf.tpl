{{- define "observability-namespace" -}}
resource "kubernetes_namespace" "observability" {
  metadata {
    name = "observability"
  }

  depends_on = [
    kubernetes_storage_class.storage_replica_1,
    kubernetes_storage_class.storage_replica_2,
    kubernetes_storage_class.storage_replica_rwm_1,
    helm_release.cilium,
    {{- if .UseLocalPathProvisioner }}
    null_resource.patch_kind_storage_class,
    {{- end }}
    helm_release.cert_manager
  ]
}
{{- end -}}

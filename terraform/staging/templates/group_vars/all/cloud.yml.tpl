# =============================================================================
# KibaShip Staging Cloud Provider Configuration
# =============================================================================
# This file explicitly disables all cloud provider integrations to ensure
# Kubernetes does not attempt to automatically manage cloud resources.
# We manage all infrastructure (load balancers, storage, networking) via Terraform.

# =============================================================================
# Cloud Provider Disable
# =============================================================================
# Disable the main cloud provider setting
cloud_provider: ""

# Disable external cloud provider (no cloud controller manager)
external_cloud_provider: ""

# =============================================================================
# Load Balancer Providers - DISABLED
# =============================================================================
# Disable automatic load balancer provisioning from cloud providers
# We use external load balancers managed by Terraform

# MetalLB (for bare metal load balancing) - disabled as we use external LBs
metallb_enabled: false

# OpenStack load balancer - disabled
external_openstack_lbaas_enabled: false

# =============================================================================
# Storage Providers - DISABLED
# =============================================================================
# Disable cloud storage provisioners as we manage storage via Terraform
# and OpenEBS Mayastor

# AWS EBS CSI
aws_ebs_csi_enabled: false

# Azure Disk CSI
azure_csi_enabled: false

# GCP Persistent Disk CSI
gcp_pd_csi_enabled: false

# OpenStack Cinder CSI
cinder_csi_enabled: false

# vSphere CSI
vsphere_csi_enabled: false

# UpCloud CSI
upcloud_csi_enabled: false

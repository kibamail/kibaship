# Install cni

First setup gateway api crds:

```
kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.2.0/config/crd/standard/gateway.networking.k8s.io_gatewayclasses.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.2.0/config/crd/standard/gateway.networking.k8s.io_gateways.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.2.0/config/crd/standard/gateway.networking.k8s.io_httproutes.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.2.0/config/crd/standard/gateway.networking.k8s.io_referencegrants.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.2.0/config/crd/standard/gateway.networking.k8s.io_grpcroutes.yaml
```

Next install cni:

```bash
cilium install \
    --set ipam.mode=kubernetes \
    --set kubeProxyReplacement=true \
    --set securityContext.capabilities.ciliumAgent="{CHOWN,KILL,NET_ADMIN,NET_RAW,IPC_LOCK,SYS_ADMIN,SYS_RESOURCE,DAC_OVERRIDE,FOWNER,SETGID,SETUID}" \
    --set securityContext.capabilities.cleanCiliumState="{NET_ADMIN,SYS_ADMIN,SYS_RESOURCE}" \
    --set cgroup.autoMount.enabled=false \
    --set cgroup.hostRoot=/sys/fs/cgroup \
    --set k8sServiceHost=localhost \
    --set k8sServicePort=7445 \
    --set gatewayAPI.enabled=true \
    --set gatewayAPI.enableAlpn=true \
    --set gatewayAPI.enableAppProtocol=true
```

Actually, a much easier approach is to generate the helm template, store it as a manifest and auto apply it on talos bootstrap as an extraManifest.

```bash
helm install \
    cilium \
    cilium/cilium \
    --version 1.18.0 \
    --namespace kube-system \
    --set ipam.mode=kubernetes \
    --set kubeProxyReplacement=true \
    --set securityContext.capabilities.ciliumAgent="{CHOWN,KILL,NET_ADMIN,NET_RAW,IPC_LOCK,SYS_ADMIN,SYS_RESOURCE,DAC_OVERRIDE,FOWNER,SETGID,SETUID}" \
    --set securityContext.capabilities.cleanCiliumState="{NET_ADMIN,SYS_ADMIN,SYS_RESOURCE}" \
    --set cgroup.autoMount.enabled=false \
    --set cgroup.hostRoot=/sys/fs/cgroup \
    --set k8sServiceHost=localhost \
    --set k8sServicePort=7445 \
    --set=gatewayAPI.enabled=true \
    --set=gatewayAPI.enableAlpn=true \
    --set=gatewayAPI.enableAppProtocol=true
```

```tf
extraManifests = [
    # Install gateway api CRDs
    "https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.2.0/config/crd/standard/gateway.networking.k8s.io_gatewayclasses.yaml",
    "https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.2.0/config/crd/standard/gateway.networking.k8s.io_gateways.yaml",
    "https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.2.0/config/crd/standard/gateway.networking.k8s.io_httproutes.yaml",
    "https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.2.0/config/crd/standard/gateway.networking.k8s.io_referencegrants.yaml",
    "https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.2.0/config/crd/standard/gateway.networking.k8s.io_grpcroutes.yaml",

    # TCP CRDs Gateway api
    "https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.2.0/config/crd/experimental/gateway.networking.k8s.io_tlsroutes.yaml",

    # Linstor - Piraeus operator
    "https://github.com/piraeusdatastore/piraeus-operator/releases/download/v2.9.0/manifest.yaml"
]
```

# Hetzner bare metal

Run the following command to identify disk unique ids:

```bash
ls -l /dev/disk/by-id

# output

root@rescue ~ # ls -l /dev/disk/by-id/
total 0
lrwxrwxrwx 1 root root 13 Sep  8 04:38 nvme-SAMSUNG_MZVL2512HCJQ-00B07_S63CNX0Y425245 -> ../../nvme0n1
lrwxrwxrwx 1 root root 15 Sep  8 04:38 nvme-SAMSUNG_MZVL2512HCJQ-00B07_S63CNX0Y425245-part1 -> ../../nvme0n1p1
lrwxrwxrwx 1 root root 15 Sep  8 04:38 nvme-SAMSUNG_MZVL2512HCJQ-00B07_S63CNX0Y425245-part2 -> ../../nvme0n1p2
lrwxrwxrwx 1 root root 15 Sep  8 04:38 nvme-SAMSUNG_MZVL2512HCJQ-00B07_S63CNX0Y425245-part3 -> ../../nvme0n1p3
lrwxrwxrwx 1 root root 13 Sep  8 04:38 nvme-SAMSUNG_MZVL2512HCJQ-00B07_S63CNX0Y425245_1 -> ../../nvme0n1
lrwxrwxrwx 1 root root 15 Sep  8 04:38 nvme-SAMSUNG_MZVL2512HCJQ-00B07_S63CNX0Y425245_1-part1 -> ../../nvme0n1p1
lrwxrwxrwx 1 root root 15 Sep  8 04:38 nvme-SAMSUNG_MZVL2512HCJQ-00B07_S63CNX0Y425245_1-part2 -> ../../nvme0n1p2
lrwxrwxrwx 1 root root 15 Sep  8 04:38 nvme-SAMSUNG_MZVL2512HCJQ-00B07_S63CNX0Y425245_1-part3 -> ../../nvme0n1p3
lrwxrwxrwx 1 root root 13 Sep  8 04:38 nvme-SAMSUNG_MZVL2512HCJQ-00B07_S63CNX0Y425254 -> ../../nvme1n1
lrwxrwxrwx 1 root root 13 Sep  8 04:38 nvme-SAMSUNG_MZVL2512HCJQ-00B07_S63CNX0Y425254_1 -> ../../nvme1n1
lrwxrwxrwx 1 root root 13 Sep  8 04:38 nvme-eui.002538b451b5c3fb -> ../../nvme0n1
lrwxrwxrwx 1 root root 15 Sep  8 04:38 nvme-eui.002538b451b5c3fb-part1 -> ../../nvme0n1p1
lrwxrwxrwx 1 root root 15 Sep  8 04:38 nvme-eui.002538b451b5c3fb-part2 -> ../../nvme0n1p2
lrwxrwxrwx 1 root root 15 Sep  8 04:38 nvme-eui.002538b451b5c3fb-part3 -> ../../nvme0n1p3
lrwxrwxrwx 1 root root 13 Sep  8 04:38 nvme-eui.002538b451b5c404 -> ../../nvme1n1
```

From the above output, we will install talos on `/dev/disk/by-id/nvme-SAMSUNG_MZVL2512HCJQ-00B07_S63CNX0Y425245`

We will then use `/dev/disk/by-id/nvme-SAMSUNG_MZVL2512HCJQ-00B07_S63CNX0Y425254` for linstor storage and volumes on the cluster

Prepare image on factory, select drbd.

Example link download and extract:

Store target disk: /dev/nvme0n1 (Always use the first attached disk on hetzner)

```bash
TARGET_DISK=/dev/disk/by-id/nvme-SAMSUNG_MZVL2512HCJQ-00B07_S63CNX0Y425254

TARGET_VERSION="v1.11.1"
```

```bash
wget https://factory.talos.dev/image/376567988ad370138ad8b2698212367b8edcb69b5fd68c80be1f2ec7d603b4ba/v1.11.0/metal-amd64.raw.zst
wget https://factory.talos.dev/image/376567988ad370138ad8b2698212367b8edcb69b5fd68c80be1f2ec7d603b4ba/v1.11.2/metal-arm64.raw.zst

# amd64 376567988ad370138ad8b2698212367b8edcb69b5fd68c80be1f2ec7d603b4ba
# arm64 376567988ad370138ad8b2698212367b8edcb69b5fd68c80be1f2ec7d603b4ba

zstd -d metal-amd64.raw.zst

dd if=metal-amd64.raw of=${TARGET_DISK} bs=1M status=progress

sync

# Reboot the server
reboot
```

Write the raw disk

```bash
dd if=metal-amd64.raw of=${TARGET_DISK} bs=1M status=progress

sync

# Reboot the server
reboot
```

After the reboot, you should be able to talk to the talos os:

```bash
NODE_IP=65.109.58.113

talosctl --nodes ${NODE_IP} get disks --insecure
talosctl --nodes ${NODE_IP} get addresses --insecure
talosctl --nodes ${NODE_IP} get links --insecure
```

# Talos image configuration

In talos factory, to generate the image hash, we need to ensure the following modules are selected for longhorn:

- siderolabs/iscsi-tools (v0.2.0)
- siderolabs/util-linux-tools (2.41.1)

# Bare metal talos linux factory image:

```yaml
customization:
  systemExtensions:
    officialExtensions:
      - siderolabs/iscsi-tools
      - siderolabs/nut-client
      - siderolabs/nvme-cli
      - siderolabs/util-linux-tools
```

Your image schematic ID is: `3df38c5e5faf43879e6ff0f13c6b0ba02aaa0eb5f9291f28749c4056c1974e7b`

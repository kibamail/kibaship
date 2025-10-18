# Pod sandbox error

This causes every single pod in the cluster to restart, essentially restarting each kubelet which restarts all pods.

I noticed the bare metal talos image i was using included `siderolabs/nut-client` . I also noticed the dmesg had error logs about this nut client.
I removed this and the error went away.

New image now includes the following extensions:

```bash
customization:
    systemExtensions:
        officialExtensions:
            - siderolabs/iscsi-tools
            - siderolabs/mdadm
            - siderolabs/nvme-cli
            - siderolabs/util-linux-tools
```

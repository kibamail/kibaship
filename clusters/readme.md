# Bare metal setup process

First, generate secrets:


```bash
talosctl gen secrets
```

Next, generate config:

```bash
talosctl gen config --with-secrets secrets.yaml --output-types talosconfig --output talosconfig.yaml $CLUSTER_NAME $$CLUSTER_ENDPOINT
```

Now for each node in the inventory.yaml, run the following command to generate the machine configs:

```bash
talosctl gen config $CLUSTER_NAME $CLUSTER_ENDPOINT --output-types controlplane --output nodes/cp-1.h.kibaship.app.yaml --with-docs=false --with-examples=false --with-secrets=secrets.yaml
```

Possible problems:

I ran into an issue where control plane one could not bootstrap. I had to reset it with the following command:

```bash
talosctl --nodes 65.109.58.113 --talosconfig talosconfig.yaml reset --graceful=false --reboot --system-labels-to-wipe=EPHEMERAL
```

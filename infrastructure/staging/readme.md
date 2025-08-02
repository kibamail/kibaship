# How to setup staging.

After running `/home/ubuntu/kubespray/install-k8s.sh` script in the jump server, we get a fully fleshed kubernetes cluster with the following installed:

- 3 worker nodes
- 3 control nodes
- Cilium CNI
- Cilium gateway crds

The first thing we need to do is get the kubeconfig for further operations.

### Get kubeconfig from server

Make sure you are in the `/infrastructure/staging` folder, and then run the following command to copy the kubeconfig into the secrets folder:

```bash
export JUMP_SERVER_IP=<jump_server_ip>

scp -i infrastructure/staging/.secrets/staging/id_ed25519 ubuntu@$JUMP_SERVER_IP:/home/ubuntu/kubespray/inventory/kibaship-staging/artifacts/admin.conf infrastructure/staging/.secrets/staging/kubeconfig
```

### Setup kubeconfig in environment

From the root of the repository, run:

```bash
export KUBECONFIG=$(pwd)/infrastructure/staging/.secrets/staging/kubeconfig
```

### Install argocd for gitops

```bash
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
```

### Retrieve argocd default admin password

Using the argocd cli, we can retrieve the default admin password.

```bash
brew install argocd

argocd admin initial-password -n argocd
```

### Login to argocd in the browser

Run the following command to port forward the argocd server to your local machine:

```bash
kubectl port-forward svc/argocd-server -n argocd 8080:443
```

### Apply the default app of apps

```bash
kubectl apply -f argocd/app-of-apps/staging.yaml
```

# Acme DNS v1.1

Acme DNS is a key component that helps automate the process of SSL certificate acquisition and renewal in the kibaship cluster.

Here's the problem it tries to solve:

## The Problem

A developer installs kibaship cluster with `h.kibaship.app` as their domain of choice. What this means is:

- New applications will automatically get assigned a subdomain like `curly-sympathy-43xk.apps.h.kibaship.app`
- New valkey database clusters will automatically get assigned a subdomain like `destroyed-bounces-iup5.valkey.h.kibaship.app`
- New mysql database clusters will automatically get assigned a subdomain like `crimson-sandbar-re5g.mysql.h.kibaship.app`
- New postgres database clusters will automatically get assigned a subdomain like `lagoon-minaret-wp9s.postgres.h.kibaship.app`

In order for this to work, first we need to acquire one wildcard certificate for your domain `h.kibaship.app` using cert-manager, and this certificate will include valid SANs (Subject Alternative Names) for `*.apps.h.kibaship.app`, `*.valkey.h.kibaship.app`, `*.mysql.h.kibaship.app`, and `*.postgres.h.kibaship.app`.

So cert-manager running inside of your cluster needs you to complete an ACME challenge to make this happen. An example of a challenge is:

```bash
# These are the TXT records Let's Encrypt requires, and cert-manager will give it to us, and we will show them to you. Cert-manager talks to let's encrypt, and gives us the information we need:

# Challenge for *.apps.h.kibaship.app
_acme-challenge.apps.h.kibaship.app. 300 IN TXT "9XJPe8r4KNvVGDfm3zF7Q8sL2nK5wH6pY1vR3bT8cN4"

# Challenge for *.valkey.h.kibaship.app
_acme-challenge.valkey.h.kibaship.app. 300 IN TXT "xM2nB4vL8zR5tP9qW3kJ7yF6hN1sD4gC8rT5mV2nB9"

# Challenge for *.mysql.h.kibaship.app
_acme-challenge.mysql.h.kibaship.app. 300 IN TXT "pL7kR2mN9vB4xT5wQ8jF3nH6yC1sG4zD7tK5rM2vB8"

# Challenge for *.postgres.h.kibaship.app
_acme-challenge.postgres.h.kibaship.app. 300 IN TXT "tQ5mV8nB2xL7kR4wP9jF6yH3sC1gN4zD8rT2vM5kB7"
```

Great. Once you configure these TXT records, cert-manager will acquire a wildcard certificate that will work for all your services. But there's a problem. In 90 days, these certificates will expire.

And we will need to ask you to complete the challenge again. Nobody wants that.

So what's the solution? Well, we could ask you which DNS provider you use, and then automate the process of renewing the certificate every 90 days for you. But in order to do this, your DNS API keys will have to live in the cluster, and now we need to think about security for those API keys, like rotation (which might still need your periodic intervention).

So this doesn't work. Well, what can we do? Ah yes.

We'll run our own DNS servers inside of your cluster (acme-dns), and instead of asking you to complete the challenge by adding those TXT records above, we'll instead ask you to configure your DNS to delegate all Let's Encrypt ACME challenges to this acme-dns server running inside of your cluster.

## The Solution: DNS Delegation

So you will configure the following instead:

```bash
# NS record for DNS delegation, telling your dns provider to forward all dns queries for _acme-challenge.h.kibaship.app to acme-dns running in your cluster
_acme-challenge    IN CNAME    eabcdb41-d89f-4580-826f-3e62e9755ef2.acme.h.kibaship.app.

# A record to reach acme-dns server. Telling the entire internet that acme.h.kibaship.app can be reached by going through your load balancer. Your load balancer is automatically configured by kibaship to forward all port 53 traffic (dns traffic) to acme-dns inside of your cluster
acme.h.kibaship.app.    IN A    <LOAD_BALANCER_IP>


# NS record: Declares that acme.h.kibaship.app is an authoritative nameserver
# for itself and all its subdomains (*.acme.h.kibaship.app).
# This tells your DNS provider: "Don't try to answer queries for
# acme.h.kibaship.app - delegate those queries to the nameserver at
# acme.h.kibaship.app itself (which is your acme-dns server)"
acme.h.kibaship.app.    IN NS   acme.h.kibaship.app.
```

```txt
┌─────────────────────────────────────────────────────────┐
│ Let's Encrypt                                           │
│ Queries: _acme-challenge.apps.h.kibaship.app TXT?       │
└────────────────────┬────────────────────────────────────┘
                     ↓
┌─────────────────────────────────────────────────────────┐
│ Your DNS Provider (Cloudflare/etc)                      │
│                                                         │
│ Step 1: Check for TXT record                            │
│ ❌ No TXT record found                                  │
│                                                         │
│ Step 2: Check for CNAME                                 │
│ ✅ Found CNAME → eabcdb41...acme.h.kibaship.app         │
│ "Redirect query to that domain instead"                 │
└────────────────────┬────────────────────────────────────┘
                     ↓
┌─────────────────────────────────────────────────────────┐
│ Your DNS Provider (new query)                           │
│ Query: eabcdb41...acme.h.kibaship.app TXT?              │
│                                                         │
│ Step 3: Do I have this TXT record?                      │
│ ❌ No, I don't                                          │
│                                                         │
│ Step 4: Check for NS delegation                         │
│ ✅ Found NS record: acme.h.kibaship.app                 │
│ "This subdomain is managed by another nameserver"       │
│ "Forward query there instead"                           │
└────────────────────┬────────────────────────────────────┘
                     ↓
┌─────────────────────────────────────────────────────────┐
│ Your DNS Provider (lookup NS server location)           │
│ "Where is acme.h.kibaship.app?"                         │
│ ✅ A record: <LOAD_BALANCER_IP>                         │
└────────────────────┬────────────────────────────────────┘
                     ↓
┌─────────────────────────────────────────────────────────┐
│ Query sent to: <LOAD_BALANCER_IP>:53                    │
└────────────────────┬────────────────────────────────────┘
                     ↓
┌─────────────────────────────────────────────────────────┐
│ acme-dns server in your cluster                         │
│ Responds: "eabcdb41...acme.h.kibaship.app TXT xyz789"   │
└─────────────────────────────────────────────────────────┘
```

When you add these records to your DNS provider, this tells your DNS provider: Hey, if anyone ever wants to know anything about DNS records for `_acme-challenge` under `h.kibaship.app`, please ask the DNS server at `acme.h.kibaship.app`.

Who will want to know? Let's Encrypt. When cert-manager requests a wildcard certificate for `*.apps.h.kibaship.app`, Let's Encrypt will need to verify you control the domain by checking for a TXT record at `_acme-challenge.apps.h.kibaship.app`. Your DNS provider will see the CNAME delegation and forward that query to your acme-dns server running inside of your cluster.

## How It Actually Works

Let's walk through the complete flow:

### Initial Setup

When you install kibaship with domain `h.kibaship.app`:

1. **Kibaship installer deploys acme-dns** in your cluster as a pod listening on port 53 (DNS)
2. **NodePort service exposes acme-dns** on port 30053 across all cluster nodes
3. **Load balancer forwards port 53** (both UDP and TCP) to NodePort 30053
4. **Installer outputs the DNS records** you need to add

You add these three records to your DNS provider **ONE TIME**:

```bash
# 1. Wildcard CNAME for all ACME challenges
_acme-challenge    IN CNAME    eabcdb41-d89f-4580-826f-3e62e9755ef2.acme.h.kibaship.app.

# 2. A record pointing to your load balancer
acme.h.kibaship.app.    IN A    45.67.89.10

# 3. NS record for subdomain delegation
acme.h.kibaship.app.    IN NS   acme.h.kibaship.app.
```

That's it. Setup done.

### Certificate Issuance Flow

Now when cert-manager requests a certificate for `*.apps.h.kibaship.app`:

**Step 1:** cert-manager sends certificate request to Let's Encrypt

**Step 2:** Let's Encrypt responds: "Prove you control `apps.h.kibaship.app` by creating a TXT record at `_acme-challenge.apps.h.kibaship.app` with value `xyz789`"

**Step 3:** cert-manager tells acme-dns: "Create this TXT record for me"

```bash
# API call to acme-dns
POST http://acme-dns.paas-system:8080/update
{
  "subdomain": "eabcdb41-d89f-4580-826f-3e62e9755ef2",
  "txt": "xyz789"
}
```

**Step 4:** acme-dns creates the TXT record internally:

```bash
eabcdb41-d89f-4580-826f-3e62e9755ef2.acme.h.kibaship.app.    IN TXT    "xyz789"
```

**Step 5:** Let's Encrypt queries DNS to verify:

```bash
Query: _acme-challenge.apps.h.kibaship.app TXT?
```

**Step 6:** Your DNS provider responds:

```bash
"I don't have that TXT record, but I have a CNAME:
_acme-challenge.apps.h.kibaship.app → eabcdb41-d89f-4580-826f-3e62e9755ef2.acme.h.kibaship.app"
```

**Step 7:** Let's Encrypt follows the CNAME and queries:

```bash
Query: eabcdb41-d89f-4580-826f-3e62e9755ef2.acme.h.kibaship.app TXT?
```

**Step 8:** Your DNS provider sees the NS record and delegates:

```bash
"For acme.h.kibaship.app subdomain queries, ask acme.h.kibaship.app itself"
```

**Step 9:** DNS query arrives at your load balancer on port 53

**Step 10:** Load balancer forwards to cluster nodes on port 30053

**Step 11:** NodePort routes to acme-dns pod on port 53

**Step 12:** acme-dns responds with the TXT record:

```bash
eabcdb41-d89f-4580-826f-3e62e9755ef2.acme.h.kibaship.app.    IN TXT    "xyz789"
```

**Step 13:** Let's Encrypt receives the correct validation string ✅

**Step 14:** Let's Encrypt issues the wildcard certificate for `*.apps.h.kibaship.app`

**Step 15:** cert-manager stores certificate in Kubernetes secret

**Step 16:** cert-manager tells acme-dns to delete the TXT record (cleanup)

### The Beautiful Part: Automatic Renewals

60 days later, when the certificate needs renewal:

1. cert-manager automatically starts renewal process
2. Let's Encrypt generates a **NEW** challenge string (e.g., `def456`)
3. cert-manager tells acme-dns to create the new TXT record
4. Let's Encrypt validates through the same DNS flow
5. New certificate issued
6. TXT record cleaned up

**You never touch DNS again.** The three records you added initially handle all future challenges automatically.

## Supporting Multiple Domains

What happens when a user wants to point their custom domain `ecommerce-giant.com` to their app?

### Adding Custom Domain

User runs:

```bash
kibaship domain add my-app ecommerce-giant.com
```

Kibaship:

1. **Registers the new domain** with acme-dns (gets a unique subdomain like `xyz789`)
2. **Shows user the DNS records** to add in `ecommerce-giant.com`:

```bash
# In ecommerce-giant.com DNS
_acme-challenge.ecommerce-giant.com    IN CNAME    xyz789.acme.h.kibaship.app.
ecommerce-giant.com                    IN A        <LOAD_BALANCER_IP>
```

3. **Waits for DNS propagation**
4. **Requests certificate** for `ecommerce-giant.com`
5. Let's Encrypt validates via the CNAME → acme-dns flow
6. **Certificate issued automatically**
7. **Ingress updated** to route `ecommerce-giant.com` → app

Future renewals? Automatic. Forever.

### Adding Wildcard Domain

User wants `*.wildcard-paas.com` to point to their app:

```bash
kibaship domain add my-app *.wildcard-paas.com
```

Same flow, but user adds in `wildcard-paas.com` DNS:

```bash
_acme-challenge.wildcard-paas.com    IN CNAME    def456.acme.h.kibaship.app.
*.wildcard-paas.com                  IN A        <LOAD_BALANCER_IP>
```

Wildcard certificate issued. Automatic renewals. Done.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    Let's Encrypt                             │
│              (Validates DNS challenges)                      │
└────────────────────┬────────────────────────────────────────┘
                     │ DNS Query: _acme-challenge.apps.h.kibaship.app?
                     ↓
┌─────────────────────────────────────────────────────────────┐
│              User's DNS Provider                             │
│    "I have CNAME → eabcdb41-d89f-4580-826f-3e62e9755ef2.acme.h.kibaship.app"             │
│    "For acme.h.kibaship.app, ask NS acme.h.kibaship.app"  │
└────────────────────┬────────────────────────────────────────┘
                     │ Follows CNAME + NS delegation
                     ↓
┌─────────────────────────────────────────────────────────────┐
│         Load Balancer (45.67.89.10)                         │
│         Port 53 UDP/TCP → Port 30053                        │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ↓
┌─────────────────────────────────────────────────────────────┐
│           Kubernetes Cluster Nodes                           │
│           NodePort 30053 on all nodes                        │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ↓
┌─────────────────────────────────────────────────────────────┐
│              acme-dns Pod (port 53)                          │
│    Dynamically creates/deletes TXT records                   │
│    eabcdb41-d89f-4580-826f-3e62e9755ef2.acme.h.kibaship.app → "xyz789"                   │
└─────────────────────────────────────────────────────────────┘
                     ↑
                     │ API calls
                     │
┌─────────────────────────────────────────────────────────────┐
│           cert-manager (in cluster)                          │
│    Orchestrates certificate requests/renewals                │
└─────────────────────────────────────────────────────────────┘
```

## Why This Solution is Perfect

1. **One-time DNS setup** - Add 3 records, never touch again
2. **Works with ANY DNS provider** - No API keys needed, no provider-specific integration
3. **Fully automated renewals** - Certificates renew every 60 days automatically
4. **No dependency on external services** - Everything runs in user's cluster
5. **Supports unlimited domains** - Each new domain just needs 1 CNAME
6. **Wildcard certificate support** - Both for main domain and custom domains
7. **Secure** - No DNS API keys stored in cluster, no secrets to rotate
8. **Self-contained** - If kibaship.app goes down, user's cluster keeps working

## Technical Requirements

### DNS Records Summary

**For main domain `h.kibaship.app` (one-time setup):**

```bash
_acme-challenge              IN CNAME    eabcdb41-d89f-4580-826f-3e62e9755ef2.acme.h.kibaship.app.
acme.h.kibaship.app.        IN A        <LOAD_BALANCER_IP>
acme.h.kibaship.app.        IN NS       acme.h.kibaship.app.
*.h.kibaship.app.           IN A        <LOAD_BALANCER_IP>
```

**For each custom domain (per domain):**

```bash
_acme-challenge.example.com    IN CNAME    <unique-id>.acme.h.kibaship.app.
example.com                    IN A        <LOAD_BALANCER_IP>
```

### Load Balancer Ports

Your load balancer must forward these ports to cluster NodePorts:

```bash
Port 53 UDP → NodePort 30053  # DNS queries
Port 53 TCP → NodePort 30053  # DNS queries (TCP fallback)
Port 80     → NodePort 30080  # HTTP traffic
Port 443    → NodePort 30443  # HTTPS traffic
```

### Cluster Components

1. **acme-dns deployment** - DNS server for ACME challenges
2. **cert-manager** - Certificate lifecycle management
3. **Gateway API / Cilium** - TLS termination and routing
4. **NodePort services** - Expose services to load balancer

## Troubleshooting

### Verify DNS delegation is working

```bash
# Should return NS record pointing to acme.h.kibaship.app
dig NS acme.h.kibaship.app

# Should resolve to your load balancer IP
dig A acme.h.kibaship.app

# Should get response from your acme-dns server
dig @<LOAD_BALANCER_IP> test.acme.h.kibaship.app
```

### Check acme-dns is running

```bash
kubectl get pods -n paas-system | grep acme-dns
kubectl logs -n paas-system deployment/acme-dns
```

### Verify cert-manager can reach acme-dns

```bash
kubectl get certificates -A
kubectl describe certificate <cert-name> -n <namespace>
```

### Common Issues

**Issue:** Certificate stuck in "Pending" state

- **Check:** DNS records are correctly configured
- **Check:** acme-dns pod is running and healthy
- **Check:** Load balancer is forwarding port 53 correctly

**Issue:** Let's Encrypt validation fails

- **Check:** CNAME record propagated (use `dig +trace`)
- **Check:** NS record is configured for acme.h.kibaship.app
- **Check:** Port 53 UDP and TCP both working

**Issue:** Renewals fail after initial success

- **Check:** acme-dns database is persisted (use PersistentVolume)
- **Check:** acme-dns pod hasn't been recreated without data

## Conclusion

With acme-dns running in your cluster, SSL certificate management becomes completely transparent. Users add a few DNS records during initial setup, and from that moment on, every new app, database, and custom domain gets automatic SSL certificates that renew forever without any manual intervention.

This is the foundation that makes kibaship feel like magic - deploy an app, get a URL with HTTPS, just works. No certificate headaches, no manual renewals, no DNS provider lock-in. Just pure automation.

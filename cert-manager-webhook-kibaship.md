## Cert-Manager Kibaship Webhook — Implementation Plan (Byte-Sized TODOs)

This document tracks the step-by-step tasks required to implement a custom cert-manager DNS01 webhook for Kibaship. Unlike provider webhooks (e.g., Vercel), this webhook does not modify DNS records. Instead, it emits DNS challenge instructions to the project’s existing stream in Valkey so the PAAS frontend can prompt the user to add the TXT record. Cert-manager will later verify and issue the certificate automatically when DNS propagates.

---

### 0) Discovery and Requirements

- [x] Confirm the exact project stream naming convention used by the operator and document the canonical key template.

  - Canonical stream key: `{project:<projectUUID>}:events` using Redis hash tags to keep all keys for a project in the same slot.
  - Sharding: When `StreamShardingEnabled=true` and traffic threshold is exceeded, shard suffix `:{1..N}` is appended, i.e., `{project:<uuid>}:events:<shardIndex>`.
  - Source: `pkg/streaming/publisher.go::generateStreamName()`.

- [x] Confirm which Certificate labels we will rely on to resolve the target project; document required labels.

  - Required (for webhook to compute stream): `platform.kibaship.com/project-uuid` (aka `validation.LabelProjectUUID`).
  - Recommended (for UX/routing):
    - `platform.kibaship.com/workspace-uuid` (aka `validation.LabelWorkspaceUUID`)
    - `platform.kibaship.com/application-uuid` (aka `validation.LabelApplicationUUID`)
    - Optional display hints: `platform.kibaship.com/slug` (project slug), and `platform.kibaship.com/name` as annotation if available.
  - For ApplicationDomain-triggered Certificates, the operator will stamp these labels when creating the Certificate resource.

- [x] Confirm Valkey deployment details and connection approach to mirror the operator.

  - Service/Secret names (defaults): `kibaship-valkey-cluster-kibaship-com` for both service and secret; port `6379`.
  - Namespace: Valkey runs in the operator namespace; use `streaming.DefaultConfig(<valkey-namespace>)` to set names/port.
  - Seed address: `"<service>.<namespace>.svc.cluster.local"` then append port if missing.
  - Auth: single password string stored as the only key in the Secret (see `pkg/streaming/secret.go`); fetched via `SecretManager.GetValkeyPassword`.
  - Client: go-redis Cluster client via `NewValkeyClusterClient` with timeouts/retries from `streaming.Config`.
  - TLS: not currently enabled in operator code path (plain TCP to cluster service).

- [x] Decide and document the message schema for DNS challenges published to the project stream.
  - Chosen approach: Reuse the operator’s stream event envelope for parity with existing consumers.
    - `resource_type = "dns_challenge"`, `operation = "present"|"cleanup"`.
    - `event_data` (JSON) holds DNS01-specific fields.
  - DNS01 event_data (v1): `{ challenge_id, fqdn, zone, txt, ttl, namespace, certificate, issuerRef }` plus optional `{ uiHints, orderName, challengeName, deadlineHint }`.
  - We will compute `challenge_id` deterministically from `namespace/certificate/fqdn/txt` to allow de-duplication.
  - Publisher: prefer reusing `pkg/streaming` abstractions (ConnectionManager + ProjectStreamPublisher) to ensure identical stream naming, metadata, and behavior.

---

### 1) Repository Scaffolding

- [x] Create module directory for the webhook (e.g., `webhooks/cert-manager-kibaship-webhook`).
- [x] Initialize `go.mod` and dependencies (cert-manager webhook libs and k8s client-go pinned for compatibility).
- [x] Add a Dockerfile to build a static binary (multi-stage build, distroless runtime).
- [x] Add `.dockerignore` and Makefile targets (build, docker-build, docker-push for the webhook image).

Notes (completed):

- Module path: `webhooks/cert-manager-kibaship-webhook` with `go.mod` pinned to:
  - cert-manager v1.18.2 (strict), Kubernetes libs v0.32.0, go=1.23
  - Notes: aligns to cert-manager 1.18.x requirements; Dockerfile builder updated to golang:1.23
- Fixed group name in code to `dns.kibaship.com` (no env needed)
- Verified:
  - `make build-cert-manager-webhook` produced `bin/cert-manager-kibaship-webhook`
  - `make docker-build-cert-manager-webhook` produced image `ghcr.io/kibamail/kibaship-operator-cert-manager-webhook:v${VERSION}` (not pushed)

---

### 2) Webhook Server Bootstrap

- [x] Implement `main.go` with fixed GroupName `"dns.kibaship.com"` and call `cmd.RunWebhookServer(GroupName, &solver{})`.
- [x] Define `solver` struct implementing cert-manager’s `webhook.Solver` interface (Initialize, Name, Present, CleanUp) — currently no-op for Present/CleanUp.
- [x] Identifiers:
  - [x] GroupName = `dns.kibaship.com` (no env override required)
  - [x] `solver.Name()` returns `"kibaship"`
- [ ] Health endpoints: rely on webhook library defaults (HTTPS `/healthz`, `/readyz`) — confirm during deployment.

---

### 3) Config Schema and Loading

- [x] Define JSON config struct decoded from `ChallengeRequest.Config`:
  - [x] Redis/Valkey connection (addr, username, passwordSecretRef, TLS bool, timeouts).
  - [x] Optional `uiHints` map for frontend routing (tenantId, projectId, etc.).
- [x] Implement `parseConfig(raw []byte)` helper to parse and validate.
- [x] Implement K8s Secret lookup helper (namespace-aware) to resolve password via SecretRef.

Notes (in progress):

- Types added: `WebhookConfig`, `ValkeyConfig`, `SecretRef`
- Helpers added:
  - `parseConfig([]byte) (WebhookConfig, error)` with minimal validation
  - `readSecretValue(ctx, defaultNS, *SecretRef) (string, error)` on solver using CoreV1 client
- Next: wire config parsing/secret resolution into Present/CleanUp flows

---

### 4) Resolving Project Context from Certificate Labels

- [ ] Implement utility to traverse owner chain to the `Certificate` (Challenge -> Order -> Certificate) to read labels.
- [ ] Document the set of labels that must be present on the `Certificate` for project resolution.
- [ ] Implement `buildProjectStreamKey(labels)` that returns the exact project stream key according to operator’s convention.
- [ ] Add unit tests for the label parsing and key construction.

---

### 5) Valkey (Redis) Client Abstraction

- [ ] Add a small interface `StreamClient` with `XAdd(ctx, stream, fields)`, `Ping(ctx)` methods.
- [ ] Implement a production client using go-redis (support TLS and auth).
- [ ] Add a fake/in-memory client for unit tests.
- [ ] Add exponential backoff connect/retry helper with bounded attempts for `Present`.

---

### 6) Message Schema for Stream Events

- [ ] Define `type` values: `present`, `cleanup`.
- [ ] Define required fields:
  - [ ] `type`, `challenge_id` (hash of namespace/cert/fqdn/txt), `fqdn`, `zone`, `txt`, `ttl`, `namespace`, `certificate`, `issuerRef`, `createdAt` (RFC3339), `version`.
- [ ] Define optional fields:
  - [ ] `uiHints` (map), `deadlineHint` (seconds), `orderName`, `challengeName`.
- [ ] Write a schema doc block and keep it versioned (e.g., `version: 1`).
- [ ] Implement helper to compute deterministic `challenge_id`.

---

### 7) Solver.Present Implementation

- [ ] Load config and Valkey credentials.
- [ ] Resolve project context from Certificate labels (owner chain traversal).
- [ ] Compute stream name using project context.
- [ ] Compose event payload for `type=present` including `challenge_id` and all required fields.
- [ ] XADD to the project stream; if stream/key missing, create implicitly via XADD.
- [ ] Log success with message ID; handle transient errors with retry/backoff and return error on final failure so cert-manager retries.
- [ ] Ensure idempotency: duplicates are OK; consumer must dedupe using `challenge_id`.

---

### 8) Solver.CleanUp Implementation

- [ ] Resolve project context and stream name the same way as Present.
- [ ] Compose event payload for `type=cleanup` (include `challenge_id`, `fqdn`, `txt`).
- [ ] XADD to the project stream (no external deletions required).
- [ ] Log success; tolerate missing context gracefully (still emit best-effort cleanup with available identifiers).

---

### 9) Initialize and K8s Client

- [ ] In `Initialize`, build a k8s clientset from `kubeClientConfig` and store on solver.
- [ ] Optionally perform a lightweight Valkey connectivity check (if static config provided via env) and log warnings (do not fail startup by default).

---

### 10) RBAC and Deployment Manifests

- [ ] Create Kustomize/Helm templates for Deployment, Service, and serving cert secret:
  - [ ] Namespace: `cert-manager`.
  - [ ] Service account and RBAC to:
    - [ ] Read `kube-system` requestheader CA ConfigMap (webhook auth).
    - [ ] Read Secrets in `cert-manager` (or configurable ns) for Valkey credentials.
    - [ ] Read `cert-manager` CRDs (Order, Challenge, Certificate) if needed for owner traversal (get/list across ns as required).
  - [ ] Pod security context and resource requests/limits.
  - [ ] Env: `GROUP_NAME=acme.kibaship.com`.
- [ ] Add a kustomization in `config/cert-manager-webhook/` with images and patches for CI.
- [ ] Add example ClusterIssuer referencing our webhook:
  - [ ] `groupName: acme.kibaship.com`, `solverName: kibaship`.
  - [ ] `dns01.webhook.config` carries Valkey connection details.

---

### 11) Integration with Operator/Streams

- [ ] Confirm the operator’s stream consumer will accept `present`/`cleanup` messages and how it surfaces them to UI.
- [ ] Agree on the exact stream name derived from project labels and document mapping.
- [ ] Add a small doc for the operator team describing the event schema and expectations.

---

### 12) Testing Strategy

- [ ] Unit tests:
  - [ ] Config parsing and secret resolution.
  - [ ] Challenge owner traversal and stream key construction.
  - [ ] Present/CleanUp with fake StreamClient (assert sent fields).
- [ ] E2E light test (cluster-internal):
  - [ ] Deploy webhook and Valkey in Kind; create a dummy Challenge/Issuer setup to trigger Present; assert event appears on the target stream via XREAD.
  - [ ] Skip real DNS propagation; focus on webhook emission path.
- [ ] Negative tests:
  - [ ] Missing labels fail with clear error.
  - [ ] Valkey connection failure triggers retries and then returns an error (so cert-manager retries).

---

### 13) Observability & Ops

- [ ] Structured logging with key fields (challenge_id, fqdn, stream).
- [ ] Optional Prometheus metrics counters: `present_total`, `cleanup_total`, `stream_errors_total`.
- [ ] Add readiness checks (e.g., ephemeral Valkey ping if config provided) and liveness probe.
- [ ] Add feature flags via env for debug logging and dry-run.

---

### 14) Security & Hardening

- [ ] Limit RBAC to only required verbs and namespaces.
- [ ] Support Valkey TLS and optional CA/cert refs.
- [ ] Handle secret rotation (rebuild clients on failure).
- [ ] Validate inputs (FQDN normalization, size limits on payloads).

---

### 15) CI/CD & Makefile

- [ ] Add Makefile targets: `build-webhook`, `docker-build-webhook`, `docker-push-webhook`.
- [ ] Integrate into `build-and-push.yml` workflow (reuse existing release pipeline per repo policy).
- [ ] Version image tags in sync with operator release versioning.

---

### 16) Documentation

- [ ] Create README for the webhook with install steps (Kustomize/Helm), example ClusterIssuer, and troubleshooting.
- [ ] Document label requirements on Certificate and how project stream is resolved.
- [ ] Document the event schema and sample payloads for `present` and `cleanup`.
- [ ] Add a section on expected UX: user sees DNS instructions, configures TXT, issuance proceeds automatically.

---

### 17) Rollout Plan

- [ ] Enable webhook in non-prod (Kind + CI) and verify stream events.
- [ ] Enable in staging with a test certificate; verify UI flow.
- [ ] Gradually roll to prod.
- [ ] Add on-call runbook (common failures, how to inspect streams, how to retry issuance).

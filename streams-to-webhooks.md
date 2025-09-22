# Streams → Webhooks Refactor Plan

This plan replaces Valkey (Redis-compatible) Streams–based event publishing with outbound HTTP webhooks. The operator will stop provisioning/using a Valkey cluster and instead POST JSON payloads to a configured webhook endpoint with robust retry, security, and observability.

## Goals

- Remove all Valkey/Streams code paths and dependencies from the operator and tests.
- Introduce a reliable, secure webhook notifier used by all controllers to publish lifecycle events.
- Provide configuration, retry/backoff, metrics, and structured payloads.
- Preserve at-least-once delivery semantics with idempotent event design.

## Non-goals

- Building a full event bus or queue system.
- Guaranteed ordering across all resource kinds (we will provide per-resource sequencing hints).

---

## Architecture: Target

- New package: `pkg/webhooks/notifier` providing a typed API:
  - `Notifier.Notify(ctx, Event) error` and async buffered sender with worker pool
  - HMAC-SHA256 signatures in `X-Kibaship-Signature`
  - Configurable `POST` URL(s), timeouts, retry policy, headers
  - JSON payload schema with event metadata (IDs, timestamps, resource refs)
- Controllers call Notifier on Create/Update/Delete and on key status transitions.
- Configuration from env/ConfigMap/Secret via `pkg/config/webhooks.go`:
  - `WEBHOOK_URL`, `WEBHOOK_HMAC_SECRET`, timeouts, max retries, backoff, concurrency, TLS opts (insecureSkipVerify=false by default)
- Observability: Prometheus counters/histograms + structured logs.

---

## Event Model

- Event minimal fields (idempotent, receiver-friendly):
  - `eventId` (uuid v4), `eventType` (Project|Application|Deployment + Created|Updated|Deleted|StatusChanged)
  - `timestamp` (RFC3339), `version` (monotonic per resource if available), `sequenceHint` (increasing number where applicable)
  - `resource`: `{ kind, apiVersion, name, namespace, uid }`
  - `correlation`: `{ projectUUID, applicationUUID, deploymentUUID }` when applicable
  - `specDelta` (optional): minimal changed fields when Update/StatusChanged
- Delivery semantics: at-least-once; receiver must de-duplicate by `eventId`.

---

## Workstreams and Task Checklists

### 0) Discovery and Freeze

- [ ] Identify all Valkey/Streams usages and entry points
  - Code:
    - `pkg/streaming/*` (interfaces, connection, redis client, publisher, tests)
    - `cmd/main.go` (Valkey connection init and publisher wiring)
    - `internal/controller/valkey_provisioner.go` (provisions Valkey CR)
  - Tests:
    - `test/e2e/infra_test.go` (checks Valkey pods)
    - `test/e2e/suite_test.go` (installs Valkey Operator during bootstrap)
  - Other:
    - Any references in docs/READMEs or kustomize overlays

### 1) Config and Secrets

- [ ] Add `pkg/config/webhooks.go` with strongly-typed settings
  - `WebhookURL string`
  - `HMACSecretRef { namespace, name, key }`
  - `RequestTimeout`, `ConnectTimeout`, `MaxRetries`, `InitialBackoff`, `MaxBackoff`, `MaxInFlight`, `TLS { InsecureSkipVerify, CABundleRef }`
- [ ] Kustomize: add Secret generator or static Secret placeholder for dev/e2e
  - `config/manager/kustomization.yaml`: pass env vars to manager
  - Add example Secret in `config/samples/`
- [ ] Document how to configure via Helm/kustomize and envs for CI/e2e
- [ ] Set sane defaults aligned with retryablehttp: `RetryMax=5`, `RetryWaitMin=500ms`, `RetryWaitMax=10s`, `MaxInFlight=8`

### 2) Notifier Implementation

- [ ] Create `pkg/webhooks/event.go` with event structs and JSON schema tags
- [ ] Create `pkg/webhooks/signature.go` for HMAC signing and header injection
- [ ] Create `pkg/webhooks/client.go` wrapping `github.com/hashicorp/go-retryablehttp`
  - Configure exponential backoff with jitter: `RetryMax`, `RetryWaitMin`, `RetryWaitMax`
  - Set HTTP timeouts and TLS options (CA bundle, InsecureSkipVerify=false by default)
  - Retry policy: retry on 408, 429, and all 5xx; do not retry other 4xx
- [ ] Create `pkg/webhooks/notifier.go`
  - Buffered channel input and worker pool for async delivery
  - Idempotent event headers: `X-Kibaship-Event-Id`, `X-Kibaship-Event-Type`, `X-Kibaship-Signature`
  - Metrics: `kibaship_webhook_events_total{result=success|retry|error}`, `kibaship_webhook_latency_seconds`
  - Structured logs for attempts and outcomes
- [ ] Unit tests covering success, retryable paths (429/5xx), non-retryable (4xx), signature generation, TLS failures

### 3) Controller Integration

- [ ] Wire `Notifier` into manager in `cmd/main.go`
  - Construct from config + secret at startup
  - Expose via a context or a shared component injected into reconcilers
- [ ] Update reconcilers to publish events
  - Project controller: on create/update/delete; on Ready status transitions
  - Application controller: create/update/delete; Ready transitions; domain changes
  - Deployment controller: create/update/delete; pipeline run status transitions
- [ ] Ensure events include resource UIDs and correlation fields
- [ ] Remove calls to streaming publisher; delete sequence logic tied to streams if not needed

### 4) Remove Valkey Provisioning and Client Code

- [ ] Delete/retire `internal/controller/valkey_provisioner.go` and its usage
- [ ] Remove `pkg/streaming/*` entirely (or mark deprecated behind feature flag, then delete)
- [ ] Remove Valkey connection initialization from `cmd/main.go`
- [ ] Update `go.mod` to remove Valkey/redis libraries; run `go mod tidy`
- [ ] Update docs that referenced Streams

### 5) E2E Test Refactor

- [ ] Remove Valkey operator installation from e2e bootstrap (`test/e2e/suite_test.go`)
- [ ] Remove infra assertions about Valkey pods (`test/e2e/infra_test.go`)
- [ ] Add webhook receiver test fixture:
  - Option A: In-process HTTP server in tests with port-forward from cluster (keeps test local)
  - Option B: Deploy a tiny HTTP receiver pod (e.g., `ghcr.io/neondatabase/http-echo` or custom) + Service; assert logs
  - Prefer A for speed in CI; add B as integration variant if needed
- [ ] Add tests that perform controller actions and assert webhook POSTs received with correct payload and signature
  - Project create/update/delete → expect matching events
  - Application create/update/delete and domain changes
  - Deployment create/update/delete and pipeline status transitions
- [ ] Negative tests: receiver returns 500 for N attempts → ensure retries and finally error metrics/log lines

### 6) Observability and Ops

- [ ] Expose Prometheus metrics from Notifier; update ServiceMonitor if present
- [ ] Add log sampling or verbosity guard for retries
- [ ] Add readiness/liveness impacts: Notifier should not block manager startup if URL misconfigured; instead set a Degraded condition and continue
- [ ] Add optional dead-letter logging after max retries

### 7) Security

- [ ] Implement HMAC-SHA256 signing using secret key
- [ ] TLS: support CA bundle via Secret; enforce TLS verification by default
- [ ] Redact secrets in logs; include only hashes/ids
- [ ] Document receiver verification steps (pseudo-code for signature check)

### 8) Migration/Compatibility

- [ ] No feature flags. Immediate removal of Streams/Valkey from operator and apiserver.
- [ ] One release drops all Streams code and Valkey provisioning.
- [ ] Clean removal of CRDs/resources related to Valkey provisioning; stop creating the system Valkey cluster. Leave any pre-existing user-managed Valkey clusters untouched.

### 9) Repository Hygiene

- [ ] Remove streaming unit tests: `pkg/streaming/redis_test.go`, `redis_test_impl.go`
- [ ] Add unit tests for webhook packages
- [ ] Update `README.md` and any dev docs

### 10) Acceptance Criteria (DoD)

- [ ] All unit tests and e2e tests pass locally and in CI
- [ ] No references to `pkg/streaming` remain
- [ ] No installation of Valkey operator in e2e
- [ ] Webhook receiver tests prove events and signatures are correct
- [ ] Metrics visible and increment on success/failure
- [ ] Security review: HMAC present; TLS enforced; secrets not logged

---

## File/Code Touch Points (Initial Inventory)

- Remove/Replace:
  - `pkg/streaming/interfaces.go`
  - `pkg/streaming/connection.go`
  - `pkg/streaming/redis.go`
  - `pkg/streaming/publisher.go`
  - `pkg/streaming/redis_test.go`
  - `pkg/streaming/redis_test_impl.go`
  - `internal/controller/valkey_provisioner.go`
  - `cmd/main.go` (Valkey init + publisher wiring)
- Add:
  - `pkg/webhooks/event.go`
  - `pkg/webhooks/signature.go`
  - `pkg/webhooks/client.go`
  - `pkg/webhooks/notifier.go`
  - `pkg/config/webhooks.go`
  - Kustomize env/secret wiring under `config/manager/`
- Tests:
  - Remove Valkey checks in `test/e2e/infra_test.go`
  - Remove Valkey operator bootstrap in `test/e2e/suite_test.go`
  - Add webhook receiver harness in e2e and associated assertions

---

## Rollout Plan

1. Phase 1 — Purge Streams/Valkey (now):
   - Remove `pkg/streaming/*`, `internal/controller/valkey_provisioner.go`, and any Valkey initialization in `cmd/main.go` and apiserver.
   - Refactor e2e: keep Valkey operator install, remove all Valkey/streaming assertions.
   - `go mod tidy`; run unit + e2e tests until green.
2. Phase 2 — Introduce Webhooks foundation:
   - Add webhook config and secrets; implement Notifier (retryablehttp + exponential backoff + HMAC + metrics).
   - Add webhook receiver harness and tests.
   - Wire Notifier; integrate Project controller events.
3. Phase 3 — Extend coverage:
   - Integrate Application and Deployment controllers; add additional event types; expand tests.
4. Phase 4 — Documentation and polish:
   - Update docs, examples, and release notes.

## Risks & Mitigations

- Receiver unavailability → robust retries with backoff; non-blocking manager.
- Payload drift → versioned payload schema; add `schemaVersion` if needed.
- Secrets misconfig → startup validation + clear error messages; health condition.
- Test flakiness → prefer in-process receiver for determinism; timeouts tuned.

---

## Quick Task List (Copy/Paste into Issues)

- [x] Purge Valkey/Streams from operator and apiserver (remove `pkg/streaming`, `internal/controller/valkey_provisioner.go`, Valkey init in `cmd/main.go`/apiserver); `go mod tidy`; ensure all tests pass
- [x] Refactor e2e: keep Valkey operator install, remove all Valkey/streaming assertions
- [ ] Add webhook config and secrets wiring
- [ ] Implement Notifier wrapping `github.com/hashicorp/go-retryablehttp` (HMAC, exponential backoff, metrics)
- [ ] Wire Notifier in manager; inject into reconcilers
- [ ] Emit events from Project controller
- [ ] Emit events from Application controller
- [ ] Emit events from Deployment controller
- [ ] Add webhook receiver tests and signature assertions
- [ ] Docs + examples updated

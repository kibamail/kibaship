# Kibaship Operator Code Review and Production Readiness Checklist

This document provides a comprehensive, actionable checklist to bring the Kibaship Operator to production-grade quality, with a focus on scalability, security, reliability, and maintainability. Each item is a task with a checkbox you can mark as completed.

## Executive summary
- [ ] Executive alignment: Define SLOs/SLAs (availability, latency for reconcile, max drift time), supported Kubernetes versions, and multi-cluster story
- [ ] Production defaults: Leader election ON, non-dev logging, tuned work queues, resource requests/limits sized via load tests
- [ ] Observability: Standard metrics, structured logs with correlation, health/readiness endpoints, tracing optional
- [ ] Testing: Unit, envtest, e2e, fuzz tests for webhooks; CI runs on PRs; scorecard passes

## Architecture and repository hygiene
- [ ] Document architecture and reconciliation flows (sequence diagrams) in operator.md
  - Description: Explain CRDs (Project, Application, Deployment), controllers, webhooks, and any external system interactions (Tekton). Link to code entrypoints: cmd/main.go
- [ ] Module and dependency policy
  - Description: Lock supported k8s/controller-runtime versions in go.mod; define upgrade policy and compatibility matrix
- [ ] Makefile targets completeness
  - Description: Ensure `make lint`, `make test`, `make build`, `make docker-buildx`, `make bundle`, `make scorecard` are green locally and in CI; remove unused targets

## API and CRD design (api/v1alpha1)
- [ ] Versioning and stability policy
  - Description: Define API guarantees for v1alpha1 and deprecation policy; plan for v1beta1/v1 promotion
- [ ] Validation via CRD schema complements webhook logic
  - Description: Ensure OpenAPI schema enforces required fields and formats; minimize custom runtime validations where CRD schema suffices
- [ ] Consistent labels/annotations contract
  - Description: Centralize label keys (e.g., platform.kibaship.com/*) as consts; document required vs optional; validate via webhooks
- [ ] Status fields are meaningful and converge on real state
  - Description: Status.Phase/Conditions should reflect underlying resources; avoid setting Ready unconditionally
- [ ] Naming conventions are consistent and documented
  - Description: Application/Deployment name regexes are codified and tested; provide helper functions and surface errors clearly

## Controllers (internal/controller)
- [ ] Reconcile idempotency and convergence
  - Description: Each Reconcile must be idempotent; update only when drift exists; use patch with strategic merge where appropriate
- [ ] Implement DeploymentReconciler logic
  - Description: Currently a TODO; define desired state (e.g., Tekton PipelineRun or Kubernetes resources), implement creation/update, status propagation, and finalizers
- [ ] ApplicationReconciler readiness is real
  - Description: Do not set status Phase=Ready unconditionally; derive readiness from dependent resources or validations
- [ ] Error handling and requeue strategy
  - Description: Use ctrl.Result with backoff for transient errors; distinguish retryable vs terminal errors; consider rate limiting
- [ ] MaxConcurrentReconciles and work queue tuning
  - Description: Configure controller builder WithOptions for concurrency; tune per-CRD
- [ ] Finalizers correctness
  - Description: Ensure finalizers guarantee cleanup ordering and are always removed; handle partial failures with retries
- [ ] OwnerReferences and garbage collection
  - Description: Where possible, set owner refs to leverage GC; for cross-scope constraints (Namespace vs namespaced CRs), document strategy
- [ ] Robust patch helpers and deep-equality checks
  - Description: Use controller-runtime CreateOrUpdate / Patch; avoid unnecessary updates to prevent hot loops

## Namespace and RBAC management (internal/controller/namespace.go, config/rbac)
- [ ] Restrict per-namespace Role rules
  - Description: Creating a Role with `Verbs: ["*"]`, `Resources: ["*"]` grants full admin in project namespaces. Reassess security model; prefer a curated set of verbs/resources or use aggregate roles
- [ ] Labeling and ownership tracking
  - Description: Keep ManagedBy and project identifiers, but add soft ownership via annotations; ensure label collisions are handled
- [ ] Namespace uniqueness and rename policy
  - Description: Current logic checks existence; document rename behavior (likely disallowed) and add tests for conflicts

## Webhooks (config/webhook, api/v1alpha1/*_types.go)
- [ ] Validate webhooks availability and cert management
  - Description: Ensure webhook service, certs, and CA bundles are provisioned (via cert-manager or self-signed patch). Add readiness gate for webhooks
- [ ] Fuzz tests for validators
  - Description: Add fuzzing for regex-based name parsing and UUID checks to avoid panics/edge cases
- [ ] FailurePolicy and sideEffects
  - Description: FailurePolicy=Fail is fine; ensure webhooks are highly available and fast; document SLO
- [ ] Idempotent validation helpers
  - Description: Consolidate duplicate isValidUUID; centralize in a shared util pkg to avoid drift

## Operator configuration and runtime (cmd/main.go, config/manager)
- [ ] Leader election enabled by default in production
  - Description: Default leader-elect to true via args or env; document how to disable for dev
- [ ] Logging configuration
  - Description: Use zap options with Development=false in production; support log level/env var; include request IDs and reconcile keys
- [ ] Health/ready endpoints and probes
  - Description: Already present; ensure timeouts/thresholds are tuned; add a metrics service/ServiceMonitor for scraping
- [ ] Resource sizing and limits
  - Description: Validate limits/requests under expected load; provide sizing guidance
- [ ] Concurrency configuration via env
  - Description: Expose per-controller concurrency flags/env vars; wire into builder options

## Observability (metrics, logging, tracing)
- [ ] Expose Prometheus metrics
  - Description: Use controller-runtime metrics register; provide counters for reconciles, errors, queue length, duration histograms
- [ ] Structured contextual logs
  - Description: Consistent key/value pairs: controller, resource GVK/NN, generation, retry count
- [ ] Optional tracing
  - Description: Add OpenTelemetry hooks optionally; trace reconcile spans including external calls (e.g., Tekton)

## Security and multi-tenancy
- [ ] Drop cluster-scoped writes for optional integrations
  - Description: Avoid creating cluster namespaces/roles that are not strictly needed; rely on platform install prereqs
- [ ] ServiceAccount and Pod security
  - Description: Keep restricted profile; avoid host mounts; pin distroless base with digest; add imagePullPolicy and seccomp
- [ ] Secrets handling
  - Description: Never log secrets; validate SecretRef existence with least privileges; consider namespacing and projected service accounts
- [ ] Supply chain
  - Description: Enable SBOM, image signing (cosign), provenance; pin base images with digests

## Scalability and performance
- [ ] Event filters and predicates
  - Description: Add predicates to ignore status-only updates and no-op changes; reduce reconcile churn
- [ ] Informers and cache usage
  - Description: Ensure efficient List/Watch usage and indexed fields for label selectors used frequently
- [ ] Backpressure and rate limits
  - Description: Configure workqueue rate limiters (exponential) and max in-flight reconciles
- [ ] Large-scale namespace management
  - Description: For many projects, batch/parallelize namespace ops; consider admission-based controls instead of controller doing heavy-lifting

## Testing strategy (internal/controller/*_test.go, test/e2e)
- [ ] Unit tests coverage targets
  - Description: Aim for >=80% on controllers/util; cover error paths and idempotency
- [ ] Envtest integration tests
  - Description: Extend suite to start manager with controllers and webhooks; assert reconcile outcomes and status conditions
- [ ] E2E tests with Kind
  - Description: Use test/e2e for install->CR lifecycle, webhook behavior, RBAC sanity; include negative tests
- [ ] Webhook fuzz and property tests
  - Description: Fuzz name/UUID validators, boundary lengths; ensure regexes don’t cause exponential backtracking
- [ ] Race detector and leak checks
  - Description: Run `-race` and goroutine leak checks in CI for reconcile path

## Build, release, and distribution
- [ ] Multi-arch images with buildx
  - Description: Ensure docker-buildx target is used in CI; publish arm64/amd64
- [ ] Image tagging and immutability
  - Description: Use semantic versions; digest pinning in kustomize; avoid latest
- [ ] OLM bundle and catalog
  - Description: Validate bundle; include accurate RBAC and webhook CABundle injection; scorecard passes
- [ ] Helm chart (optional)
  - Description: Offer a Helm chart with values for flags (leader election, concurrency, feature gates)

## CI/CD and quality gates
- [ ] GitHub Actions (or equivalent) pipeline
  - Description: Jobs for lint (golangci-lint), unit/envtest, e2e (Kind), buildx, SBOM, cosign attest, bundle build/validate
- [ ] PR gates and required checks
  - Description: Require all tests, lint, and scorecard to pass; enforce DCO/signoff if desired
- [ ] Static analysis and vuln scanning
  - Description: Gosec, trivy/grype on images, license scanning

## Documentation and samples (README.md, config/samples)
- [ ] Fill README with clear getting-started and production guidance
  - Description: Replace TODOs; add requirements, limits, upgrade notes, and troubleshooting
- [ ] Samples aligned with validation
  - Description: Ensure sample names conform to webhook regex; include Project, Application, Deployment happy-path and negative examples
- [ ] Operator configuration matrix
  - Description: Document config flags/env vars, feature gates, and defaults

## Backward compatibility and upgrades
- [ ] CRD conversion and webhook for future versions
  - Description: Plan for v1beta1 with conversion strategy; add max skew supported and deprecation windows
- [ ] Safe upgrades and rollback
  - Description: Document helm/kustomize upgrade steps; use leader-election and readiness to ensure no downtime

## Concrete refactor and implementation tasks
  - Description: Tighten config/rbac/role.yaml and code paths that require cluster-scoped access
- [ ] Implement DeploymentReconciler end-to-end
  - Description: Define desired Tekton or K8s resources, reconcile creates/updates, watch related objects, maintain status conditions
- [ ] Make Application status meaningful
  - Description: Populate conditions based on referenced Project readiness and any type-specific validations
- [ ] Centralize label keys and validators
  - Description: Create a shared package for labels/regex/UUID; import from controllers and webhooks
- [ ] Add controller concurrency and predicates
  - Description: Use builder options and predicates to reduce no-op reconciles
- [ ] Add metrics and instrument reconciles
  - Description: Histogram for reconcile duration, counters for successes/failures; expose Service/ServiceMonitor
- [ ] Harden Dockerfile and image
  - Description: Pin distroless digest, set non-root UID/GID, drop shells/tools, set user explicitly, add .dockerignore
- [ ] Production defaults in cmd/main.go
  - Description: leader-elect=true default, Development=false, configurable probe addr via env
- [ ] Extend tests (unit, envtest, e2e)
  - Description: Cover namespace manager, RBAC creation paths, webhook regex edge cases, finalizers, requeues

## Nice-to-have enhancements
- [ ] OpenAPI published docs for CRDs
  - Description: Generate docs site with examples; improve UX for users
- [ ] Telemetry opt-in
  - Description: Anonymous usage metrics to guide improvements (opt-in, documented)
- [ ] Controller feature-gates
  - Description: Use feature flags for risky/intrusive features; enable progressive rollout

---

### File-level notes and quick wins
- [ ] cmd/main.go: set leader election true by default; configure concurrency; Development=false logs
- [ ] internal/controller/deployment_controller.go: implement reconcile; add tests
- [ ] internal/controller/application_controller.go: make Ready conditional; ensure error paths set conditions; avoid unconditional updates
- [ ] internal/controller/namespace.go: remove "*" roles; add feature gate for Tekton; avoid creating tekton-pipelines namespace by default
- [ ] api/v1alpha1/*: centralize isValidUUID; ensure CRD schema complements webhook; add JSON schema defaults where appropriate
- [ ] config/manager/manager.yaml: add Service for metrics; ensure resource limits are tuned; add imagePullPolicy: IfNotPresent/Always as policy dictates


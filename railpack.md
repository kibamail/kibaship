## Railpack-based container build plan (Tekton)

This document proposes a production-grade design to build container images from cloned application sources using Railpack and BuildKit in Tekton, at high scale and multi‑tenant. It follows Railpack guidance from "Running Railpack in Production" and adapts it to Kubernetes/Tekton.

### Goals

- Add a custom Tekton Task to prepare and build images using Railpack’s BuildKit frontend
- Support public and private repos (already cloned by previous task) with build secrets
- Push images to OCI registries (GHCR/ECR/GCR), produce SBOM/attestations, and emit webhooks
- Scale to thousands of concurrent builds safely and cost‑efficiently

### Key references (from Railpack docs)

- Prefer custom BuildKit frontend image: ghcr.io/railwayapp/railpack-frontend (version should match plan)
- Use `railpack prepare` to generate: build plan (plan.json) and build info (info.json)
- Pass secrets to prepare (names only) via `--env`, and mount actual values to build via BuildKit `--secret`
- Provide `secrets-hash` build arg to force layer invalidation when secrets change
- Provide `cache-key` build arg to isolate caches in multi-tenant environments

---

## Implementation progress (2025-09-23)

- [x] Created lightweight Railpack CLI image (v0.7.2)

  - Path: build/railpack-cli/Dockerfile; Alpine base, non-root user, includes ca-certificates for HTTPS
  - Publishes to: ghcr.io/kibamail/kibaship-railpack-cli:<tag>
  - Excerpt:

  ```dockerfile
  FROM alpine:3.20
  RUN apk add --no-cache ca-certificates
  ENTRYPOINT ["/usr/local/bin/railpack"]
  ```

- [x] CI split into per-image workflows with gating on Tests/Lint/E2E

  - Files: .github/workflows/build-operator.yml, build-apiserver.yml, build-cert-manager-webhook.yml, build-railpack-cli.yml
  - Triggered on workflow_run for ["Tests", "E2E Tests", "Lint"] (success, main) and on push tags v\*
  - Excerpt:

  ```yaml
  on:
    workflow_run:
      workflows: [Tests, E2E Tests, Lint]
      types: [completed]
  ```

- [x] Release script validates Railpack image build

  - scripts/release.sh now builds railpack-cli locally and fails early if it breaks

  ```bash
  log_info "Validating Docker build (railpack-cli)..."
  docker build -f build/railpack-cli/Dockerfile -t kibaship-railpack-cli:validate build/railpack-cli
  ```

- [x] Prepare step image selection and usage updated

  - Use ghcr.io/kibamail/kibaship-railpack-cli:v0.7.2 for the Tekton prepare step

  ```yaml
  steps:
    - name: prepare
      image: ghcr.io/kibamail/kibaship-railpack-cli:v0.7.2
      workingDir: $(workspaces.output.path)/repo/$(params.contextPath)
      command: ["railpack"]
      args:
        [
          "prepare",
          "--plan-out",
          "$(results.planPath.path)",
          "--info-out",
          "$(results.infoPath.path)",
          "$(workspaces.output.path)/repo/$(params.contextPath)",
        ]
  ```

- [x] Rationale for ca-certificates in the image
  - Ensures TLS verification works for any HTTPS calls during prepare (e.g., metadata lookups)

---

## Implementation updates (2025-09-24)

- Tekton Railpack Prepare Task implemented and applied

  - File: config/tekton-resources/tasks/platform.operator.kibaship.com_railpack_prepare_tasks.yaml
  - Params:
    - contextPath (string, default ".") — set from Application.spec.gitRepository.rootDirectory
    - railpackVersion (string, default "0.1.2") — selects the railpack-cli image tag
    - envArgs (string, default "") — optional extra args passed to `railpack prepare` (e.g., `--env FOO=$FOO`)
  - Workdir: `$(workspaces.output.path)/repo/$(params.contextPath)` (clone task writes into `repo/`)
  - Outputs: plan and info written to workspace root under `/railpack/plan.json` and `/railpack/info.json`; Tekton results expose absolute paths

- Operator wiring

  - The operator passes Application.spec.gitRepository.rootDirectory to the prepare task as `contextPath`.
  - RootDirectory is interpreted relative to the repository root. The clone step writes to `workspace/repo`, so the task uses `repo/<RootDirectory>` as working directory.
  - The GitRepository.Path field is currently unused and slated for removal.

- railpack-cli image adjustments

  - Switched base to Debian bookworm-slim to avoid glibc/musl issues seen with Alpine in CI/e2e.
  - Includes CA certificates and runs as a non-root user.
  - The image bundles the Railpack binary (currently v0.7.2 musl build). The Tekton `railpackVersion` parameter selects the container image tag, not the internal binary version.

- E2E test updates and verification

  - Switched test application to the following repository and commit:
    - Repository: https://github.com/railwayapp/railpack (public)
    - Commit SHA: 960ef4fb6190de6aa8b394bf2f0d552ee67675c3
    - RootDirectory: `examples/node-next`
  - Ensured prepare runs inside the cloned repo by using workingDir `$(workspaces.output.path)/repo/$(params.contextPath)`.
  - Implemented local image override for tests: build railpack-cli, load into kind, kustomize edit image, and patch the Tekton Task image post-apply.
  - Full e2e suite passes (10/10): project/application/deployment CRUD, domains, Tekton pipeline provisioning, and prepare task success.

- Known notes/quirks
  - `railpackVersion` chooses the railpack-cli image tag; the bundled Railpack binary version is pinned in the Dockerfile and may differ in naming.
  - RootDirectory is the only field used to select the working directory. Path is unused and will be removed.

## Architecture overview

- Pipeline sequence:
  1. git-clone (existing)
  2. railpack-prepare (generate plan/info)
  3. railpack-build (BuildKit-driven image build + push)
- Tekton Task model:
  - Single custom Task `railpack-prepare-build` with two steps OR two separate Tasks connected via Pipeline (preferred for clarity and caching)
  - BuildKit runs rootless via sidecar `moby/buildkit` (buildkitd) with step `buildctl` client
- Storage & caching:
  - Workspace PVC for source and plan artifacts
  - Optional persistent cache PVC or registry cache for BuildKit
- Security:
  - Rootless BuildKit; restricted PodSecurity; runAsNonRoot; readOnlyRootFilesystem where possible
  - Registry auth via dedicated per-tenant secrets mounted as DOCKER_CONFIG
- Observability:
  - Tekton Results for image reference, digest, plan path, build info
  - Emit webhook events with build metadata (duration, image, digest, SBOM link)

---

## New Tekton resources (high-level)

- Task: `platform.operator.kibaship.com_railpack_prepare_task.yaml`
- Task: `platform.operator.kibaship.com_railpack_build_task.yaml`
- Optional combined Task: `platform.operator.kibaship.com_railpack_prepare_build_task.yaml`
- Pipeline changes: extend deployment pipeline to include prepare → build after clone

### Parameters (common)

- contextPath: relative path of app within workspace (default: ".")
- railpackVersion: pin CLI/frontend version (e.g., vX.Y.Z) to align plan and frontend
- imageRegistry: e.g., ghcr.io
- imageRepository: e.g., kibamail/<app>
- imageTag: default commit SHA; optionally branch, timestamp, semver
- platforms: e.g., linux/amd64[,linux/arm64]
- cacheKey: tenant/project key for mount cache isolation
- buildSecrets: list of secret names to mount at build (e.g., NPM_TOKEN, STRIPE_LIVE_KEY)
- additionalBuildArgs: arbitrary list of `key=value`

### Workspaces

- output (required): contains cloned repo and produced plan/info; shared between steps
- build-cache (optional): persistent cache PVC for BuildKit

### Results

- image: fully qualified image reference pushed
- digest: OCI digest
- planPath: path to plan.json
- infoPath: path to info.json
- sbomRef: reference/URL to SBOM if published

---

## Step design

### Step 1: railpack-prepare

- Image: custom builder image with railpack CLI (or official if provided)
- Command:
  - `railpack prepare $CONTEXT --plan-out plan.json --info-out info.json` plus `--env` for each secret name (no values)
- Inputs:
  - context: $(workspaces.output.path)/$(params.contextPath)
  - buildSecrets names via params; pass `--env NAME=$NAME` (names only) so plan includes required secret IDs
- Outputs:
  - plan.json, info.json in workspace
- Version pinning:
  - Use $(params.railpackVersion) to select CLI version (image tag)

Example Tekton Task (railpack-prepare):

```yaml
apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: railpack-prepare
spec:
  params:
    - name: contextPath
      default: "."
    - name: railpackVersion
      default: "v0.7.2"
    - name: buildSecrets
      type: array
      default: []
  workspaces:
    - name: output
  results:
    - name: planPath
    - name: infoPath
  steps:
    - name: prepare
      image: ghcr.io/kibamail/kibaship-railpack-cli:$(params.railpackVersion)
      workingDir: $(workspaces.output.path)/$(params.contextPath)
      script: |
        #!/bin/sh
        set -eu
        ARGS=""
        for s in $(params.buildSecrets); do ARGS="$ARGS --env $s=$s"; done
        railpack prepare . \
          --plan-out plan.json \
          --info-out info.json $ARGS
        printf "%s" "$(pwd)/plan.json" > $(results.planPath.path)
        printf "%s" "$(pwd)/info.json" > $(results.infoPath.path)
```

### Step 2: railpack-build (BuildKit)

- Sidecar: `moby/buildkit:rootless` (buildkitd)
  - SecurityContext: runAsUser non-root; allow necessary mounts; adjust sysctls if needed for rootless
  - Expose buildkitd to `buildctl` client via TCP or Unix socket within pod (use env BUILDKIT_HOST=tcp://localhost:1234)
- Client step: `moby/buildkit:rootless` (contains buildctl) or `tonistiigi/buildkit` tool image
- Auth:
  - Mount DOCKER_CONFIG secret with registry creds at `/kaniko/.docker` (or `/tekton/home/.docker`) and set `DOCKER_CONFIG`
- Secrets:
  - For each secret name in params, mount value from Kubernetes Secret as a file (`/secrets/<NAME>`) and pass `--secret id=<NAME>,src=/secrets/<NAME>`
- Build args:
  - `--opt source=ghcr.io/railwayapp/railpack:railpack-frontend` (matching $(params.railpackVersion))
  - `--build-arg BUILDKIT_SYNTAX=ghcr.io/railwayapp/railpack-frontend`
  - `--build-arg secrets-hash=$(sha256 of concatenated NAME=VALUE pairs)`
  - `--build-arg cache-key=$(params.cacheKey)`
  - Pass $(params.additionalBuildArgs)
- Caching:
  - Use `--export-cache type=inline` and/or `--import-cache type=registry,ref=<cache-image>`
  - Optionally mount persistent volume as BuildKit cache directory
- Multi-arch:
  - If multi-arch, run QEMU binfmt on nodes (pre-installed by ops) and use `--opt platforms=$(params.platforms)` with `--output type=image,push=true,name=$(imageRef),oci-mediatypes=true` and `--provenance=mode=max` for attestations
- Outputs:
  - Image pushed to $(params.imageRegistry)/$(params.imageRepository):$(params.imageTag)
  - Capture digest from buildctl output; write to Tekton result
  - Optionally push SBOM as OCI artifact (Syft/BuildKit provenance attestation)

Example Tekton Task (railpack-build):

```yaml
apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: railpack-build
spec:
  params:
    - name: imageRegistry
    - name: imageRepository
    - name: imageTag
    - name: platforms
      default: linux/amd64
    - name: cacheKey
      default: ""
    - name: railpackVersion
      default: "v0.7.2"
    - name: buildSecrets
      type: array
      default: []
  workspaces:
    - name: output
    - name: build-cache
      optional: true
  results:
    - name: image
    - name: digest
  sidecars:
    - name: buildkitd
      image: moby/buildkit:rootless
      args: ["--addr", "tcp://0.0.0.0:1234"]
      securityContext:
        runAsUser: 1000
        runAsNonRoot: true
      ports:
        - containerPort: 1234
  stepTemplate:
    env:
      - name: BUILDKIT_HOST
        value: tcp://localhost:1234
      - name: DOCKER_CONFIG
        value: /tekton/home/.docker
  steps:
    - name: build
      image: moby/buildkit:rootless
      workingDir: $(workspaces.output.path)
      volumeMounts:
        - name: docker-config
          mountPath: /tekton/home/.docker
        - name: build-secrets
          mountPath: /secrets
          readOnly: true
      script: |
        #!/bin/sh
        set -eu
        IMG="$(params.imageRegistry)/$(params.imageRepository):$(params.imageTag)"
        PLAN="$(workspaces.output.path)/plan.json"
        # Compute secrets-hash (sorted NAME=VALUE lines)
        SH=""
        if [ -d /secrets ] && ls /secrets/* >/dev/null 2>&1; then
          SH=$( (for f in /secrets/*; do n=$(basename "$f"); printf "%s=%s\n" "$n" "$(cat "$f")"; done) | sort | sha256sum | awk '{print $1}')
        fi
        buildctl build \
          --local context=$(workspaces.output.path) \
          --local dockerfile=$(workspaces.output.path) \
          --frontend=gateway.v0 \
          --opt source=ghcr.io/railwayapp/railpack:railpack-frontend \
          --opt filename=$(basename "$PLAN") \
          --opt build-arg:BUILDKIT_SYNTAX=ghcr.io/railwayapp/railpack-frontend \
          --opt build-arg:cache-key=$(params.cacheKey) \
          --opt build-arg:secrets-hash=$SH \
          $(for s in $(params.buildSecrets); do printf -- "--secret id=%s,src=/secrets/%s " "$s" "$s"; done) \
          --output type=image,name=$IMG,push=true,oci-mediatypes=true \
          --provenance mode=max | tee /tekton/home/buildctl.out >/dev/null
        # Extract digest (prefer registry query)
        if command -v skopeo >/dev/null 2>&1; then
          DIGEST=$(skopeo inspect docker://$IMG | jq -r .Digest || true)
        fi
        [ -n "${DIGEST:-}" ] || DIGEST=$(grep -Eo 'sha256:[0-9a-f]+' /tekton/home/buildctl.out | head -n1 || true)
        printf "%s" "$IMG" > $(results.image.path)
        printf "%s" "${DIGEST:-unknown}" > $(results.digest.path)
  volumes:
    - name: docker-config
      secret:
        secretName: registry-docker-config
        optional: false
    - name: build-secrets
      projected:
        sources:
          - secret:
              name: build-secret-bundle
              optional: true
```

---

## Image naming & tagging

- Name pattern: `$(registry)/kibaship/$(projectSlug)/$(appSlug):$(commitSHA)`
- Additional tags:
  - `:branch-<branch>` (mutable), `:latest` (per environment), promotion tags (`:staging`, `:prod`)
  - Write immutable digest pin to Deployment status for rollout
- OCI annotations & labels:
  - org.opencontainers.image.source, revision, created, authors
  - Railpack metadata from info.json

---

## Secrets handling (prepare vs build)

- Prepare: include secret NAMES via `railpack prepare --env NAME=$NAME` so plan records which secrets are needed
- Build: mount secret VALUES via Tekton and pass with buildctl `--secret id=NAME,src=/secrets/NAME`
- Layer invalidation: compute `secrets-hash` as sha256 of sorted `NAME=VALUE` joined by `\n`; pass as `--build-arg secrets-hash=...`
- Tenancy isolation: include `--build-arg cache-key=$(projectUuid or tenantKey)`

Example: computing secrets-hash inside the build step

```sh
#!/bin/sh
set -eu
dir=/secrets
if [ ! -d "$dir" ] || ! ls "$dir"/* >/dev/null 2>&1; then
  echo ""; exit 0
fi
( for f in "$dir"/*; do n=$(basename "$f"); printf "%s=%s\n" "$n" "$(cat "$f")"; done ) \
  | sort | sha256sum | awk '{print $1}'
```

---

## Observability & webhooks

- Tekton results: image, digest, build duration, planPath, infoPath
- Emit deployment webhook on build completion including:
  - status (Succeeded/Failed), image ref, digest, build logs URL, info
- Metrics: record counts, durations, cache hit rate; expose via Prometheus
- Logs: standardize step logs; attach log links to webhooks

---

## Security hardening

- Rootless BuildKit; readOnlyRootFilesystem where possible
- Drop capabilities; seccompProfile RuntimeDefault; runAsNonRoot; fsGroup where needed
- DOCKER_CONFIG secret is mounted read-only; per-tenant SA & RBAC
- Scan built images with Trivy; fail gates configurable; publish reports
- Cosign signing & provenance attestations; keyless (OIDC) if supported by registry
- NetworkPolicies limiting egress to allowed registries and git providers

---

## Scalability strategy

- Horizontal scaling: Tekton controllers and worker nodes; nodeSelector/taints for build nodes
- Concurrency control: queue builds per tenant; rate limit pushes to registry
- Caching: registry-based cache to share across workers; per-tenant cache-key to avoid cross-tenant bleed
- Deduplication: skip build if same commit+plan+secrets-hash already produced digest (lookup in artifact index)
- TTLAfterFinished on TaskRuns; prune old PVCs and caches per policy

---

## Failure handling & retries

- Retry policy for transient network/registry errors (HTTP 5xx, timeouts)
- Clear error surface: distinguish plan generation errors vs build errors
- Fallbacks: if multi-arch not available, build single-arch and mark degraded
- Timeouts: per step and overall; cancel BuildKit gracefully

---

## SBOM, provenance, and compliance

- Generate SBOM (Syft) step after build or via BuildKit attestation; attach to registry (OCI referrer)
- Cosign sign image and attest SBOM/provenance (SLSA provenance from BuildKit `--provenance=mode=max`)
- Policy: Only signed images admitted to prod; validate on deploy
- Data residency: choose regional registries per tenant if required

---

## Integration into current pipeline

- Extend `createGitRepositoryPipeline` to append:
  - Task `railpack-prepare` (consumes workspace from clone)
  - Task `railpack-build` (consumes plan; produces image)
- Extend PipelineRun watcher to update Deployment with image+digest and include build results in webhooks
- RBAC: grant build tasks read access to needed secrets and push privileges to registry

Example Pipeline wiring (YAML fragment):

```yaml
apiVersion: tekton.dev/v1
kind: Pipeline
spec:
  workspaces:
    - name: ws
  params:
    - name: git-commit
  tasks:
    - name: clone
      taskRef:
        resolver: cluster
        params:
          - name: kind
            value: task
          - name: name
            value: tekton-task-git-clone-kibaship-com
          - name: namespace
            value: tekton-pipelines
      workspaces:
        - name: output
          workspace: ws
    - name: prepare
      runAfter: ["clone"]
      taskRef:
        name: railpack-prepare
      params:
        - name: contextPath
          value: .
        - name: railpackVersion
          value: vX.Y.Z
        - name: buildSecrets
          value:
            - NPM_TOKEN
      workspaces:
        - name: output
          workspace: ws
    - name: build
      runAfter: ["prepare"]
      taskRef:
        name: railpack-build
      params:
        - name: imageRegistry
          value: ghcr.io
        - name: imageRepository
          value: kibamail/$(context.pipelineRun.name)
        - name: imageTag
          value: $(params.git-commit)
        - name: railpackVersion
          value: vX.Y.Z
      workspaces:
        - name: output
          workspace: ws
```

---

## Testing & validation

- E2E in kind: mock registry (registry:2) with auth; single-arch builds; minimal secrets
- Conformance: sample apps across languages Railpack supports
- Load tests: N concurrent builds, measure success rate, p95 duration, registry errors
- Chaos tests: registry throttling, DNS failures, secret rotation

---

## Rollout plan

- Phase 1: Implement tasks and run in staging; manual trigger builds
- Phase 2: Wire into deployment pipeline; gated behind feature flag
- Phase 3: Enable for select tenants; monitor; scale nodes; enable caches
- Phase 4: Enable multi-arch and signing in production; enforce policies

---

## Risks & mitigations

- Registry rate limits → backoff, token bucket, dedicated org
- Cache poisoning → per-tenant cache-key and auth; immutable tags
- Secret leakage → strict mount paths, no echoing in logs, ephemeral volumes
- Buildkitd resource starvation → resource requests/limits; separate node pool

---

## Detailed Task Checklist

### Task A: Railpack prepare (new Task)

- [ ] Create Task spec with params: contextPath, railpackVersion, buildSecrets (array)
- [ ] Mount workspace `output`
- [x] Choose railpack CLI image or build custom with pinned version
- [x] Created custom image build/railpack-cli/Dockerfile pinned to v0.7.2 and CI publishing via build-railpack-cli.yml

- [ ] Implement step to run `railpack prepare` with `--plan-out` and `--info-out`
- [ ] Inject `--env NAME=$NAME` for each secret name (no values)
- [ ] Emit Tekton results: planPath, infoPath
- [ ] Resource requests/limits; securityContext

### Task B: Railpack build (new Task)

- [ ] Params: imageRegistry, imageRepository, imageTag, platforms, cacheKey, railpackVersion, additionalBuildArgs
- [ ] Workspaces: output (read plan), build-cache (optional)
- [ ] Sidecar buildkitd rootless; configure BUILDKIT_HOST
- [ ] Mount DOCKER_CONFIG secret for registry auth
- [ ] Mount build secrets at /secrets and map to buildctl `--secret`
- [ ] Compute `secrets-hash` and pass as build arg
- [ ] Run buildctl with railpack frontend, using plan json as dockerfile
- [ ] Push to registry; capture digest; write Tekton results (image, digest)
- [ ] Optional: export-cache/import-cache; SBOM/provenance

### Task C: Pipeline wiring

- [ ] Update Deployment pipeline generation to add prepare → build after clone
- [ ] Wire params from Deployment/Application (registry, repo, tags)
- [ ] Add ServiceAccount and RBAC for registry push and secret read
- [ ] Update watcher to set Deployment image+digest; include build info in webhooks

### Task D: Ops enablement

- [ ] Provision builder node pool; binfmt for multi-arch (if needed)
- [ ] Registry org/project and policies; robot accounts; rate limits
- [ ] Monitoring dashboards and alerts (error rate, duration, cache hit)
- [ ] Retention policies for PVC caches and TaskRuns

### Task E: Security & compliance

- [ ] Rootless containers; PodSecurity; read-only FS
- [ ] Cosign signing + attestations; verification on deploy
- [ ] Image scanning with Trivy; policy gates
- [ ] Secret management rotation procedures

### Task F: Testing

- [ ] E2E builds across sample apps; verify image runs
- [ ] Load tests with parallel builds
- [ ] Failure injection (registry/network)
- [ ] Documentation for tenants (custom needs, secrets, build args)

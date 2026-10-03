# AGENTS.md

This file provides guidance to AI coding agents working with code in this repository.

## Project Overview

This is `gpu-booking`, a community plugin for the **Red Hat OpenShift AI (RHOAI) Dashboard** that provides GPU resource reservation with Kueue integration. It uses Webpack 5 Module Federation to expose remote modules that the RHOAI dashboard host application loads at runtime, plus a Go API backend for bookings and Kueue resource management.

## Git Rules (IMPORTANT)

- **Mike is the git master.** NEVER run `git commit`, `git push`, `git merge`, or any other git state-changing command. Only Mike touches git.
- Do not create branches, tags, or PRs. Leave the working tree dirty for Mike to review and commit.

## Build & Development Commands

Use the Makefile (`make help` for all targets):

```bash
make build          # frontend production build to dist/
make build-dev      # frontend dev build (webpack dev mode)
make test           # frontend (vitest) + Go tests
make test-go        # go test ./pkg/...
make lint           # frontend lint
make lint-chart     # helm lint chart/
make typecheck      # frontend TypeScript check
make validate       # typecheck + lint + tests + chart lint
make dev            # frontend dev server (port 9500)
make dev-bff        # Go backend locally on :3000 (DEV_MODE=true, plain HTTP)
make image          # build frontend + backend container images (BUILDER=podman)
make image-push     # build and push both images to quay.io/rh-ai-community-plugins
make chart-push     # package and push Helm chart to OCI registry
make clean          # remove dist/, bin/, coverage/
```

Frontend scripts live in `package.json` (`npm run build`, `npm test`, `npm run lint`, ...).

## Architecture

### Module Federation Plugin System

The plugin exposes two remote modules to the RHOAI dashboard host:

- **`./extensions`** (`gpuBooking/extensions`) — extension points for the feature area, navigation, and route at `/gpu-booking`
- **`./Icon`** (`gpuBooking/Icon`) — SVG icon for the sidebar nav entry

Module Federation container name and scope: **`gpuBooking`** (must match `plugin.yaml` and the `MODULE_FEDERATION_CONFIG` entry on `deployment/rhods-dashboard`).

### Pages

The frontend routes under `/gpu-booking/*` (booking calendar, admin page, help). The Go backend serves `/api/*` on port 3000 behind the dashboard's proxy (`/gpu-booking/api` → `gpu-booking-bff:3000`, Bearer token forwarded).

### Backend (Go)

- `cmd/backend/main.go` — entry point (PORT env, default 9443 locally; the chart sets 3000). Requires `DEV_MODE=true` for plain HTTP outside a cluster.
- `pkg/api/` — HTTP handlers, auth middleware (TokenReview + SubjectAccessReview against the user's Bearer token), rate limiting.
- `pkg/database/` — SQLite persistence (`mattn/go-sqlite3`, CGO required) on `DB_PATH` (PVC `/app/data/bookings.db` in-cluster).
- `pkg/kube/` — K8s API access with the ServiceAccount token: Kueue ClusterQueue/LocalQueue/Cohort/HardwareProfile management, GPU node discovery, workload listing.

### Deployment

- **Frontend container**: multi-stage `Containerfile` — UBI9 Node 22 builder → UBI9 Nginx 1.24 serving `dist/` on 8080 as UID 1001, CORS header on `remoteEntry.js`.
- **Backend container**: multi-stage `Containerfile.bff` — UBI9 go-toolset 1.25 builder (CGO_ENABLED=1) → UBI9 minimal runtime on 3000 as UID 1001, with `sqlite-libs`.
- **Helm chart**: `chart/` deploys both services into `cp-gpu-booking` (configurable via `namespace`), plus a PVC for SQLite, an optional GPU config ConfigMap, and a `gpu-booking-bff` ClusterRole/ClusterRoleBinding (TokenReview/SAR create + Kueue + HardwareProfiles + nodes/namespaces).
- Chart name is `gpu-booking-chart` (with `-chart` suffix to avoid OCI registry collisions with the frontend image repo); `nameOverride: gpu-booking` keeps deployed resources named `gpu-booking` / `gpu-booking-bff`.
- Dashboard registration: set the literal `MODULE_FEDERATION_CONFIG` env var on `deployment/rhods-dashboard` (the operator reverts ConfigMap edits). See `docs/deployment/OPENSHIFT_DEPLOY.md`.

### Version Sync

`scripts/sync-chart-version.js` syncs the version from root `package.json` into `chart/Chart.yaml` (`version` + `appVersion`), `plugin.yaml` (`version` + image/bff image `tag`), and the `--version` flag in docs.

## Key Conventions

- Path alias: `~` maps to `./src` (webpack); use `~` in frontend source imports.
- UI components use **PatternFly 6** (`@patternfly/react-core`, `@patternfly/react-icons`).
- Module Federation name/scope is **`gpuBooking`** — keep `plugin.yaml`, `config/`, and dashboard registration in sync.
- TypeScript strict mode for the frontend; Go 1.25 for the backend.
- Helm templates follow the superset-console-plugin pattern: `_helpers.tpl` name/label helpers, namespace skipped when it equals the release namespace, `helm.sh/resource-policy: keep` on the PVC.
- RBAC derives from the backend code — if you add K8s API calls in `pkg/`, update `chart/templates/bff-rbac.yaml` and the RBAC section in `plugin.yaml`.

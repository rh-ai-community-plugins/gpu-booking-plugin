# GPU Booking Plugin

A community plugin for the **Red Hat OpenShift AI (RHOAI) Dashboard** that provides GPU resource reservation with Kueue integration. Users reserve GPU quota (full GPUs or MIG slices) for a time window; the backend provisions the matching Kueue ClusterQueue/LocalQueue/HardwareProfile resources and cleans them up when the reservation expires.

## What it is

- **Frontend** — Webpack 5 Module Federation plugin (Module Federation name `gpuBooking`, route `/gpu-booking`) served by Nginx, loaded by the RHOAI Dashboard host at runtime.
- **Backend** (`cmd/backend/`, `pkg/`) — Go API server on port 3000. Verifies users via TokenReview/SubjectAccessReview, persists bookings in SQLite on a PVC, manages Kueue resources with its ServiceAccount, and auto-discovers GPU nodes/MIG slices.

## Repository layout

```text
chart/                  Helm chart (frontend Deployment/Service + backend Deployment/Service + RBAC + PVC)
cmd/backend/            Go backend entry point
pkg/                    Go packages (api, database, kube)
src/                    Frontend source
config/                 Webpack configs
plugin.yaml             Charter registry + Module Federation plugin manifest
Containerfile           Frontend image (UBI9 node → UBI9 nginx, port 8080)
Containerfile.bff       Backend image (UBI9 go-toolset 1.25 → UBI9 minimal, port 3000)
scripts/                Build tooling (chart version sync)
```

## Build

```bash
make build          # frontend production build to dist/
make test           # frontend (vitest) + Go tests
make lint-chart     # helm lint chart/
make image          # build both container images (podman/docker)
```

See `make help` for all targets.

## Deploy

```bash
helm install gpu-booking oci://quay.io/rh-ai-community-plugins/gpu-booking-chart \
  --version 0.1.0 --namespace cp-gpu-booking --create-namespace
```

Then register the plugin with the RHOAI Dashboard via `MODULE_FEDERATION_CONFIG` — full steps in [docs/deployment/OPENSHIFT_DEPLOY.md](docs/deployment/OPENSHIFT_DEPLOY.md).

## Support

- Repo: https://github.com/rh-ai-community-plugins/gpu-booking
- Issues: https://github.com/rh-ai-community-plugins/gpu-booking/issues

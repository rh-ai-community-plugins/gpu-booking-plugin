# Deploying the Plugin on OpenShift

This guide walks through deploying the GPU Booking plugin on an OpenShift cluster that already has the Red Hat OpenShift AI (RHOAI) Dashboard running.

## Prerequisites

- **Helm** — to install the plugin chart
- **`oc` CLI** — logged in to the target OpenShift cluster
- **Cluster access for the backend** — the backend's ServiceAccount needs cluster-scoped Kueue/HardwareProfile permissions (created automatically by the Helm chart)
- **Kueue** — must be installed on the cluster for GPU quota reservations
- **Access to `redhat-ods-applications`** — typically requires cluster-admin, since you need to modify the dashboard's Deployment

> **ODH vs RHOAI:** This guide uses the RHOAI dashboard namespace `redhat-ods-applications` and deployment name `rhods-dashboard`. If you are running the Open Data Hub (ODH) upstream distribution instead, substitute `opendatahub` for the namespace and `odh-dashboard` for the deployment name throughout.

---

## 1. Install the Plugin

Install directly from the OCI registry — no need to clone the repo:

```bash
helm install gpu-booking oci://quay.io/rh-ai-community-plugins/gpu-booking-chart \
  --version 0.1.0 \
  --namespace cp-gpu-booking \
  --create-namespace
```

Or, from a local checkout of the repository:

```bash
helm install gpu-booking chart/ \
  --namespace cp-gpu-booking \
  --create-namespace
```

This creates (in `cp-gpu-booking` by default):

- A **Deployment** and **Service** (`gpu-booking`) serving the plugin's static assets (including `remoteEntry.js`) via Nginx on port 8080
- A **Backend Deployment** and **Service** (`gpu-booking-bff`) running the Go API on port 3000
- A **ServiceAccount**, **ClusterRole**, and **ClusterRoleBinding** (`gpu-booking-bff`) with the backend's cluster-scoped permissions (TokenReview/SubjectAccessReview create, Kueue management, HardwareProfiles, node listing)
- A **PersistentVolumeClaim** (`gpu-booking-data`, 2Gi default) for the SQLite booking database
- Optionally a **ConfigMap** with static GPU resource config (only when `gpuDiscovery.enabled=false`)

### Overriding Defaults

Pass `--set` flags to customize the installation:

```bash
helm install gpu-booking oci://quay.io/rh-ai-community-plugins/gpu-booking-chart \
  --version 0.1.0 \
  --namespace cp-gpu-booking \
  --create-namespace \
  --set persistence.size=5Gi
```

To deploy the frontend only (no backend):

```bash
helm install gpu-booking oci://quay.io/rh-ai-community-plugins/gpu-booking-chart \
  --version 0.1.0 \
  --namespace cp-gpu-booking \
  --create-namespace \
  --set bff.enabled=false
```

See [Helm Chart Reference](#helm-chart-reference) for the full list of configurable values.

---

## 2. Register with the RHOAI Dashboard

The dashboard discovers plugins through the `MODULE_FEDERATION_CONFIG` environment variable on its Deployment. You need to append this plugin's entry to that configuration.

### Frontend Only

If you deployed without the backend, use this configuration:

```bash
oc get configmap federation-config \
  -n redhat-ods-applications \
  -o jsonpath='{.data.module-federation-config\.json}' \
| python3 -c "
import json, sys
config = json.load(sys.stdin)
config.append({
  'name': 'gpuBooking',
  'backend': {
    'remoteEntry': '/remoteEntry.js',
    'authorize': False,
    'tls': False,
    'service': {
      'name': 'gpu-booking',
      'namespace': 'cp-gpu-booking',
      'port': 8080
    }
  }
})
print(json.dumps(config))
" > /tmp/mf-config-extended.json

oc set env deployment/rhods-dashboard \
  -n redhat-ods-applications \
  "MODULE_FEDERATION_CONFIG=$(cat /tmp/mf-config-extended.json)"
```

### Frontend + Backend

If you deployed with the backend enabled, add a `proxyService` entry so the dashboard proxies API requests to the backend service:

```bash
oc get configmap federation-config \
  -n redhat-ods-applications \
  -o jsonpath='{.data.module-federation-config\.json}' \
| python3 -c "
import json, sys
config = json.load(sys.stdin)
config.append({
  'name': 'gpuBooking',
  'backend': {
    'remoteEntry': '/remoteEntry.js',
    'authorize': False,
    'tls': False,
    'service': {
      'name': 'gpu-booking',
      'namespace': 'cp-gpu-booking',
      'port': 8080
    }
  },
  'proxyService': [{
    'path': '/gpu-booking/api',
    'pathRewrite': '/api',
    'authorize': True,
    'tls': False,
    'service': {
      'name': 'gpu-booking-bff',
      'namespace': 'cp-gpu-booking',
      'port': 3000
    }
  }]
})
print(json.dumps(config))
" > /tmp/mf-config-extended.json

oc set env deployment/rhods-dashboard \
  -n redhat-ods-applications \
  "MODULE_FEDERATION_CONFIG=$(cat /tmp/mf-config-extended.json)"
```

The `proxyService` entry tells the dashboard to forward requests from `/gpu-booking/api/*` to the backend service, rewriting the path to `/api/*` and forwarding the user's Bearer token (`authorize: True`). The backend's auth middleware verifies that token with TokenReview/SubjectAccessReview.

### Why `MODULE_FEDERATION_CONFIG` Instead of the ConfigMap?

The RHOAI operator reconciles the `federation-config` ConfigMap, which means direct edits to it may be reverted. Setting the environment variable on the Deployment overrides the ConfigMap value and survives operator reconciliation.

New dashboard pods roll out automatically after the environment variable is set. After roughly two minutes, reload the RHOAI Dashboard in your browser to see the plugin's sidebar entry.

---

## 3. Verify

### Check registration

Confirm the plugin appears in the dashboard's federation config:

```bash
oc set env deployment/rhods-dashboard -n redhat-ods-applications --list \
  | grep MODULE_FEDERATION_CONFIG \
  | python3 -c "
import json, sys
data = json.loads(sys.stdin.read().split('=', 1)[1])
for entry in data:
    name = entry['name']
    has_proxy = bool(entry.get('proxyService'))
    print(f'  {name}' + (' (+ proxy)' if has_proxy else ''))
"
```

### Check pods

Verify the plugin pods are running:

```bash
oc get pods -n cp-gpu-booking
```

You should see pods for `gpu-booking` (Nginx frontend) and `gpu-booking-bff` (Go backend), both in `Running` status.

### Check the backend health

```bash
oc exec -n cp-gpu-booking deploy/gpu-booking-bff -- \
  curl -s http://localhost:3000/api/health
```

### Check the dashboard

Open the RHOAI Dashboard in your browser. You should see the plugin in the sidebar under **Community Plugins > GPU Booking**.

---

## Uninstalling

### 1. Remove from the dashboard federation config

Retrieve the current config, remove the `gpuBooking` entry, and re-apply:

```bash
oc get configmap federation-config \
  -n redhat-ods-applications \
  -o jsonpath='{.data.module-federation-config\.json}' \
| python3 -c "
import json, sys
config = json.load(sys.stdin)
config = [e for e in config if e.get('name') != 'gpuBooking']
print(json.dumps(config))
" > /tmp/mf-config-reduced.json

oc set env deployment/rhods-dashboard \
  -n redhat-ods-applications \
  "MODULE_FEDERATION_CONFIG=$(cat /tmp/mf-config-reduced.json)"
```

### 2. Uninstall the Helm release

```bash
helm uninstall gpu-booking -n cp-gpu-booking
oc delete namespace cp-gpu-booking   # optional: remove the namespace entirely
```

> **Note:** The booking-data PVC has `helm.sh/resource-policy: keep`, so it survives `helm uninstall`. Delete it manually if you want to drop all booking data:
>
> ```bash
> oc delete pvc gpu-booking-data -n cp-gpu-booking
> ```

---

## Helm Chart Reference

### Frontend Values

| Parameter | Default | Description |
|---|---|---|
| `namespace` | `cp-gpu-booking` | Target namespace for all namespaced resources |
| `image.repository` | `quay.io/rh-ai-community-plugins/gpu-booking` | Frontend container image |
| `image.tag` | `""` (defaults to appVersion) | Frontend image tag |
| `image.pullPolicy` | `IfNotPresent` | Image pull policy |
| `replicaCount` | `1` | Frontend replicas |
| `service.type` | `ClusterIP` | Frontend Service type |
| `service.port` | `8080` | Frontend Service port |
| `resources.requests.cpu` | `50m` | Frontend CPU request |
| `resources.requests.memory` | `64Mi` | Frontend memory request |
| `resources.limits.cpu` | `100m` | Frontend CPU limit |
| `resources.limits.memory` | `128Mi` | Frontend memory limit |

### Backend Values

| Parameter | Default | Description |
|---|---|---|
| `bff.enabled` | `true` | Deploy the backend service |
| `bff.image.repository` | `quay.io/rh-ai-community-plugins/gpu-booking-bff` | Backend container image |
| `bff.image.tag` | `""` (defaults to appVersion) | Backend image tag |
| `bff.service.port` | `3000` | Backend Service port |
| `bff.tls.enabled` | `false` | Use an OpenShift serving cert for backend TLS |
| `bff.rbac.create` | `true` | Create the ClusterRole/ClusterRoleBinding for the backend SA |
| `bff.resources.requests.cpu` | `100m` | Backend CPU request |
| `bff.resources.requests.memory` | `256Mi` | Backend memory request |
| `bff.resources.limits.cpu` | `500m` | Backend CPU limit |
| `bff.resources.limits.memory` | `512Mi` | Backend memory limit |

### Persistence and Backend Tuning Values

| Parameter | Default | Description |
|---|---|---|
| `persistence.enabled` | `true` | Create the SQLite data PVC |
| `persistence.size` | `2Gi` | PVC size |
| `persistence.storageClass` | `""` (cluster default) | Storage class for the PVC |
| `persistence.mountPath` | `/app/data` | PVC mount path in the backend pod |
| `bookingWindowDays` | `30` | How far ahead users can book |
| `kueue.syncEnabled` | `true` | Sync Kueue usage into the booking grid |
| `kueue.syncIntervalSeconds` | `30` | Kueue sync interval (seconds) |
| `kueue.bookingDays` | `7` | Days of Kueue workload history to sync |
| `gpuDiscovery.enabled` | `true` | Auto-discover GPU nodes/MIG slices (disable to use the static `gpuConfig`) |
| `gpuDiscovery.intervalSeconds` | `600` | Discovery re-scan interval (seconds) |

For the complete list, see [`chart/values.yaml`](../../chart/values.yaml).

# Cloud Cafe — TARSy Demo Application

A whimsical cloud-themed cafe app deployed on OpenShift, purpose-built to demonstrate TARSy's incident investigation capabilities.

Users browse fantasy drinks and place orders. A deliberately injected failure creates a cascading outage that TARSy investigates and resolves.

## Architecture

```
┌───────────┐     ┌──────────┐     ┌────────────┐
│ Frontend  │────▶│ Backend  │────▶│ PostgreSQL │
│  (Nginx)  │     │   (Go)   │     │            │
└───────────┘     └──────────┘     └────────────┘
   :8080           :3000              :5432
```

- **Frontend** — Nginx serving static HTML/CSS/JS. Proxies `/api/*` to the backend.
- **Backend** — Go HTTP API. Endpoints: `/api/menu`, `/api/orders`, `/healthz`.
- **Database** — PostgreSQL 16. Tables: `drinks`, `orders`. Seeded on startup by backend.

## Images

| Component | Image |
|-----------|-------|
| Backend   | `quay.io/alexeykazakov/cloud-cafe-backend:latest` |
| Frontend  | `quay.io/alexeykazakov/cloud-cafe-frontend:latest` |
| Database  | `mirror.gcr.io/library/postgres:16-alpine` |

## Prerequisites

- `oc login` to an OpenShift cluster
- Cluster can pull from `quay.io` and `mirror.gcr.io`
- TARSy running and reachable (for the investigation step)

## Demo Flow

### Step 1 — Deploy the healthy app

```bash
./scripts/setup-demo.sh
```

This creates the `cloud-cafe-prod` namespace with a ResourceQuota (512Mi memory, 500m CPU), deploys all three tiers, waits for everything to be ready, and prints the route URL.

Open the URL in a browser — the menu loads, you can place orders.

### Step 2 — Break it

```bash
./scripts/break-demo.sh
```

Patches the PostgreSQL Deployment to request **1Gi memory** — exceeding the 512Mi namespace quota. The `Recreate` strategy kills the old DB pod first, and the new one can't schedule (Pending). The backend loses its database connection and enters CrashLoopBackOff. The frontend shows an error banner.

The cascade:
1. PostgreSQL pod — **Pending** (quota exceeded)
2. Backend pod — **CrashLoopBackOff** (DB connection timeout)
3. Frontend — serves error state to users

### Step 3 — Submit alert to TARSy

```bash
./scripts/submit-alert.sh https://tarsy.apps.your-cluster.example.com
```

Sends a terse alert to TARSy's API:

```
CRITICAL: Pods in namespace cloud-cafe-prod are in CrashLoopBackOff.
Application: cloud-cafe-api. Restarts: 5+. Duration: 10m.
```

TARSy's `orchestrator-investigation-anthropic` chain dispatches sub-agents that trace the failure across pods, events, and quota — arriving at the root cause.

### Step 4 — Follow TARSy's recommendations

**No pre-written fix script.** The operator reads TARSy's recommendations on camera and applies them. The expected fix is something like:

```bash
oc patch deployment postgres -n cloud-cafe-prod -p \
  '{"spec":{"template":{"spec":{"containers":[{"name":"postgres","resources":{"requests":{"memory":"256Mi"}}}]}}}}'
```

The DB pod schedules, backend reconnects, cafe is back.

### Step 5 — Tear down

```bash
./scripts/teardown-demo.sh
```

Deletes the `cloud-cafe-prod` namespace and everything in it.

## Local Development

Build and run locally with podman-compose:

```bash
make build    # build images
make up       # start all services
make down     # stop
make logs     # tail logs
make clean    # stop + remove volumes and images
```

App is available at `http://localhost:8080`.

## Pushing Images

```bash
make push     # tag and push both images to quay.io
```

## Resource Budget

| Component  | CPU Request | Memory Request |
|------------|-------------|----------------|
| PostgreSQL | 200m        | 256Mi          |
| Backend    | 100m        | 128Mi          |
| Frontend   | 50m         | 64Mi           |
| **Total**  | **350m**    | **448Mi**      |
| **Quota**  | **500m**    | **512Mi**      |

The break patches PostgreSQL to request 1Gi memory, which exceeds the 512Mi total quota.

## File Structure

```
cloud-cafe/
├── backend/
│   ├── main.go           # Go API server
│   ├── go.mod
│   ├── go.sum
│   └── Dockerfile
├── frontend/
│   ├── index.html        # Single-page UI
│   ├── style.css
│   ├── app.js
│   ├── nginx.conf        # Reverse proxy config
│   ├── images/           # Drink images
│   └── Dockerfile
├── k8s/
│   ├── 00-namespace.yaml
│   ├── 01-resource-quota.yaml
│   ├── 02-postgres.yaml
│   ├── 03-backend.yaml
│   └── 04-frontend.yaml
├── scripts/
│   ├── setup-demo.sh
│   ├── break-demo.sh
│   ├── submit-alert.sh
│   └── teardown-demo.sh
├── compose.yml           # Local dev stack
├── Makefile
└── README.md
```

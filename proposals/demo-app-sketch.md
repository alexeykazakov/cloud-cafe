# Cloud Cafe — TARSy Demo Application for OpenShift

**Status:** Sketch complete — ready for implementation

## Problem

TARSy needs a live, reproducible demo environment in an OpenShift cluster for a recorded video. The demo shows a real application with a real problem that TARSy investigates via the `orchestrator-investigation-anthropic` chain, produces actionable recommendations, and the operator follows those recommendations to fix the issue visibly.

## Demo Flow

```
1. Show healthy app       → Cloud Cafe running, menu loads, orders work
2. Apply bad change       → patch DB deployment with oversized resource requests
3. Show broken state      → DB Pending, backend CrashLoop, frontend shows errors
4. Submit alert to TARSy  → terse alert via curl or dashboard
5. TARSy investigates     → orchestrator dispatches sub-agents, traces the cascade
6. Follow recommendations → operator applies TARSy's suggested fix (no pre-written script)
7. Show recovery          → DB schedules, backend reconnects, cafe is back
```

## Application: Cloud Cafe

A whimsical cloud-themed cafe where users order fantasy drinks (Rainbow Latte, Thunderstorm Espresso, Midnight Nebula Cappuccino). The fun theme contrasts with the dead-serious SRE investigation — TARSy treats it like a mission-critical production service.

### Architecture

- **Frontend**: Nginx serving a clean, nice-looking static page — drink menu + order button. Shows an error state when the backend is down. Simple but polished (custom images welcome for visual appeal).
- **Backend**: Go HTTP API — endpoints for menu, order submission, health check. Connects to PostgreSQL on startup. Produces detailed, professional error logs on DB connection failure.
- **Database**: PostgreSQL — tables for drinks and orders.

### Repo & Images

- Source in a standalone public GitHub repo (locally at `temp/cloud-cafe/`, gitignored by TARSy repo)
- Images published to quay.io
- Namespace: `cloud-cafe-prod`

## Failure Scenario

**ResourceQuota blocking the database pod — cascading failure.**

The app is deployed healthy with a namespace ResourceQuota that has enough room. Then a "bad change" is applied: the DB Deployment is patched with higher resource requests that exceed the quota. The rollout terminates the old DB pod, but the new one can't schedule (Pending). The backend loses its DB connection and enters CrashLoopBackOff. The frontend shows errors.

**Investigation chain** (multi-hop, not obvious from any single resource):
1. Backend pods → CrashLoopBackOff, logs show DB connection timeout
2. DB pods → Pending, not running
3. Namespace events → quota exceeded for the DB pod
4. ResourceQuota → hard limit vs. requested resources mismatch

**Fix**: lower the DB resource requests to fit within the quota (or increase the quota). Single `oc patch` command from TARSy's recommendations. DB pod schedules → backend reconnects → cafe is back.

## Chain Under Demo

`orchestrator-investigation-anthropic` (line 197 in `deploy/config/tarsy.yaml`):

- **Alert type**: `"Orchestrator Investigation - Anthropic"`
- **Two parallel Orchestrators**:
  - Anthropic (vertexai-anthropic / langchain) with sub-agents: KubernetesAgent, WebResearcher, CodeExecutor, GeneralWorker
  - Gemini 3.1 Pro (google-native) with sub-agents: KubernetesAgent, WebResearcher, CodeExecutor, GeneralWorker
- **Synthesis** via Anthropic

Exercises: dynamic sub-agent dispatch, multi-model comparison, K8s tool calling, web research, synthesis.

## Alert Payload

Terse, minimal — forces TARSy to investigate from almost nothing:

```
CRITICAL: Pods in namespace cloud-cafe-prod are in CrashLoopBackOff.
Application: cloud-cafe-api. Restarts: 5+. Duration: 10m.
```

## Scripts

All scripts live in the Cloud Cafe GitHub repo.

### `setup-demo.sh`

- Creates `cloud-cafe-prod` namespace (warns/cleans if exists)
- Applies ResourceQuota
- Deploys all three tiers (DB, backend, frontend)
- Waits for all pods to be healthy
- Prints status and app URL

### `break-demo.sh`

- Patches DB Deployment with oversized resource requests
- Waits for cascade (DB Pending → backend CrashLoop)
- Prints broken status

### `submit-alert.sh` (optional convenience)

- Sends `POST /api/v1/alerts` with the terse alert payload
- Takes TARSy URL as argument

### `teardown-demo.sh`

- Deletes the `cloud-cafe-prod` namespace

**No fix script** — the operator follows TARSy's recommendations on camera.

## Implementation Plan

### Phase 1: Build the Application

Get Cloud Cafe running locally as a real working app.

**Backend (Go)**:
- HTTP server with endpoints: `GET /api/menu`, `POST /api/orders`, `GET /api/orders`, `GET /healthz`
- PostgreSQL connection with retry logic and clear error logging on failure
- Seed data: 5-6 fantasy drinks with names, descriptions, prices
- Structured JSON logging (connection errors, request handling)

**Database**:
- PostgreSQL schema: `drinks` table (id, name, description, price, image_url) and `orders` table (id, drink_id, customer_name, status, created_at)
- Migration SQL file run by the backend on startup (or init container)

**Frontend**:
- Static HTML/CSS/JS — single page
- Drink menu grid with images, names, prices
- Order button per drink, simple order form (name + drink)
- Error state when backend is unreachable ("Cloud Cafe is temporarily closed")
- Clean, polished look with pastel/cloud theme (custom images can be added later)

**Local dev**:
- `docker-compose.yml` for local testing (PostgreSQL + backend + frontend/nginx)
- Verify: menu loads, orders submit, health check returns OK

**Done when**: the app runs locally via docker-compose, you can browse the menu, place an order, and see it reflected.

### Phase 2: Containerize & Publish

- Dockerfile for backend (multi-stage Go build)
- Dockerfile for frontend (nginx + static files)
- Build and push both images to quay.io
- Test images run correctly via docker-compose with published tags

**Done when**: `docker-compose up` with quay.io image tags works identically to local build.

### Phase 3: OpenShift Manifests & Demo Scripts

- K8s manifests: Namespace, ResourceQuota, Deployments (frontend, backend, PostgreSQL), Services, ConfigMaps, Route
- `setup-demo.sh` — deploys everything, waits for healthy, prints URL
- `break-demo.sh` — patches DB with oversized resource requests, waits for cascade
- `submit-alert.sh` — sends alert to TARSy API
- `teardown-demo.sh` — deletes namespace

**Done when**: full `setup → verify healthy → break → verify broken → teardown` cycle works on a real OpenShift cluster.

### Phase 4: End-to-End Demo Run

- Deploy to target cluster
- Run the full demo flow including TARSy investigation
- Verify TARSy finds the root cause and produces actionable recommendations
- Apply recommendations, verify recovery
- Record the video

**Done when**: successful end-to-end recording.

---

## Out of Scope

- Automated action stage (remediation) — demo focuses on investigation + human follow-up
- Slack integration — optional, not required for the demo
- TARSy deployment itself — assumes TARSy is already running and reachable
- MCP server setup — assumes kubernetes-mcp-server is configured and connected to the target cluster

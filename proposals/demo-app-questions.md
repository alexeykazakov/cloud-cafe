# TARSy Demo Application — Sketch Questions

**Status:** Open — decisions pending
**Related:** [Sketch document](demo-app-sketch.md)

Each question has options with trade-offs and a recommendation. Go through them one by one to form the sketch, then update the sketch document.

---

## Q1: What type of application to deploy?

The demo app needs to be simple enough to deploy in one script, realistic enough to be credible, and its failure mode must produce enough Kubernetes artifacts (logs, events, pod status) for TARSy's agents to investigate.

### Option C: Multi-tier app (frontend + backend + database)

Deploy a small stack — e.g., a frontend Nginx proxy, a backend API, and a PostgreSQL database.

- **Pro:** More realistic, resembles production setups
- **Pro:** Cascading failure is inherently interesting (DB issue → backend crash → frontend errors)
- **Con:** More resources, longer setup, more things that can go wrong unrelated to the demo
- **Con:** Harder to explain quickly

**Decision:** Option C — a multi-tier app looks more realistic on camera and the cascading-failure narrative is more compelling. The problem itself stays simple; the extra setup complexity is acceptable for a recorded video (retakes are possible).

_Considered and rejected: Option A (full control but looks toy-like on video), Option B (less control over failure behavior and logs)_

---

## Q2: What failure scenario to inject?

The fault must be: (a) detectable by inspecting K8s resources (pods, events, logs, configmaps), (b) non-obvious enough that an AI investigation adds value, (c) fixable with a single command.

### Option F: Resource quota blocking database pod (cascading failure)

The app is deployed and running healthy (all three tiers up). A namespace ResourceQuota is in place with enough room for the initial deployment. Then a "bad change" is applied — the DB Deployment is patched with higher resource requests (simulating someone anticipating heavier workload). The new requests exceed the quota. The rollout terminates the old DB pod, but the new one is stuck Pending. The backend loses its DB connection and starts crash-looping. The frontend starts returning errors.

**Investigation chain** (multi-hop, not obvious from any single resource):
1. Backend pods → CrashLoopBackOff, logs show DB connection timeout
2. DB pods → Pending, not running
3. Namespace events → quota exceeded for the DB pod
4. ResourceQuota → hard limit vs. requested resources mismatch

**Fix** (single command): lower the DB resource requests back to fit within the quota (or increase the quota). The DB pod schedules, backend reconnects, frontend recovers. Full cascade in reverse.

- **Pro:** Multi-hop investigation — root cause is two levels removed from the visible symptom
- **Pro:** "Was working, then broke" narrative — realistic incident triggered by a config change
- **Pro:** Exercises multiple K8s resource types (Pods, Events, ResourceQuota, Deployments across tiers)
- **Pro:** Fix is a single `oc patch` — easy to apply from TARSy's recommendations on camera
- **Pro:** Recovery is visible — pods go Running, app responds again
- **Con:** Requires ResourceQuota setup (minor extra complexity in setup script)

**Decision:** Option F — the root cause (ResourceQuota blocking the DB pod after a config change) is invisible from the backend's perspective, requiring a multi-hop investigation across tiers. The full cycle works cleanly: healthy → break → investigate → fix → healthy.

_Considered and rejected: Option A (too simple — SRE spots it in under a minute from logs), Option B (OOMKill is obvious from pod status), Option C (ImagePullBackOff is trivial), Option D (subtle but not dramatic), Option E (two unrelated issues risks partial discovery and muddied narrative)_

---

## Q3: What container images to use for the demo app?

With a multi-tier app (Q1) and a ResourceQuota fault (Q2), the database is always a stock PostgreSQL image and the frontend is stock Nginx. The real question is the backend — it must connect to PostgreSQL on startup and produce meaningful error logs when it can't.

### Option C: Custom app on a public registry (quay.io)

Build a real multi-tier application — a Go or Python backend API that connects to PostgreSQL, served behind an Nginx frontend. Push images to quay.io. Host the source in its own public GitHub repo (standalone, not part of the TARSy repo).

- **Pro:** Looks like a real production app — realistic logs, proper health checks, meaningful error messages
- **Pro:** Full control over log output — the backend can produce detailed connection error logs that make the investigation interesting
- **Pro:** Open-source repo doubles as a reference for anyone wanting to try TARSy
- **Pro:** Setup script just references image tags — no build step in the cluster
- **Con:** Requires building and pushing images upfront
- **Con:** Another repo to maintain (but it's small and stable once built)

**Decision:** Option C — a real custom app published to quay.io with source in a standalone GitHub repo. The source lives locally at `temp/<app-name>/` (gitignored by the TARSy repo, same pattern as sandbox-sre). The app looks authentic on camera, produces realistic logs, and the repo serves as an open-source demo reference.

_Considered and rejected: Option A (standard image with command hack — logs look artificial), Option B (real OSS app like Adminer — less control over error messages and behavior)_

---

## Q4: Namespace and resource naming strategy?

### Option A: Fixed namespace (e.g., `tarsy-demo`)

- **Pro:** Simple, predictable
- **Pro:** Easy to reference in documentation and scripts
- **Con:** Conflicts if someone already has this namespace

### Option B: Generated namespace with prefix (e.g., `tarsy-demo-<random>`)

- **Pro:** No conflicts, can run multiple demos in parallel
- **Con:** Harder to reference, need to pass around the name

**Decision:** Option A — fixed `tarsy-demo` namespace. Simple, predictable, easy to reference in scripts and docs. Setup script checks if it exists and warns/cleans up.

_Considered and rejected: Option B (generated names add complexity for no benefit — parallel demos not needed)_

---

## Q5: Provide a fix script or let the operator follow TARSy's recommendations?

### Option A: No fix script — operator follows TARSy's output

The operator reads TARSy's recommendations and applies them directly. No pre-written fix.

- **Pro:** This IS the demo — proving TARSy's recommendations are actionable
- **Pro:** Most authentic and impressive on camera
- **Con:** Requires TARSy to actually produce correct recommendations (but that's the point)

**Decision:** Option A — the operator follows TARSy's recommendations, not a pre-written script. A prepared fix script would undermine the demo's core message. Since it's a recording, retakes handle any edge cases.

_Considered and rejected: Option B (backup script — safety net nobody should need), Option C (pre-written fix — defeats the purpose of the demo)_

---

## Q6: Alert payload style?

The `data` field in the alert is free-text. It becomes the user prompt that drives the investigation.

### Option A: Terse alert (like a real monitoring alert)

```
CRITICAL: Pods in namespace tarsy-demo are in CrashLoopBackOff.
Application: demo-app. Restarts: 5+. Duration: 10m.
```

- **Pro:** Realistic — this is what AlertManager/PagerDuty sends
- **Pro:** Forces TARSy to investigate from minimal information
- **Con:** Might lead to a slower investigation

### Option B: Descriptive alert with context

```
Multiple pods for the demo-app deployment in the tarsy-demo namespace are crash-looping.
The team reports the application was recently updated with new configuration.
Please investigate the root cause and provide remediation steps.
```

- **Pro:** Gives TARSy more context to start from, faster to a good result
- **Pro:** Feels like a human asking for help
- **Con:** Less realistic as an automated alert

### Option C: AlertManager-style structured alert

```json
[FIRING] demo-app CrashLoopBackOff
Namespace: tarsy-demo
Deployment: demo-app
Status: CrashLoopBackOff
Restarts: 5
Message: Back-off restarting failed container
```

- **Pro:** Looks like a real AlertManager payload
- **Pro:** Structured data helps TARSy focus immediately
- **Con:** Slightly artificial if not actually from AlertManager

**Decision:** Option A — terse alert with minimal information. Forces TARSy to do the full detective work from almost nothing, which is the most impressive demonstration of its value.

_Considered and rejected: Option B (too much hand-holding — less impressive), Option C (structured format gives too many hints)_

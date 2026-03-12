#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="cloud-cafe-prod"

echo "=== Injecting Failure ==="
echo ""
echo "Patching PostgreSQL deployment with oversized memory request (1Gi)..."
echo "The namespace quota only allows 512Mi total — this will exceed it."
echo ""

oc patch deployment postgres -n "$NAMESPACE" -p '{
  "spec": {
    "template": {
      "spec": {
        "containers": [{
          "name": "postgres",
          "resources": {
            "requests": {
              "memory": "1Gi",
              "cpu": "200m"
            }
          }
        }]
      }
    }
  }
}'

echo ""
echo "Patch applied. The Recreate strategy will kill the old DB pod first."
echo "Waiting for the cascade..."
echo ""

# Wait for the old postgres pod to terminate and new one to get stuck
sleep 5

echo "--- Pod Status ---"
oc get pods -n "$NAMESPACE"

echo ""
echo "--- ReplicaSet Events (look for quota exceeded) ---"
oc get events -n "$NAMESPACE" --sort-by='.lastTimestamp' --field-selector reason=FailedCreate 2>/dev/null | tail -5 || true

echo ""
echo "Waiting for backend to enter CrashLoopBackOff (DB is gone)..."
echo "(This may take 30-60 seconds)"

for i in $(seq 1 12); do
    sleep 10
    BACKEND_STATUS=$(oc get pods -n "$NAMESPACE" -l app=backend -o jsonpath='{.items[0].status.containerStatuses[0].state.waiting.reason}' 2>/dev/null || echo "")
    BACKEND_RESTARTS=$(oc get pods -n "$NAMESPACE" -l app=backend -o jsonpath='{.items[0].status.containerStatuses[0].restartCount}' 2>/dev/null || echo "0")

    echo "  [$i] backend: status=${BACKEND_STATUS:-Running} restarts=${BACKEND_RESTARTS}"

    if [[ "$BACKEND_STATUS" == "CrashLoopBackOff" ]]; then
        break
    fi
done

echo ""
echo "=== Broken State ==="
oc get pods -n "$NAMESPACE"

echo ""
echo "=== Resource Quota ==="
oc get resourcequota cloud-cafe-quota -n "$NAMESPACE" -o custom-columns=\
"NAME:.metadata.name,"\
"CPU_USED:.status.used.requests\.cpu,"\
"CPU_HARD:.status.hard.requests\.cpu,"\
"MEM_USED:.status.used.requests\.memory,"\
"MEM_HARD:.status.hard.requests\.memory"

echo ""
echo "=== Recent Events ==="
oc get events -n "$NAMESPACE" --sort-by='.lastTimestamp' | tail -10

echo ""
echo "Cloud Cafe is DOWN. Run 'scripts/submit-alert.sh <TARSY_URL>' to alert TARSy."

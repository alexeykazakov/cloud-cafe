#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="cloud-cafe-prod"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
K8S_DIR="$SCRIPT_DIR/../k8s"

echo "=== Cloud Cafe Demo Setup ==="
echo ""

# Clean up if namespace already exists
if oc get namespace "$NAMESPACE" &>/dev/null; then
    echo "⚠  Namespace $NAMESPACE already exists — deleting..."
    oc delete namespace "$NAMESPACE" --wait=true
    echo "   Waiting for namespace to be fully removed..."
    while oc get namespace "$NAMESPACE" &>/dev/null; do
        sleep 2
    done
    echo "   Done."
    echo ""
fi

# Create namespace and quota
echo "1/4  Creating namespace and resource quota..."
oc apply -f "$K8S_DIR/00-namespace.yaml"
oc apply -f "$K8S_DIR/01-resource-quota.yaml"

# Grant SCC for postgres (needs writable volume as specific UID)
oc adm policy add-scc-to-user anyuid -z default -n "$NAMESPACE" 2>/dev/null || true

echo ""
echo "2/4  Deploying PostgreSQL..."
oc apply -f "$K8S_DIR/02-postgres.yaml"

echo "     Waiting for PostgreSQL to be ready..."
oc rollout status deployment/postgres -n "$NAMESPACE" --timeout=120s

echo ""
echo "3/4  Deploying backend..."
oc apply -f "$K8S_DIR/03-backend.yaml"

echo "     Waiting for backend to be ready..."
oc rollout status deployment/backend -n "$NAMESPACE" --timeout=120s

echo ""
echo "4/4  Deploying frontend..."
oc apply -f "$K8S_DIR/04-frontend.yaml"

echo "     Waiting for frontend to be ready..."
oc rollout status deployment/frontend -n "$NAMESPACE" --timeout=60s

echo ""
echo "=== Status ==="
oc get pods -n "$NAMESPACE"

echo ""
echo "=== Resource Quota ==="
oc get resourcequota cloud-cafe-quota -n "$NAMESPACE" -o custom-columns=\
"NAME:.metadata.name,"\
"CPU_USED:.status.used.requests\.cpu,"\
"CPU_HARD:.status.hard.requests\.cpu,"\
"MEM_USED:.status.used.requests\.memory,"\
"MEM_HARD:.status.hard.requests\.memory"

ROUTE_URL=$(oc get route cloud-cafe -n "$NAMESPACE" -o jsonpath='{.spec.host}' 2>/dev/null || echo "")
echo ""
if [[ -n "$ROUTE_URL" ]]; then
    echo "=== Cloud Cafe is LIVE ==="
    echo "   URL: https://$ROUTE_URL"
else
    echo "=== Cloud Cafe is running (no route URL detected) ==="
fi
echo ""
echo "Run 'scripts/break-demo.sh' when ready to inject the failure."

#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="cloud-cafe-prod"

echo "=== Cloud Cafe Demo Teardown ==="
echo ""

if ! oc get namespace "$NAMESPACE" &>/dev/null; then
    echo "Namespace $NAMESPACE does not exist. Nothing to tear down."
    exit 0
fi

echo "Deleting namespace $NAMESPACE (this removes everything)..."
oc delete namespace "$NAMESPACE" --wait=false

echo ""
echo "Namespace deletion initiated. It may take a minute to fully remove."
echo "Check with: oc get namespace $NAMESPACE"

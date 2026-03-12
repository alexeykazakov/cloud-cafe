#!/usr/bin/env bash
set -euo pipefail

TARSY_URL="${1:-http://localhost:8080}"

# Strip trailing slash
TARSY_URL="${TARSY_URL%/}"

ALERT_PAYLOAD='{
  "alert_type": "Incident Investigation",
  "data": "CRITICAL: Pods in namespace cloud-cafe-prod are in CrashLoopBackOff. Application: cloud-cafe-api. Restarts: 5+. Duration: 10m."
}'

echo "=== Submitting Alert to TARSy ==="
echo ""
echo "Endpoint: $TARSY_URL/api/v1/alerts"
echo "Payload:"
echo "$ALERT_PAYLOAD" | python3 -m json.tool 2>/dev/null || echo "$ALERT_PAYLOAD"
echo ""

RESPONSE=$(curl -s -w "\n%{http_code}" -X POST \
    "$TARSY_URL/api/v1/alerts" \
    -H "Content-Type: application/json" \
    -d "$ALERT_PAYLOAD")

HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')

echo "Response ($HTTP_CODE):"
echo "$BODY" | python3 -m json.tool 2>/dev/null || echo "$BODY"
echo ""

if [[ "$HTTP_CODE" == "202" ]]; then
    SESSION_ID=$(echo "$BODY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('session_id',''))" 2>/dev/null || echo "")
    echo "Alert accepted! TARSy is investigating."
    if [[ -n "$SESSION_ID" ]]; then
        echo "Session ID: $SESSION_ID"
        echo "Dashboard:  $TARSY_URL/sessions/$SESSION_ID"
    fi
else
    echo "Unexpected response code: $HTTP_CODE"
    exit 1
fi

#!/usr/bin/env bash
# Share the Moodle++ dev stack over ngrok (single tunnel).
#
#   ./scripts/ngrok-share.sh [frontend-port]
#
# Single-tunnel design: browser sees ONE origin (the ngrok URL). Vite's dev
# proxy forwards /api to the local backend, so no CORS or cookie changes are
# needed. The frontend sends `ngrok-skip-browser-warning` (dev only) to dodge
# the free-tier interstitial on API calls.
#
# Requires: go, node/npm, ngrok v3 with authtoken (ngrok config default).
# Leaves nothing running on Ctrl-C (trap kills backend, vite, ngrok).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PORT="${1:-5173}"
BACKEND_DIR="$ROOT/backend"
FRONTEND_DIR="$ROOT/frontend"

need() { command -v "$1" >/dev/null 2>&1 || { echo "missing: $1" >&2; exit 1; }; }
need go; need npm; need ngrok; need curl

cleanup() {
  echo; echo "-- stopping (backend $BE_PID, vite $VITE_PID, ngrok $NGROK_PID)"
  kill "$BE_PID" "$VITE_PID" "$NGROK_PID" 2>/dev/null || true
  rm -f "$BACKEND_DIR/moodleplusplus"
}
trap cleanup EXIT INT TERM

echo "== build backend"
cd "$BACKEND_DIR"
go build -o moodleplusplus . 1>&2

echo "== start backend :8080"
./moodleplusplus > /tmp/mpp-backend.log 2>&1 &
BE_PID=$!
for i in $(seq 1 15); do
  curl -sf -o /dev/null http://localhost:8080/health && break
  sleep 1
  [ "$i" = 15 ] && { echo "backend never came up (see /tmp/mpp-backend.log)" >&2; exit 1; }
done
echo "backend ok (pid $BE_PID)"

echo "== start vite :$PORT"
cd "$FRONTEND_DIR"
npm run dev -- --port "$PORT" > /tmp/mpp-vite.log 2>&1 &
VITE_PID=$!
for i in $(seq 1 20); do
  curl -sf -o /dev/null "http://localhost:$PORT/" && break
  sleep 1
  [ "$i" = 20 ] && { echo "vite never came up (see /tmp/mpp-vite.log)" >&2; exit 1; }
done
echo "vite ok (pid $VITE_PID)"

echo "== open tunnel"
ngrok http "$PORT" --log /tmp/mpp-ngrok.log > /tmp/mpp-ngrok.log 2>&1 &
NGROK_PID=$!
URL=""
for i in $(seq 1 20); do
  URL=$(curl -sf http://localhost:4040/api/tunnels 2>/dev/null \
    | python3 -c "import json,sys; ts=json.load(sys.stdin)['tunnels']; print([t['public_url'] for t in ts if t['proto']=='https'][0])" 2>/dev/null || true)
  [ -n "$URL" ] && break
  sleep 1
done
[ -z "$URL" ] && { echo "tunnel never came up (see /tmp/mpp-ngrok.log)" >&2; exit 1; }

echo
echo "  LIVE: $URL"
echo "  (dismiss the ngrok warning page once in a real browser;"
echo "   API calls bypass it via header. Ctrl-C stops everything.)"
echo
wait "$NGROK_PID"

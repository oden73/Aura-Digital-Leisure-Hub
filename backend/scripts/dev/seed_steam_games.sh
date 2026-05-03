#!/usr/bin/env bash
# Seed 20 popular Steam games via POST /v1/sync/external.
#
# Usage:
#   ./scripts/dev/seed_steam_games.sh [BASE_URL] [JWT_TOKEN]
#
# Defaults:
#   BASE_URL   http://localhost:8080
#   JWT_TOKEN  read from AURA_JWT env var (required if not passed)
#
# The script first registers a throwaway admin user to obtain a JWT,
# then fires one sync request per AppID. Already-existing items are
# safe to re-sync (the use case upserts).

set -euo pipefail

BASE_URL="${1:-http://localhost:8080}"
JWT_TOKEN="${2:-${AURA_JWT:-}}"

# ---------------------------------------------------------------------------
# If no token is provided, register a seed user and log in.
# ---------------------------------------------------------------------------
if [[ -z "$JWT_TOKEN" ]]; then
  echo "[seed] No JWT provided — registering seed user..."
  REGISTER=$(curl -sf -X POST "$BASE_URL/v1/auth/register" \
    -H "Content-Type: application/json" \
    -d '{"username":"seed_admin","email":"seed@aura.local","password":"seed-password-42"}' \
    || true)

  LOGIN=$(curl -sf -X POST "$BASE_URL/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"seed@aura.local","password":"seed-password-42"}')

  JWT_TOKEN=$(echo "$LOGIN" | grep -o '"access":"[^"]*"' | cut -d'"' -f4 || true)
  if [[ -z "$JWT_TOKEN" ]]; then
    echo "[seed] ERROR: Could not obtain JWT. Login response:" >&2
    echo "$LOGIN" >&2
    exit 1
  fi
  echo "[seed] JWT obtained."
fi

# ---------------------------------------------------------------------------
# 20 popular Steam AppIDs
# ---------------------------------------------------------------------------
declare -a APP_IDS=(
  "570"       # Dota 2
  "730"       # Counter-Strike 2
  "440"       # Team Fortress 2
  "271590"    # Grand Theft Auto V
  "1091500"   # Cyberpunk 2077
  "1245620"   # Elden Ring
  "1086940"   # Baldur's Gate 3
  "105600"    # Terraria
  "413150"    # Stardew Valley
  "367520"    # Hollow Knight
  "1623730"   # Vampire Survivors
  "892970"    # Valheim
  "374320"    # Dark Souls III
  "814380"    # Sekiro: Shadows Die Twice
  "548430"    # Deep Rock Galactic
  "359550"    # Tom Clancy's Rainbow Six Siege
  "578080"    # PUBG: Battlegrounds
  "990080"    # Hogwarts Legacy
  "252950"    # Rocket League
  "1817070"   # Lies of P
)

SUCCESS=0
FAIL=0

for APPID in "${APP_IDS[@]}"; do
  RESP=$(curl -sf -X POST "$BASE_URL/v1/sync/external" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $JWT_TOKEN" \
    -d "{\"external_id\":\"$APPID\",\"source\":\"steam\"}" \
    2>&1 || true)

  TITLE=$(echo "$RESP" | grep -o '"title":"[^"]*"' | head -1 | cut -d'"' -f4)
  if [[ -n "$TITLE" ]]; then
    echo "[seed] OK  appid=$APPID  title=$TITLE"
    (( SUCCESS++ )) || true
  else
    echo "[seed] ERR appid=$APPID  response=$RESP"
    (( FAIL++ )) || true
  fi

  # Respect Steam's rate limit (~200 req/5 min on the store API).
  sleep 0.4
done

echo ""
echo "[seed] Done — success=$SUCCESS fail=$FAIL"

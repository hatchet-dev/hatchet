#!/usr/bin/env bash
set -uo pipefail

LITE_IMAGE="${LITE_IMAGE:-hatchet-lite:ci}"
LITE_PORT="${LITE_PORT:-8892}"
TENANT_ID="${TENANT_ID:-c0ffee00-0000-4000-8000-00000000cafe}"

NET=hatchet-bearer-authz
fail=0

cleanup() {
  docker rm -f bearerauthz-lite bearerauthz-pg >/dev/null 2>&1 || true
  docker network rm "$NET" >/dev/null 2>&1 || true
}
cleanup
trap cleanup EXIT

wait_for_url() {
  for _ in $(seq 1 60); do curl -fsS "$1" >/dev/null 2>&1 && return 0; sleep 2; done
  return 1
}

wait_for_pg() {
  for _ in $(seq 1 30); do docker exec "$1" pg_isready -U hatchet >/dev/null 2>&1 && return 0; sleep 2; done
  return 1
}

assert_status() {
  local name="$1" expected="$2"
  shift 2

  local code
  code=$(curl -s -o /dev/null -w '%{http_code}' "$@")

  if [ "$code" != "$expected" ]; then
    echo "::error::$name: api token got $code, expected $expected"
    fail=1
    return 1
  fi

  echo "$name: rejected with $code"
}

count_rows() {
  curl -fsS -H "$AUTH" "$1" | python3 -c 'import sys,json; print(len(json.load(sys.stdin)["rows"]))'
}

docker network create "$NET" >/dev/null

docker run -d --name bearerauthz-pg --network "$NET" \
  -e POSTGRES_USER=hatchet -e POSTGRES_PASSWORD=hatchet -e POSTGRES_DB=hatchet postgres:15.6 >/dev/null
wait_for_pg bearerauthz-pg || { echo "::error::postgres not ready"; exit 1; }

docker run -d --name bearerauthz-lite --network "$NET" \
  -e DATABASE_URL="postgresql://hatchet:hatchet@bearerauthz-pg:5432/hatchet?sslmode=disable" \
  -e DEFAULT_TENANT_ID="$TENANT_ID" \
  -e SERVER_GRPC_BIND_ADDRESS=0.0.0.0 \
  -e SERVER_GRPC_BROADCAST_ADDRESS=localhost:7070 \
  -e SERVER_GRPC_INSECURE=true \
  -e SERVER_AUTH_COOKIE_INSECURE=true \
  -e SERVER_AUTH_COOKIE_DOMAIN=localhost \
  -p "$LITE_PORT":8888 \
  "$LITE_IMAGE" >/dev/null

if ! wait_for_url "http://localhost:$LITE_PORT/api/live"; then
  echo "::error::lite never became ready"
  docker logs bearerauthz-lite 2>&1 | tail -60
  exit 1
fi

TOKEN=$(docker exec bearerauthz-lite ./hatchet-admin token create --config ./config --tenant-id "$TENANT_ID" --name bearer-authz-ci 2>/dev/null | tail -1)
if [ -z "$TOKEN" ]; then
  echo "::error::could not mint an api token for $TENANT_ID"
  exit 1
fi

AUTH="Authorization: Bearer $TOKEN"
JSON="Content-Type: application/json"
BASE="http://localhost:$LITE_PORT/api/v1/tenants/$TENANT_ID"

MEMBER_ID=$(curl -fsS -H "$AUTH" "$BASE/members" | python3 -c 'import sys,json; print(json.load(sys.stdin)["rows"][0]["metadata"]["id"])')
if [ -z "$MEMBER_ID" ]; then
  echo "::error::could not read the seeded tenant member"
  exit 1
fi

assert_status "TenantMemberDelete" 403 -X DELETE -H "$AUTH" "$BASE/members/$MEMBER_ID"
assert_status "TenantMemberUpdate" 403 -X PATCH -H "$AUTH" -H "$JSON" -d '{"role":"OWNER"}' "$BASE/members/$MEMBER_ID"
assert_status "TenantInviteCreate" 403 -X POST -H "$AUTH" -H "$JSON" -d '{"email":"escalate@example.com","role":"OWNER"}' "$BASE/invites"
assert_status "ApiTokenCreate" 403 -X POST -H "$AUTH" -H "$JSON" -d '{"name":"escalate"}' "$BASE/api-tokens"
assert_status "ApiTokenList" 403 -H "$AUTH" "$BASE/api-tokens"

members_after=$(count_rows "$BASE/members")
if [ "$members_after" != "1" ]; then
  echo "::error::tenant member count went from 1 to $members_after"
  fail=1
fi

invites_after=$(count_rows "$BASE/invites")
if [ "$invites_after" != "0" ]; then
  echo "::error::an api token created $invites_after invite(s)"
  fail=1
fi

if [ "$fail" -eq 0 ]; then
  echo "lite: api tokens cannot escalate privileges on user-scoped tenant operations"
fi

exit $fail

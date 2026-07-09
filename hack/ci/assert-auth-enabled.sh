#!/usr/bin/env bash
# Boots the non-dev api, dashboard and lite images against Postgres and asserts authentication is
# ENABLED, i.e. the images were not accidentally compiled with the "authdisabled" build tag.
set -uo pipefail

API_IMAGE="${API_IMAGE:-hatchet-api:ci}"
DASHBOARD_IMAGE="${DASHBOARD_IMAGE:-hatchet-dashboard:ci}"
LITE_IMAGE="${LITE_IMAGE:-hatchet-lite:ci}"
ADMIN_IMAGE="${ADMIN_IMAGE:-hatchet-admin-tmp:amd64}"
MIGRATE_IMAGE="${MIGRATE_IMAGE:-hatchet-migrate-tmp:amd64}"
API_PORT="${API_PORT:-8890}"
DASHBOARD_PORT="${DASHBOARD_PORT:-8891}"
LITE_PORT="${LITE_PORT:-8888}"

NET=hatchet-authcheck
fail=0

cleanup() {
  docker rm -f authcheck-api authcheck-dashboard authcheck-lite authcheck-pg-api authcheck-pg-lite >/dev/null 2>&1 || true
  docker volume rm authcheck-cfg >/dev/null 2>&1 || true
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

assert_auth_enabled() {
  local name="$1" base="$2" meta code
  meta=$(curl -fsS "$base/api/v1/meta") || { echo "::error::$name: could not fetch /api/v1/meta"; return 1; }
  echo "$meta"
  if ! echo "$meta" | python3 -c "import sys,json; sys.exit(0 if json.load(sys.stdin).get('authDisabled') is not True else 1)"; then
    echo "::error::$name: /api/v1/meta reports authDisabled=true — a non-dev image was built with GO_BUILD_TAGS=authdisabled"
    return 1
  fi
  code=$(curl -s -o /dev/null -w "%{http_code}" "$base/api/v1/tenants/00000000-0000-0000-0000-000000000000/workers")
  if [ "$code" != "401" ] && [ "$code" != "403" ]; then
    echo "::error::$name: unauthenticated tenant request returned $code (expected 401/403) — auth is not enforced"
    return 1
  fi
  echo "$name: auth enabled (authDisabled=false, unauthenticated request rejected with $code)"
}

docker network create "$NET" >/dev/null

echo "::group::api"
docker volume create authcheck-cfg >/dev/null
docker run -d --name authcheck-pg-api --network "$NET" \
  -e POSTGRES_USER=hatchet -e POSTGRES_PASSWORD=hatchet -e POSTGRES_DB=hatchet postgres:15.6 >/dev/null
wait_for_pg authcheck-pg-api || { echo "::error::api postgres not ready"; exit 1; }
DBURL_API="postgresql://hatchet:hatchet@authcheck-pg-api:5432/hatchet?sslmode=disable"
docker run --rm --network "$NET" -e DATABASE_URL="$DBURL_API" "$MIGRATE_IMAGE" /hatchet/hatchet-migrate \
  || { echo "::error::api migrate failed"; exit 1; }
docker run --rm --network "$NET" -e DATABASE_URL="$DBURL_API" -v authcheck-cfg:/hatchet/config \
  "$ADMIN_IMAGE" /hatchet/hatchet-admin quickstart --skip certs --generated-config-dir /hatchet/config --overwrite=false \
  || { echo "::error::api quickstart failed"; exit 1; }
docker run -d --name authcheck-api --network "$NET" -e DATABASE_URL="$DBURL_API" \
  -e SERVER_MSGQUEUE_KIND=postgres -e SERVER_GRPC_INSECURE=true \
  -e SERVER_AUTH_COOKIE_INSECURE=true -e SERVER_AUTH_COOKIE_DOMAIN=localhost \
  -v authcheck-cfg:/hatchet/config -p "$API_PORT":8080 \
  "$API_IMAGE" /hatchet/hatchet-api --config /hatchet/config >/dev/null
if ! wait_for_url "http://localhost:$API_PORT/api/live"; then
  echo "::error::api never became ready"; docker logs authcheck-api 2>&1 | tail -60; exit 1
fi
assert_auth_enabled api "http://localhost:$API_PORT" || fail=1
echo "::endgroup::"

echo "::group::dashboard"
# The dashboard bundles the api binary behind nginx (port 80); reuse the api's DB + config.
docker run -d --name authcheck-dashboard --network "$NET" -e DATABASE_URL="$DBURL_API" \
  -e SERVER_MSGQUEUE_KIND=postgres -e SERVER_GRPC_INSECURE=true \
  -e SERVER_AUTH_COOKIE_INSECURE=true -e SERVER_AUTH_COOKIE_DOMAIN=localhost \
  -v authcheck-cfg:/hatchet/config -p "$DASHBOARD_PORT":80 \
  "$DASHBOARD_IMAGE" sh ./entrypoint.sh --config /hatchet/config >/dev/null
if ! wait_for_url "http://localhost:$DASHBOARD_PORT/api/live"; then
  echo "::error::dashboard never became ready"; docker logs authcheck-dashboard 2>&1 | tail -60; exit 1
fi
assert_auth_enabled dashboard "http://localhost:$DASHBOARD_PORT" || fail=1
echo "::endgroup::"

echo "::group::lite"
docker run -d --name authcheck-pg-lite --network "$NET" \
  -e POSTGRES_USER=hatchet -e POSTGRES_PASSWORD=hatchet -e POSTGRES_DB=hatchet postgres:15.6 >/dev/null
wait_for_pg authcheck-pg-lite || { echo "::error::lite postgres not ready"; exit 1; }
docker run -d --name authcheck-lite --network "$NET" \
  -e DATABASE_URL="postgresql://hatchet:hatchet@authcheck-pg-lite:5432/hatchet?sslmode=disable" \
  -e SERVER_GRPC_BIND_ADDRESS=0.0.0.0 -e SERVER_GRPC_BROADCAST_ADDRESS=localhost:7070 \
  -e SERVER_GRPC_INSECURE=true -e SERVER_AUTH_COOKIE_INSECURE=true -e SERVER_AUTH_COOKIE_DOMAIN=localhost \
  -p "$LITE_PORT":8888 \
  "$LITE_IMAGE" >/dev/null
if ! wait_for_url "http://localhost:$LITE_PORT/api/live"; then
  echo "::error::lite never became ready"; docker logs authcheck-lite 2>&1 | tail -60; exit 1
fi
assert_auth_enabled lite "http://localhost:$LITE_PORT" || fail=1
echo "::endgroup::"

exit $fail

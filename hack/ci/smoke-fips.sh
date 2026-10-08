#!/usr/bin/env bash
# Boots a -fips lite image with GODEBUG=fips140=only against a SCRAM-only Postgres and exercises the
# paths FIPS touches: migrations, keyset generation, session cookies, password hashing.
set -uo pipefail

LITE_IMAGE="${LITE_IMAGE:-hatchet-lite-fips:ci}"
LITE_PORT="${LITE_PORT:-8889}"

NET=hatchet-fipssmoke
fail=0

cleanup() {
  docker rm -f fipssmoke-lite fipssmoke-pg >/dev/null 2>&1 || true
  docker network rm "$NET" >/dev/null 2>&1 || true
}
cleanup
trap cleanup EXIT

wait_for_url() {
  for _ in $(seq 1 60); do curl -fsS "$1" >/dev/null 2>&1 && return 0; sleep 2; done
  return 1
}

docker network create "$NET" >/dev/null
docker run -d --name fipssmoke-pg --network "$NET" \
  -e POSTGRES_USER=hatchet -e POSTGRES_PASSWORD=hatchet -e POSTGRES_DB=hatchet \
  -e POSTGRES_INITDB_ARGS="--auth-host=scram-sha-256 --auth-local=scram-sha-256" \
  postgres:15.6 -c password_encryption=scram-sha-256 >/dev/null
for _ in $(seq 1 30); do docker exec fipssmoke-pg pg_isready -U hatchet >/dev/null 2>&1 && break; sleep 2; done

docker run -d --name fipssmoke-lite --network "$NET" \
  -e GODEBUG=fips140=only \
  -e DATABASE_URL="postgresql://hatchet:hatchet@fipssmoke-pg:5432/hatchet?sslmode=disable" \
  -e SERVER_GRPC_BIND_ADDRESS=0.0.0.0 -e SERVER_GRPC_BROADCAST_ADDRESS=localhost:7070 \
  -e SERVER_GRPC_INSECURE=true -e SERVER_AUTH_COOKIE_INSECURE=true -e SERVER_AUTH_COOKIE_DOMAIN=localhost \
  -e SERVER_AUTH_SET_EMAIL_VERIFIED=true -e SERVER_URL="http://localhost:$LITE_PORT" \
  -p "$LITE_PORT":8888 "$LITE_IMAGE" >/dev/null

if ! wait_for_url "http://localhost:$LITE_PORT/api/ready"; then
  echo "::error::lite never became ready under fips140=only"; docker logs fipssmoke-lite 2>&1 | tail -80; exit 1
fi

base="http://localhost:$LITE_PORT"
ck=$(mktemp)
code=$(curl -s -o /dev/null -w '%{http_code}' -H 'Content-Type: application/json' -X POST "$base/api/v1/users/register" \
  -d '{"name":"FIPS Smoke","email":"fips-smoke@example.com","password":"Sm0ke-Test-Passw0rd"}')
[ "$code" = 200 ] && echo "register: $code" || { echo "::error::register returned $code"; fail=1; }
code=$(curl -s -o /dev/null -w '%{http_code}' -c "$ck" -H 'Content-Type: application/json' -X POST "$base/api/v1/users/login" \
  -d '{"email":"fips-smoke@example.com","password":"Sm0ke-Test-Passw0rd"}')
[ "$code" = 200 ] && echo "login: $code" || { echo "::error::login returned $code"; fail=1; }
code=$(curl -s -o /dev/null -w '%{http_code}' -b "$ck" "$base/api/v1/users/current")
[ "$code" = 200 ] && echo "current user with session cookie: $code" || { echo "::error::/users/current returned $code"; fail=1; }
code=$(curl -s -o /dev/null -w '%{http_code}' -H 'Content-Type: application/json' -X POST "$base/api/v1/users/login" \
  -d '{"email":"fips-smoke@example.com","password":"wrong"}')
[ "$code" = 400 ] && echo "wrong password rejected: $code" || { echo "::error::wrong password returned $code"; fail=1; }
printf '%s\n' 'R3set-Passw0rd-FIPS' | docker exec -i fipssmoke-lite ./hatchet-admin user set-password --config ./config --email fips-smoke@example.com --password-stdin \
  && echo "hatchet-admin user set-password: ok" || { echo "::error::hatchet-admin user set-password failed"; fail=1; }
code=$(curl -s -o /dev/null -w '%{http_code}' -H 'Content-Type: application/json' -X POST "$base/api/v1/users/login" \
  -d '{"email":"fips-smoke@example.com","password":"R3set-Passw0rd-FIPS"}')
[ "$code" = 200 ] && echo "login with reset password: $code" || { echo "::error::login with reset password returned $code"; fail=1; }
rm -f "$ck"

if docker logs fipssmoke-lite 2>&1 | grep -q 'panic:'; then
  echo "::error::lite panicked under fips140=only"; docker logs fipssmoke-lite 2>&1 | grep -A20 'panic:' | head -40; fail=1
fi

exit $fail

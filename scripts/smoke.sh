#!/usr/bin/env bash
# End-to-end check of the three binaries against local dependencies:
#
#   API  ->  PostgreSQL
#        ->  Pub/Sub  ->  worker  ->  PostgreSQL
#
# Needs Postgres and the Pub/Sub emulator running with the topic and
# subscription created (`make up`), and the variables from .env.
# Run it with `make smoke`.
set -euo pipefail

base="http://localhost:${PORT:-8080}"
auth='Authorization: Bearer smoke-user'
body="$(mktemp)"

go build -o bin/ ./cmd/...

./bin/job migrate

./bin/api &
api=$!
./bin/worker &
worker=$!

cleanup() {
	kill -TERM "$api" "$worker" 2>/dev/null || true
	wait "$api" "$worker" 2>/dev/null || true
	rm -f "$body"
}
trap cleanup EXIT

fail() {
	echo "smoke: $*" >&2
	exit 1
}

for _ in $(seq 1 30); do
	curl -sf -o /dev/null "$base/readyz" && break
	sleep 1
done
curl -sf -o /dev/null "$base/readyz" || fail "the API did not become ready"

# Unauthenticated requests are refused.
code="$(curl -s -o "$body" -w '%{http_code}' "$base/v1/orders/anything")"
[ "$code" = "401" ] || fail "request without a token returned $code, want 401"

# Creating an order stores it and publishes order.created.
id="$(curl -sf "$base/v1/orders" -H "$auth" -d '{"product_id":"p1","quantity":2,"amount":5000}' | jq -r .id)"
[ -n "$id" ] && [ "$id" != "null" ] || fail "create order returned no id"
echo "smoke: created order $id"

# The worker receives the event, charges the order and marks it paid.
status=""
for _ in $(seq 1 30); do
	status="$(curl -sf "$base/v1/orders/$id" -H "$auth" | jq -r .status)"
	[ "$status" = "paid" ] && break
	sleep 1
done
[ "$status" = "paid" ] || fail "order is '$status' after 30s, want 'paid'"
echo "smoke: the worker paid the order"

# A business rule answers with its own code and a translated message.
code="$(curl -s -o "$body" -w '%{http_code}' -X POST "$base/v1/orders/$id/cancel" -H "$auth" -H 'Accept-Language: id')"
[ "$code" = "409" ] || fail "cancel of a paid order returned $code, want 409: $(cat "$body")"
[ "$(jq -r .code "$body")" = "ORDER_CANCEL_NOT-ALLOWED" ] || fail "unexpected error body: $(cat "$body")"
[ "$(jq -r 'keys | join(",")' "$body")" = "code,message" ] || fail "error body is not exactly {code, message}: $(cat "$body")"
echo "smoke: error response $(cat "$body")"

# A validation failure is a 422 with a specific code.
code="$(curl -s -o "$body" -w '%{http_code}' "$base/v1/orders" -H "$auth" -d '{"product_id":"p1","quantity":0,"amount":5000}')"
[ "$code" = "422" ] || fail "invalid quantity returned $code, want 422"
[ "$(jq -r .code "$body")" = "ORDER_CREATE_QUANTITY-INVALID" ] || fail "unexpected error body: $(cat "$body")"

# A body that cannot be read is a 400.
code="$(curl -s -o "$body" -w '%{http_code}' "$base/v1/orders" -H "$auth" -d '{"product_id":')"
[ "$code" = "400" ] || fail "malformed body returned $code, want 400"

./bin/job expire-orders

# Both long-running binaries stop cleanly on SIGTERM.
kill -TERM "$api" "$worker"
wait "$api" || fail "the API exited with status $?"
wait "$worker" || fail "the worker exited with status $?"
trap - EXIT
rm -f "$body"

echo "smoke: passed"

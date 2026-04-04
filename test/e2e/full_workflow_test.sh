#!/usr/bin/env bash
# Full E2E workflow test — pure curl/bash, no Go required.
# Tests every API route through the HTTP gateway end-to-end.
#
# Environment variables (all optional):
#   API_GATEWAY_URL               default: http://localhost:8080
#   MANAGER_USERNAME / _PASSWORD  default: lms-manager / manager@413
#   ADMIN_USERNAME / _PASSWORD    default: lms-admin / @dm1n79
#   WAIT_TIMEOUT_SECONDS          default: 180
#   KEEP_STACK=1                  skip docker compose down on exit
#   SKIP_START=1                  skip docker compose up (services already running)
#   STRICT_DB_CHECK=1             fail on postgres validation errors
#   NOTIFICATION_REQUIRE_SUCCESS=1 fail if email was not actually delivered
#   NOTIFICATION_CONTAINER        default: lms-notification-service

set -euo pipefail

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
COMPOSE_FILE="${REPO_ROOT}/test/docker-compose.test.yaml"

API_GATEWAY_URL="${API_GATEWAY_URL:-http://localhost:8080}"
WAIT_TIMEOUT_SECONDS="${WAIT_TIMEOUT_SECONDS:-180}"
KEEP_STACK="${KEEP_STACK:-0}"
SKIP_START="${SKIP_START:-0}"
STRICT_DB_CHECK="${STRICT_DB_CHECK:-0}"
NOTIFICATION_REQUIRE_SUCCESS="${NOTIFICATION_REQUIRE_SUCCESS:-0}"
NOTIFICATION_CONTAINER="${NOTIFICATION_CONTAINER:-lms-notification-service}"
MANAGER_USERNAME="lms-manager"
MANAGER_PASSWORD="manager@413"
ADMIN_USERNAME="lms-admin"
ADMIN_PASSWORD="@dm1n79"
# __________________________________________________
# Check for required commands before starting tests
# __________________________________________________
if ! command -v docker >/dev/null 2>&1; then
  echo -e "${RED}[ERROR] docker is not installed or not in PATH${NC}"
  exit 1
fi

if ! command -v curl >/dev/null 2>&1; then
  echo -e "${RED}[ERROR] curl is not installed or not in PATH${NC}"
  exit 1
fi

if ! command -v python3 >/dev/null 2>&1; then
  echo -e "${RED}[ERROR] python3 is not installed or not in PATH${NC}"
  exit 1
fi

if docker compose version >/dev/null 2>&1; then
  COMPOSE_CMD=(docker compose)
elif command -v docker-compose >/dev/null 2>&1; then
  COMPOSE_CMD=(docker-compose)
else
  echo -e "${RED}[ERROR] docker compose/docker-compose was not found${NC}"
  exit 1
fi

# _____________________________________________________________
# Shared state (populated during test run)
# _____________________________________________________________
RESP_STATUS=""
RESP_BODY=""
RESP_BODY_FILE="/tmp/lms_e2e_resp_body.txt"

# Unique suffix for usernames / ISBNs to avoid conflicts on re-runs.
# Use nanoseconds when available (Linux), fall back to seconds+RANDOM.
TMP_SUFFIX="$(date +%s%N 2>/dev/null || echo "$(date +%s)${RANDOM}")"
TMP_SUFFIX="${TMP_SUFFIX: -9}"   # keep last 9 digits

USER_TOKEN=""
USER_ID=""
USER_USERNAME=""
MANAGER_TOKEN=""
ADMIN_TOKEN=""
BOOK_ID_1=""
BOOK_ID_2=""
ORDER_ID=""
CANCEL_ORDER_ID=""

# _____________________________________________________________
# Logging helpers
# _____________________________________________________________
log_section() { echo -e "\n${BLUE}══════════════════════════════════════════════${NC}"; echo -e "${BLUE}  $*${NC}"; echo -e "${BLUE}══════════════════════════════════════════════${NC}"; }
log_info()    { echo -e "${YELLOW}[INFO] $*${NC}"; }
log_pass()    { echo -e "${GREEN}[PASS] $*${NC}"; }
log_fail()    { echo -e "${RED}[FAIL] $*${NC}"; }

# _____________________________________________________________
# Assertion helpers
# _____________________________________________________________
assert_equal() {
  local expected="$1"
  local actual="$2"
  local message="$3"

  if [[ "${expected}" == "${actual}" ]]; then
    log_pass "${message}"
  else
    log_fail "${message} (expected=${expected}, actual=${actual})"
    log_fail "Response body: $(cat "${RESP_BODY_FILE}" 2>/dev/null || echo '<none>')"
    exit 1
  fi
}

assert_not_empty() {
  local value="$1"
  local message="$2"

  if [[ -n "${value}" ]]; then
    log_pass "${message}"
  else
    log_fail "${message} (value is empty)"
    log_fail "Response body: $(cat "${RESP_BODY_FILE}" 2>/dev/null || echo '<none>')"
    exit 1
  fi
}

assert_contains() {
  local haystack="$1"
  local needle="$2"
  local message="$3"

  if [[ "${haystack}" == *"${needle}"* ]]; then
    log_pass "${message}"
  else
    log_fail "${message} (missing '${needle}')"
    log_fail "Haystack: ${haystack}"
    exit 1
  fi
}

# Assert HTTP status is in [400, 499]
assert_4xx() {
  local actual="$1"
  local message="$2"

  if [[ "${actual}" -ge 400 && "${actual}" -lt 500 ]]; then
    log_pass "${message} (status=${actual})"
  else
    log_fail "${message} (expected 4xx, got ${actual})"
    log_fail "Response body: $(cat "${RESP_BODY_FILE}" 2>/dev/null || echo '<none>')"
    exit 1
  fi
}

# Assert numeric actual >= expected minimum
assert_ge() {
  local min="$1"
  local actual="$2"
  local message="$3"

  if python3 -c "import sys; sys.exit(0 if int(sys.argv[1]) >= int(sys.argv[2]) else 1)" \
       "${actual}" "${min}" 2>/dev/null; then
    log_pass "${message} (${actual} >= ${min})"
  else
    log_fail "${message} (expected >= ${min}, got ${actual})"
    exit 1
  fi
}

# ______________________________________________________________
# HTTP helper
#   api_call METHOD PATH [BODY] [TOKEN]
#   Sets globals: RESP_STATUS, RESP_BODY
# _____________________________________________________________
api_call() {
  local method="$1"
  local path="$2"
  local body="${3:-}"
  local token="${4:-}"
  local url="${API_GATEWAY_URL}${path}"

  local -a curl_args=(-sS -o "${RESP_BODY_FILE}" -w "%{http_code}" -X "${method}")

  if [[ -n "${token}" ]]; then
    curl_args+=(-H "Authorization: Bearer ${token}")
  fi

  if [[ -n "${body}" ]]; then
    curl_args+=(-H "Content-Type: application/json" -d "${body}")
  fi

  RESP_STATUS="$(curl "${curl_args[@]}" "${url}")"
  RESP_BODY="$(cat "${RESP_BODY_FILE}")"
}

# _____________________________________________________________
# JSON field extractor (dot-separated path, e.g. "order.id", "created_books.0.id")
# Usage: json_get KEY [JSON_STRING]
#        Omit JSON_STRING to use $RESP_BODY
# _____________________________________________________________
json_get() {
  local key="$1"
  local json="${2:-${RESP_BODY}}"

  python3 - "${json}" "${key}" <<'PY'
import json, sys

data = json.loads(sys.argv[1])
parts = sys.argv[2].split(".")
v = data
for p in parts:
    if isinstance(v, list):
        v = v[int(p)]
    else:
        v = v[p]

if isinstance(v, bool):
    print("true" if v else "false")
elif isinstance(v, (list, dict)):
    print(json.dumps(v))
else:
    print(v)
PY
}

# _____________________________________________________________
# Postgres helpers (used only when STRICT_DB_CHECK=1)
# _____________________________________________________________
query_postgres() {
  local container="$1"
  local database="$2"
  local sql="$3"

  local out=""
  set +e
  out="$(docker exec -e PGPASSWORD=postgres "${container}" \
    psql -U postgres -d "${database}" -X -qAt -v ON_ERROR_STOP=1 -c "${sql}" 2>/dev/null)"
  local rc=$?
  set -e

  [[ ${rc} -eq 0 ]] && printf "%s" "${out}" | tr -d '\r' | xargs || echo ""
}

eventually_equal() {
  local expected="$1"
  local timeout_s="$2"
  local message="$3"
  shift 3

  local deadline=$((SECONDS + timeout_s))
  local value=""

  while (( SECONDS < deadline )); do
    value="$("$@" 2>/dev/null || echo "")"
    if [[ "${value}" == "${expected}" ]]; then
      log_pass "${message}"
      return
    fi
    sleep 1
  done

  log_fail "${message} (expected=${expected}, actual=${value})"
  [[ "${STRICT_DB_CHECK}" == "1" ]] && exit 1 || true
}

# _____________________________________________________________
# Optional cross-database consistency check
# _____________________________________________________________
validate_cross_db() {
  eventually_equal "1" 20 "User exists in user_db" \
    query_postgres postgres-user user_db \
    "SELECT COUNT(1) FROM users WHERE id='${USER_ID}';"

  eventually_equal "1" 20 "Book 1 exists in book_db" \
    query_postgres postgres-book book_db \
    "SELECT COUNT(1) FROM books WHERE id='${BOOK_ID_1}';"

  eventually_equal "1" 20 "Book 2 exists in book_db" \
    query_postgres postgres-book book_db \
    "SELECT COUNT(1) FROM books WHERE id='${BOOK_ID_2}';"

  eventually_equal "1" 20 "Order exists in order_db" \
    query_postgres postgres-order order_db \
    "SELECT COUNT(1) FROM orders WHERE id='${ORDER_ID}';"

  eventually_equal "${USER_ID}" 20 "Order belongs to correct user in order_db" \
    query_postgres postgres-order order_db \
    "SELECT user_id FROM orders WHERE id='${ORDER_ID}' LIMIT 1;"

  eventually_equal "1" 20 "order_books links order and book in order_db" \
    query_postgres postgres-order order_db \
    "SELECT COUNT(1) FROM order_books WHERE order_id='${ORDER_ID}' AND book_id='${BOOK_ID_1}';"
}

# _____________________________________________________________
# Stack management
# _____________________________________________________________
cleanup() {
  [[ "${KEEP_STACK}" == "1" ]] && { log_info "KEEP_STACK=1 — skipping docker compose down"; return; }
  log_info "Stopping test stack"
  "${COMPOSE_CMD[@]}" -f "${COMPOSE_FILE}" down >/dev/null 2>&1 || true
}

wait_for_gateway() {
  log_info "Waiting for gateway at ${API_GATEWAY_URL}/healthy (timeout=${WAIT_TIMEOUT_SECONDS}s)"
  local deadline=$((SECONDS + WAIT_TIMEOUT_SECONDS))

  while (( SECONDS < deadline )); do
    local code
    code="$(curl -sS -o /dev/null -w "%{http_code}" "${API_GATEWAY_URL}/healthy" 2>/dev/null || echo 000)"
    [[ "${code}" == "200" ]] && { log_pass "Gateway is healthy"; break; }
    sleep 2
  done

  if (( SECONDS >= deadline )); then
    log_fail "Gateway did not become healthy within ${WAIT_TIMEOUT_SECONDS}s"
    exit 1
  fi

  log_info "Waiting for gateway readiness at ${API_GATEWAY_URL}/ready (all upstream gRPC services must be up)"
  while (( SECONDS < deadline )); do
    local code
    code="$(curl -sS -o /dev/null -w "%{http_code}" "${API_GATEWAY_URL}/ready" 2>/dev/null || echo 000)"
    [[ "${code}" == "200" ]] && { log_pass "Gateway is ready (all upstream services healthy)"; return; }
    sleep 2
  done

  log_fail "Gateway did not become ready within ${WAIT_TIMEOUT_SECONDS}s (upstream gRPC services may still be starting)"
  exit 1
}

# ═════════════════════════════════════════════════════════════
# MAIN
# ═════════════════════════════════════════════════════════════
main() {
  trap cleanup EXIT

  echo "══════════════════════════════════════════════"
  echo "   FULL WORKFLOW E2E TEST  (curl-only)        "
  echo "   $(date '+%Y-%m-%d %H:%M:%S')               "
  echo "══════════════════════════════════════════════"

  if [[ ! -f "${COMPOSE_FILE}" ]]; then
    log_fail "Compose file not found: ${COMPOSE_FILE}"
    exit 1
  fi

  if [[ "${SKIP_START}" != "1" ]]; then
    log_info "Starting test stack from ${COMPOSE_FILE}"
    "${COMPOSE_CMD[@]}" -f "${COMPOSE_FILE}" up -d
  else
    log_info "SKIP_START=1 — assuming services are already running"
  fi

  wait_for_gateway

  # ───────────────────────────────────────────────
  # Phase 0: Infrastructure health
  # ───────────────────────────────────────────────
  log_section "Phase 0: Infrastructure Health"

  api_call GET /healthy
  assert_equal "200" "${RESP_STATUS}" "GET /healthy → 200"
  assert_contains "${RESP_BODY}" "healthy" "Health body contains 'healthy'"

  api_call GET /ready
  assert_equal "200" "${RESP_STATUS}" "GET /ready → 200"

  # ───────────────────────────────────────────────
  # Phase 1: Authentication
  # ───────────────────────────────────────────────
  log_section "Phase 1: Authentication"

  # 1a. Register a new user
  USER_USERNAME="e2e-user-${TMP_SUFFIX}"
  local user_email="e2e-${TMP_SUFFIX}@test.local"
  local user_password="E2eTest!Pass1"

  api_call POST /api/v1/auth/register \
    "{\"username\":\"${USER_USERNAME}\",\"password\":\"${user_password}\",\"email\":\"${user_email}\",\"phone_number\":\"0900000001\"}"
  assert_equal "201" "${RESP_STATUS}" "POST /auth/register -> 201"
  USER_ID="$(json_get user_id)"
  assert_not_empty "${USER_ID}" "Register returns user_id"
  log_info "Registered user: username=${USER_USERNAME} id=${USER_ID}"

  # 1b. Login as user
  api_call POST /api/v1/auth/login \
    "{\"identifier\":\"${USER_USERNAME}\",\"password\":\"${user_password}\"}"
  assert_equal "200" "${RESP_STATUS}" "POST /auth/login (user) -> 200"
  USER_TOKEN="$(json_get token_pair.access_token)"
  assert_not_empty "${USER_TOKEN}" "Login returns access_token"
  log_info "User token acquired"

  # 1c. Login as manager
  api_call POST /api/v1/auth/login \
    "{\"identifier\":\"${MANAGER_USERNAME}\",\"password\":\"${MANAGER_PASSWORD}\"}"
  assert_equal "200" "${RESP_STATUS}" "POST /auth/login (manager) -> 200"
  MANAGER_TOKEN="$(json_get token_pair.access_token)"
  assert_not_empty "${MANAGER_TOKEN}" "Manager login returns access_token"
  log_info "Manager token acquired"

  # 1d. Login as admin
  api_call POST /api/v1/auth/login \
    "{\"identifier\":\"${ADMIN_USERNAME}\",\"password\":\"${ADMIN_PASSWORD}\"}"
  assert_equal "200" "${RESP_STATUS}" "POST /auth/login (admin) -> 200"
  ADMIN_TOKEN="$(json_get token_pair.access_token)"
  assert_not_empty "${ADMIN_TOKEN}" "Admin login returns access_token"
  log_info "Admin token acquired"

  # 1e. Duplicate registration → 409
  api_call POST /api/v1/auth/register \
    "{\"username\":\"${USER_USERNAME}\",\"password\":\"${user_password}\",\"email\":\"${user_email}\",\"phone_number\":\"0900000001\"}"
  assert_equal "409" "${RESP_STATUS}" "POST /auth/register (duplicate) -> 409"

  # ───────────────────────────────────────────────
  # Phase 2: User profile
  # ───────────────────────────────────────────────
  log_section "Phase 2: User Profile"

  api_call GET /api/v1/user/profile "" "${USER_TOKEN}"
  assert_equal "200" "${RESP_STATUS}" "GET /user/profile -> 200"
  local profile_username
  profile_username="$(json_get user.username)"
  assert_equal "${USER_USERNAME}" "${profile_username}" "Profile username matches"

  api_call PATCH /api/v1/user/profile \
    '{"phone_number":"0911111111"}' "${USER_TOKEN}"
  assert_equal "200" "${RESP_STATUS}" "PATCH /user/profile -> 200"

  # Protected route without token → 401
  api_call GET /api/v1/user/profile
  assert_equal "401" "${RESP_STATUS}" "GET /user/profile (no token) -> 401"

  # ───────────────────────────────────────────────
  # Phase 3: Book management
  # ───────────────────────────────────────────────
  log_section "Phase 3: Book Management"

  # 3a. Create books (manager)
  local isbn1="978-E2E-${TMP_SUFFIX}-A"
  local isbn2="978-E2E-${TMP_SUFFIX}-B"
  api_call POST /api/v1/management/books \
    "{\"books_payload\":[{\"title\":\"E2E Book Alpha\",\"author\":\"E2E Author\",\"isbn\":\"${isbn1}\",\"category\":\"Test\",\"description\":\"E2E test book A\",\"quantity\":10},{\"title\":\"E2E Book Beta\",\"author\":\"E2E Author\",\"isbn\":\"${isbn2}\",\"category\":\"Test\",\"description\":\"E2E test book B\",\"quantity\":5}]}" \
    "${MANAGER_TOKEN}"
  assert_equal "201" "${RESP_STATUS}" "POST /management/books -> 201"
  BOOK_ID_1="$(json_get created_books.0.id)"
  BOOK_ID_2="$(json_get created_books.1.id)"
  assert_not_empty "${BOOK_ID_1}" "Created book 1 has ID"
  assert_not_empty "${BOOK_ID_2}" "Created book 2 has ID"
  log_info "Created books: BOOK_ID_1=${BOOK_ID_1}  BOOK_ID_2=${BOOK_ID_2}"

  # 3b. List books (public — no token needed)
  api_call GET /api/v1/books
  assert_equal "200" "${RESP_STATUS}" "GET /books (public) -> 200"
  local book_list_count
  book_list_count="$(json_get total_count)"
  assert_ge 1 "${book_list_count}" "Book list has at least 1 entry"

  # 3c. Get book by ID (public)
  api_call GET "/api/v1/books/${BOOK_ID_1}"
  assert_equal "200" "${RESP_STATUS}" "GET /books/:id -> 200"
  local got_book_id
  got_book_id="$(json_get book.id)"
  assert_equal "${BOOK_ID_1}" "${got_book_id}" "GET /books/:id returns correct book"

  # 3d. Get non-existent book -> 404
  api_call GET "/api/v1/books/00000000-0000-0000-0000-000000000000"
  assert_equal "404" "${RESP_STATUS}" "GET /books/non-existent -> 404"

  # 3e. Update book title/author (manager)
  api_call PUT "/api/v1/management/books/${BOOK_ID_1}" \
    '{"title":"E2E Book Alpha (Updated)","author":"E2E Author v2"}' \
    "${MANAGER_TOKEN}"
  assert_equal "200" "${RESP_STATUS}" "PUT /management/books/:id -> 200"

  # 3f. Update book quantity (manager)
  api_call PATCH "/api/v1/management/books/${BOOK_ID_1}/quantity" \
    '{"change_amount":2}' "${MANAGER_TOKEN}"
  assert_equal "200" "${RESP_STATUS}" "PATCH /management/books/:id/quantity -> 200"

  # 3g. Check book availability (manager)
  api_call GET "/api/v1/management/books/${BOOK_ID_1}/availability" "" "${MANAGER_TOKEN}"
  assert_equal "200" "${RESP_STATUS}" "GET /management/books/:id/availability -> 200"
  log_info "Book 1 availability: $(json_get available_quantity) available"

  # ───────────────────────────────────────────────
  # Phase 4: Order flow
  # ───────────────────────────────────────────────
  log_section "Phase 4: Order Flow"

  # 4a. Create order 1 (to be approved → borrowed)
  api_call POST /api/v1/orders \
    "{\"book_ids\":[\"${BOOK_ID_1}\"],\"borrow_days\":7}" \
    "${USER_TOKEN}"
  assert_equal "201" "${RESP_STATUS}" "POST /orders (order 1) -> 201"
  ORDER_ID="$(json_get order.id)"
  local order_status
  order_status="$(json_get order.status)"
  assert_not_empty "${ORDER_ID}" "Order 1 has ID"
  assert_equal "PENDING" "${order_status}" "Order 1 initial status is PENDING"
  log_info "Order 1 created: id=${ORDER_ID}"

  # 4b. Create order 2 (to be cancelled)
  api_call POST /api/v1/orders \
    "{\"book_ids\":[\"${BOOK_ID_2}\"],\"borrow_days\":3}" \
    "${USER_TOKEN}"
  assert_equal "201" "${RESP_STATUS}" "POST /orders (order 2) -> 201"
  CANCEL_ORDER_ID="$(json_get order.id)"
  assert_not_empty "${CANCEL_ORDER_ID}" "Order 2 has ID"
  log_info "Order 2 created: id=${CANCEL_ORDER_ID}"

  # 4c. List my orders
  api_call GET /api/v1/orders "" "${USER_TOKEN}"
  assert_equal "200" "${RESP_STATUS}" "GET /orders (list my orders) -> 200"
  local my_order_count
  my_order_count="$(json_get total_count)"
  assert_ge 1 "${my_order_count}" "User has at least 1 order"

  # 4d. Get order by ID
  api_call GET "/api/v1/orders/${ORDER_ID}" "" "${USER_TOKEN}"
  assert_equal "200" "${RESP_STATUS}" "GET /orders/:id -> 200"
  local got_order_id
  got_order_id="$(json_get order.id)"
  assert_equal "${ORDER_ID}" "${got_order_id}" "GET /orders/:id returns correct order"

  # 4e. Cancel order 2
  api_call POST "/api/v1/orders/${CANCEL_ORDER_ID}/cancel" \
    '{"cancel_reason":"e2e test cancellation"}' \
    "${USER_TOKEN}"
  assert_equal "200" "${RESP_STATUS}" "POST /orders/:id/cancel -> 200"
  local cancel_status
  cancel_status="$(json_get order.status)"
  assert_equal "CANCELED" "${cancel_status}" "Cancelled order status is CANCELED"

  # 4f. Manager lists all orders
  api_call GET /api/v1/management/orders "" "${MANAGER_TOKEN}"
  assert_equal "200" "${RESP_STATUS}" "GET /management/orders -> 200"
  local all_order_count
  all_order_count="$(json_get total_count)"
  assert_ge 1 "${all_order_count}" "System has at least 1 order"

  # 4g. Manager approves order 1
  api_call PATCH "/api/v1/management/orders/${ORDER_ID}/status" \
    '{"new_status":"APPROVED"}' "${MANAGER_TOKEN}"
  assert_equal "200" "${RESP_STATUS}" "PATCH /management/orders/:id/status (APPROVED) -> 200"
  local approved_status
  approved_status="$(json_get order.status)"
  assert_equal "APPROVED" "${approved_status}" "Order 1 status is APPROVED"

  # 4h. Manager marks order 1 as BORROWED
  api_call PATCH "/api/v1/management/orders/${ORDER_ID}/status" \
    '{"new_status":"BORROWED","note":"collected at front desk"}' "${MANAGER_TOKEN}"
  assert_equal "200" "${RESP_STATUS}" "PATCH /management/orders/:id/status (BORROWED) -> 200"
  local borrowed_status
  borrowed_status="$(json_get order.status)"
  assert_equal "BORROWED" "${borrowed_status}" "Order 1 status is BORROWED"

  # ───────────────────────────────────────────────
  # Phase 5: Service resilience / error handling
  # ───────────────────────────────────────────────
  log_section "Phase 5: Service Resilience"

  # 5a. Order with non-existent book → 4xx (not 500)
  api_call POST /api/v1/orders \
    '{"book_ids":["00000000-0000-0000-0000-000000000000"],"borrow_days":7}' \
    "${USER_TOKEN}"
  assert_4xx "${RESP_STATUS}" "Order with non-existent book returns 4xx (not 500)"

  # 5b. Invalid JSON body → 400
  RESP_STATUS="$(curl -sS -o "${RESP_BODY_FILE}" -w "%{http_code}" \
    -X POST \
    -H "Authorization: Bearer ${USER_TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{this is not json}' \
    "${API_GATEWAY_URL}/api/v1/orders")"
  assert_equal "400" "${RESP_STATUS}" "Invalid JSON body -> 400"

  # 5c. Missing required fields → 400
  api_call POST /api/v1/orders '{"borrow_days":7}' "${USER_TOKEN}"
  assert_equal "400" "${RESP_STATUS}" "Missing book_ids in CreateOrder -> 400"

  # ───────────────────────────────────────────────
  # Phase 6: Role-Based Access Control
  # ───────────────────────────────────────────────
  log_section "Phase 6: Role-Based Access Control"

  # USER denied on management routes
  for path in /api/v1/management/users /api/v1/management/orders; do
    api_call GET "${path}" "" "${USER_TOKEN}"
    assert_equal "403" "${RESP_STATUS}" "USER on ${path} -> 403"
  done

  # USER denied on admin-only route
  api_call PATCH "/api/v1/admin/users/${USER_ID}/vip" \
    '{"is_vip":true}' "${USER_TOKEN}"
  assert_equal "403" "${RESP_STATUS}" "USER on /admin/users/:id/vip -> 403"

  # MANAGER can access management routes
  for path in /api/v1/management/users /api/v1/management/orders; do
    api_call GET "${path}" "" "${MANAGER_TOKEN}"
    assert_equal "200" "${RESP_STATUS}" "MANAGER on ${path} -> 200"
  done

  # MANAGER denied on admin-only route
  api_call PATCH "/api/v1/admin/users/${USER_ID}/vip" \
    '{"is_vip":true}' "${MANAGER_TOKEN}"
  assert_equal "403" "${RESP_STATUS}" "MANAGER on /admin/users/:id/vip -> 403"

  # ADMIN can access management routes
  api_call GET /api/v1/management/orders "" "${ADMIN_TOKEN}"
  assert_equal "200" "${RESP_STATUS}" "ADMIN on /management/orders -> 200"

  # ADMIN can list users
  api_call GET /api/v1/management/users "" "${ADMIN_TOKEN}"
  assert_equal "200" "${RESP_STATUS}" "ADMIN on /management/users -> 200"

  # ADMIN can grant VIP status
  api_call PATCH "/api/v1/admin/users/${USER_ID}/vip" \
    '{"is_vip":true}' "${ADMIN_TOKEN}"
  assert_equal "200" "${RESP_STATUS}" "ADMIN grants VIP on /admin/users/:id/vip -> 200"
  log_info "VIP status granted to user ${USER_ID}"

  # ADMIN can delete users (DELETE body required)
  # NOTE: we do NOT delete USER_ID as it is still used below; create a throwaway account
  local throwaway="e2e-del-${TMP_SUFFIX}"
  api_call POST /api/v1/auth/register \
    "{\"username\":\"${throwaway}\",\"password\":\"D3le!te99\",\"email\":\"${throwaway}@test.local\",\"phone_number\":\"0900000002\"}"
  local throwaway_id
  throwaway_id="$(json_get user_id)"
  if [[ -n "${throwaway_id}" ]]; then
    api_call DELETE /api/v1/admin/users \
      "{\"user_ids\":[\"${throwaway_id}\"]}" "${ADMIN_TOKEN}"
    assert_equal "204" "${RESP_STATUS}" "ADMIN DELETE /admin/users -> 204"
  fi

  # ───────────────────────────────────────────────
  # Phase 7: Token lifecycle
  # ───────────────────────────────────────────────
  log_section "Phase 7: Token Lifecycle"

  # No token → 401
  api_call GET /api/v1/user/profile
  assert_equal "401" "${RESP_STATUS}" "No token on protected route -> 401"

  # Fully invalid token string → 401
  api_call GET /api/v1/user/profile "" "this.is.not.a.valid.token"
  assert_equal "401" "${RESP_STATUS}" "Invalid token string -> 401"

  # Malformed Authorization header (no 'Bearer' prefix) → 401
  RESP_STATUS="$(curl -sS -o "${RESP_BODY_FILE}" -w "%{http_code}" \
    -H "Authorization: notbearer ${USER_TOKEN}" \
    "${API_GATEWAY_URL}/api/v1/user/profile")"
  assert_equal "401" "${RESP_STATUS}" "Malformed Authorization header -> 401"

  # Tampered token: corrupt a character in the middle (not the last char,
  # which only carries padding bits in base64url and may be a no-op).
  local mid_idx=$(( ${#USER_TOKEN} / 2 ))
  local mid_char="${USER_TOKEN:${mid_idx}:1}"
  local replacement="Z"
  [[ "${mid_char}" == "Z" ]] && replacement="a"
  local tampered_token="${USER_TOKEN:0:${mid_idx}}${replacement}${USER_TOKEN:$((mid_idx + 1))}"
  api_call GET /api/v1/user/profile "" "${tampered_token}"
  assert_equal "401" "${RESP_STATUS}" "Tampered token (middle char) -> 401"

  # Public book route works without any token
  api_call GET /api/v1/books
  assert_equal "200" "${RESP_STATUS}" "GET /books (public, no token) -> 200"

  # Valid token still works after all the above
  api_call GET /api/v1/user/profile "" "${USER_TOKEN}"
  assert_equal "200" "${RESP_STATUS}" "Valid token still works after abuse attempts -> 200"

  # ───────────────────────────────────────────────
  # Phase 8: Rate-limit check (soft — informational only)
  # ───────────────────────────────────────────────
  log_section "Phase 8: Rate-Limit Check"

  local last_rl_code=""
  for _ in $(seq 1 5); do
    last_rl_code="$(curl -sS -o /dev/null -w "%{http_code}" \
      -H "Authorization: Bearer ${USER_TOKEN}" \
      "${API_GATEWAY_URL}/api/v1/user/profile")"
  done
  if [[ "${last_rl_code}" == "429" ]]; then
    log_pass "Rate limiter is active (received 429 after rapid requests)"
  else
    log_info "Rate limit not triggered in 5 quick requests (last status=${last_rl_code})"
  fi

  # ───────────────────────────────────────────────
  # Phase 9: Notification service
  # ───────────────────────────────────────────────
  log_section "Phase 9: Notification Service"

  local notif_start
  notif_start="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  sleep 1

  # Trigger a fresh order to produce an order.created event
  local isbn_notif="978-E2E-${TMP_SUFFIX}-N"
  api_call POST /api/v1/management/books \
    "{\"books_payload\":[{\"title\":\"E2E Notif Book\",\"author\":\"E2E Author\",\"isbn\":\"${isbn_notif}\",\"category\":\"Test\",\"quantity\":3}]}" \
    "${MANAGER_TOKEN}"
  local notif_book_id
  notif_book_id="$(json_get created_books.0.id 2>/dev/null || echo "")"

  if [[ -n "${notif_book_id}" ]]; then
    api_call POST /api/v1/orders \
      "{\"book_ids\":[\"${notif_book_id}\"],\"borrow_days\":5}" "${USER_TOKEN}"
    log_info "Triggered order.created event for notification service"
  fi

  local notif_deadline=$((SECONDS + 25))
  local notif_ok=0
  while (( SECONDS < notif_deadline )); do
    if docker logs --since "${notif_start}" "${NOTIFICATION_CONTAINER}" 2>&1 \
         | grep -qF "Received event: order.created"; then
      notif_ok=1
      break
    fi
    sleep 1
  done

  if (( notif_ok )); then
    log_pass "Notification service consumed order.created event"
    local notif_logs
    notif_logs="$(docker logs --since "${notif_start}" "${NOTIFICATION_CONTAINER}" 2>&1 || true)"
    if [[ "${NOTIFICATION_REQUIRE_SUCCESS}" == "1" ]]; then
      assert_contains "${notif_logs}" "Email sent to" "Notification service sent email"
    elif [[ "${notif_logs}" == *"Email sent to"* ]]; then
      log_pass "Notification service sent email successfully"
    elif [[ "${notif_logs}" == *"Failed to send email"* || "${notif_logs}" == *"Failed to process message"* ]]; then
      log_info "Notification service consumed event but email delivery failed in local env (expected)"
    fi
  else
    log_fail "Notification service did not consume order.created event within 25s"
    [[ "${NOTIFICATION_REQUIRE_SUCCESS}" == "1" ]] && exit 1 || true
  fi

  # ───────────────────────────────────────────────
  # Phase 10: Cross-DB validation (optional)
  # ───────────────────────────────────────────────
  if [[ "${STRICT_DB_CHECK}" == "1" ]]; then
    log_section "Phase 10: Cross-DB Validation"
    validate_cross_db
  fi

  echo ""
  echo "══════════════════════════════════════════════"
  log_pass "FULL WORKFLOW E2E TEST PASSED"
  echo "══════════════════════════════════════════════"
}

main "$@"


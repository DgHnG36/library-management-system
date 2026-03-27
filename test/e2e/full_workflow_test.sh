#!/usr/bin/env bash

set -euo pipefail

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
COMPOSE_FILE="${REPO_ROOT}/test/docker-compose.local.yaml"

API_GATEWAY_URL="${API_GATEWAY_URL:-http://localhost:8080}"
JWT_SECRET="${JWT_SECRET:-local-dev-secret-key}"
EXPECTED_PROXY_STATUS="${EXPECTED_PROXY_STATUS:-503}"
WAIT_TIMEOUT_SECONDS="${WAIT_TIMEOUT_SECONDS:-180}"
KEEP_STACK="${KEEP_STACK:-0}"
SKIP_START="${SKIP_START:-0}"
STRICT_DB_CHECK="${STRICT_DB_CHECK:-0}"
GRPC_USER_ADDR="${GRPC_USER_ADDR:-localhost:40041}"
GRPC_BOOK_ADDR="${GRPC_BOOK_ADDR:-localhost:40042}"
GRPC_ORDER_ADDR="${GRPC_ORDER_ADDR:-localhost:40043}"
NOTIFICATION_REQUIRE_SUCCESS="${NOTIFICATION_REQUIRE_SUCCESS:-0}"
FLOW_RESULT_JSON="/tmp/lms_e2e_flow_result.json"

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

log_info() {
  echo -e "${YELLOW}[INFO] $*${NC}"
}

log_pass() {
  echo -e "${GREEN}[PASS] $*${NC}"
}

log_fail() {
  echo -e "${RED}[FAIL] $*${NC}"
}

assert_equal() {
  local expected="$1"
  local actual="$2"
  local message="$3"

  if [[ "${expected}" == "${actual}" ]]; then
    log_pass "${message} (expected=${expected}, actual=${actual})"
  else
    log_fail "${message} (expected=${expected}, actual=${actual})"
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
    exit 1
  fi
}

assert_file_exists() {
  local path="$1"
  local message="$2"

  if [[ -f "${path}" ]]; then
    log_pass "${message}"
  else
    log_fail "${message} (missing file: ${path})"
    exit 1
  fi
}

http_status() {
  local method="$1"
  local url="$2"
  local auth_header="${3:-}"

  if [[ -n "${auth_header}" ]]; then
    curl -sS -o /tmp/lms_e2e_body.txt -w "%{http_code}" -X "${method}" -H "Authorization: ${auth_header}" "${url}"
  else
    curl -sS -o /tmp/lms_e2e_body.txt -w "%{http_code}" -X "${method}" "${url}"
  fi
}

generate_jwt() {
  python3 - <<'PY'
import base64
import hashlib
import hmac
import json
import os
import time

secret = os.environ.get("JWT_SECRET", "local-dev-secret-key").encode("utf-8")

header = {"alg": "HS256", "typ": "JWT"}
payload = {
    "user_id": "e2e-user",
    "role": "user",
    "email": "e2e@example.com",
    "iss": "lib-management-system",
    "aud": "gateway-service",
    "sub": "e2e-user",
    "iat": int(time.time()),
    "exp": int(time.time()) + 3600,
}

def b64url(data: bytes) -> bytes:
    return base64.urlsafe_b64encode(data).rstrip(b"=")

header_enc = b64url(json.dumps(header, separators=(",", ":")).encode("utf-8"))
payload_enc = b64url(json.dumps(payload, separators=(",", ":")).encode("utf-8"))
signing_input = header_enc + b"." + payload_enc
signature = b64url(hmac.new(secret, signing_input, hashlib.sha256).digest())

print((signing_input + b"." + signature).decode("utf-8"))
PY
}

query_postgres() {
  local container="$1"
  local database="$2"
  local sql="$3"

  local out
  local err
  local rc
  err="$(mktemp /tmp/lms_e2e_psql_err_XXXX.log)"

  set +e
  out="$(docker exec -e PGPASSWORD=postgres "${container}" \
    psql -U postgres -d "${database}" -X -qAt -v ON_ERROR_STOP=1 -c "${sql}" 2>"${err}")"
  rc=$?
  set -e

  if [[ ${rc} -ne 0 ]]; then
    if [[ "${STRICT_DB_CHECK}" == "1" ]]; then
      log_fail "SQL check failed (container=${container}, db=${database})"
      if [[ -s "${err}" ]]; then
        cat "${err}" >&2
      fi
      rm -f "${err}"
      exit 1
    fi

    rm -f "${err}"
    printf ""
    return 0
  fi

  rm -f "${err}"

  # Normalize psql output so assertions are not tripped by invisible whitespace.
  printf "%s" "${out}" | tr -d '\r' | xargs
}

eventually_equal() {
  local expected="$1"
  local timeout_seconds="$2"
  local message="$3"
  shift 3

  local deadline=$((SECONDS + timeout_seconds))
  local value=""

  while (( SECONDS < deadline )); do
    if value="$("$@")"; then
      :
    else
      if [[ "${STRICT_DB_CHECK}" == "1" ]]; then
        log_fail "Strict DB check mode: command failed while asserting '${message}'"
        exit 1
      fi
      value=""
    fi

    if [[ "${value}" == "${expected}" ]]; then
      log_pass "${message} (value=${value})"
      return
    fi
    sleep 1
  done

  log_fail "${message} (expected=${expected}, actual=${value})"
  exit 1
}

eventually_command_success() {
  local timeout_seconds="$1"
  local message="$2"
  shift 2

  local deadline=$((SECONDS + timeout_seconds))
  while (( SECONDS < deadline )); do
    if "$@" >/dev/null 2>&1; then
      log_pass "${message}"
      return
    fi
    sleep 1
  done

  log_fail "${message}"
  exit 1
}

extract_last_json_object() {
  local raw="$1"

  python3 - "$raw" <<'PY'
import json
import sys

raw = sys.argv[1]
lines = [line.strip() for line in raw.splitlines() if line.strip()]

for line in reversed(lines):
  try:
    obj = json.loads(line)
    print(json.dumps(obj))
    sys.exit(0)
  except Exception:
    continue

try:
  obj = json.loads(raw)
  print(json.dumps(obj))
  sys.exit(0)
except Exception:
  pass

print("", end="")
sys.exit(1)
PY
}

run_grpc_business_flow() {
  local tmp_go_file
  local json_output

  tmp_go_file="$(mktemp /tmp/lms_e2e_grpc_flow_XXXX.go)"

  cat > "${tmp_go_file}" <<'GO'
package main

import (
  "context"
  "encoding/json"
  "fmt"
  "os"
  "time"

  commonv1 "github.com/DgHnG36/lib-management-system/shared/go/v1"
  bookv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/book"
  orderv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/order"
  userv1 "github.com/DgHnG36/lib-management-system/shared/go/v1/user"
  "google.golang.org/grpc"
  "google.golang.org/grpc/credentials/insecure"
)

type flowResult struct {
  UserID         string   `json:"user_id"`
  Username       string   `json:"username"`
  Email          string   `json:"email"`
  LoginToken     string   `json:"login_token"`
  BookIDs        []string `json:"book_ids"`
  OrderedBookID  string   `json:"ordered_book_id"`
  OrderID        string   `json:"order_id"`
  OrderStatus    string   `json:"order_status"`
  ListOrderFound bool     `json:"list_order_found"`
  ListOrderTotal int32    `json:"list_order_total"`
}

func fatalf(format string, args ...interface{}) {
  fmt.Fprintf(os.Stderr, format+"\n", args...)
  os.Exit(1)
}

func getenv(key, fallback string) string {
  if v := os.Getenv(key); v != "" {
    return v
  }
  return fallback
}

func contains(xs []string, target string) bool {
  for _, x := range xs {
    if x == target {
      return true
    }
  }
  return false
}

func main() {
  userAddr := getenv("GRPC_USER_ADDR", "localhost:40041")
  bookAddr := getenv("GRPC_BOOK_ADDR", "localhost:40042")
  orderAddr := getenv("GRPC_ORDER_ADDR", "localhost:40043")

  dialOpt := grpc.WithTransportCredentials(insecure.NewCredentials())

  userConn, err := grpc.NewClient(userAddr, dialOpt)
  if err != nil {
    fatalf("failed to connect user-service: %v", err)
  }
  defer userConn.Close()

  bookConn, err := grpc.NewClient(bookAddr, dialOpt)
  if err != nil {
    fatalf("failed to connect book-service: %v", err)
  }
  defer bookConn.Close()

  orderConn, err := grpc.NewClient(orderAddr, dialOpt)
  if err != nil {
    fatalf("failed to connect order-service: %v", err)
  }
  defer orderConn.Close()

  userClient := userv1.NewUserServiceClient(userConn)
  bookClient := bookv1.NewBookServiceClient(bookConn)
  orderClient := orderv1.NewOrderServiceClient(orderConn)

  suffix := time.Now().UnixNano()
  short := suffix % 1000000
  username := fmt.Sprintf("e2e_user_%d", suffix)
  email := fmt.Sprintf("e2e_user_%d@example.com", suffix)
  password := "E2e!Passw0rd123"

  ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
  defer cancel()

  regResp, err := userClient.Register(ctx, &userv1.RegisterRequest{
    Username:    username,
    Password:    password,
    Email:       email,
    PhoneNumber: "0900000000",
  })
  if err != nil {
    fatalf("register failed: %v", err)
  }
  if regResp.GetStatus() != 200 || regResp.GetUserId() == "" {
    fatalf("register returned invalid response: status=%d user_id=%q", regResp.GetStatus(), regResp.GetUserId())
  }
  userID := regResp.GetUserId()

  loginResp, err := userClient.Login(ctx, &userv1.LoginRequest{
    Identifier: &userv1.LoginRequest_Username{Username: username},
    Password:   password,
  })
  if err != nil {
    fatalf("login failed: %v", err)
  }
  if loginResp.GetStatus() != 200 || loginResp.GetToken() == "" || loginResp.GetUser() == nil {
    fatalf("login returned invalid payload")
  }
  if loginResp.GetUser().GetId() != userID {
    fatalf("login user_id mismatch: expected=%s actual=%s", userID, loginResp.GetUser().GetId())
  }

  isbn1 := fmt.Sprintf("E2E%06dA", short)
  isbn2 := fmt.Sprintf("E2E%06dB", short)
  createBooksResp, err := bookClient.CreateBooks(ctx, &bookv1.CreateBooksRequest{
    Books: []*bookv1.CreateBookPayload{
      {
        Title:         "E2E Dist Sys A",
        Author:        "Test Runner",
        Isbn:          isbn1,
        Category:      "Test",
        Description:   "E2E test book A",
        TotalQuantity: 5,
      },
      {
        Title:         "E2E Go Micro B",
        Author:        "Test Runner",
        Isbn:          isbn2,
        Category:      "Test",
        Description:   "E2E test book B",
        TotalQuantity: 4,
      },
    },
  })
  if err != nil {
    fatalf("create books failed: %v", err)
  }
  if createBooksResp.GetSuccessCount() < 2 || len(createBooksResp.GetBooks()) < 2 {
    fatalf("create books returned insufficient books")
  }

  bookIDs := make([]string, 0, len(createBooksResp.GetBooks()))
  for _, b := range createBooksResp.GetBooks() {
    if b.GetId() == "" {
      fatalf("created book has empty id")
    }
    bookIDs = append(bookIDs, b.GetId())
  }

  listBooksResp, err := bookClient.ListBooks(ctx, &bookv1.ListBooksRequest{
    Pagination: &commonv1.PaginationRequest{Page: 1, Limit: 100},
  })
  if err != nil {
    fatalf("list books failed: %v", err)
  }

  listedIDs := make([]string, 0, len(listBooksResp.GetBooks()))
  for _, b := range listBooksResp.GetBooks() {
    listedIDs = append(listedIDs, b.GetId())
  }
  for _, id := range bookIDs {
    if !contains(listedIDs, id) {
      fatalf("created book id not found in list books: %s", id)
    }
  }

  orderedBookID := bookIDs[0]
  createOrderResp, err := orderClient.CreateOrder(ctx, &orderv1.CreateOrderRequest{
    UserId:     userID,
    BookIds:    []string{orderedBookID},
    BorrowDays: 7,
  })
  if err != nil {
    fatalf("create order failed: %v", err)
  }

  if createOrderResp.GetOrder() == nil || createOrderResp.GetOrder().GetId() == "" {
    fatalf("create order returned empty order")
  }

  orderID := createOrderResp.GetOrder().GetId()
  orderStatus := createOrderResp.GetOrder().GetStatus().String()

  listOrdersResp, err := orderClient.ListMyOrders(ctx, &orderv1.ListMyOrdersRequest{
    UserId: userID,
    Pagination: &commonv1.PaginationRequest{
      Page:  1,
      Limit: 20,
    },
  })
  if err != nil {
    fatalf("list my orders failed: %v", err)
  }

  found := false
  for _, o := range listOrdersResp.GetOrders() {
    if o.GetId() == orderID {
      found = true
      break
    }
  }
  if !found {
    fatalf("created order not found in list my orders")
  }

  out, err := json.Marshal(flowResult{
    UserID:         userID,
    Username:       username,
    Email:          email,
    LoginToken:     loginResp.GetToken(),
    BookIDs:        bookIDs,
    OrderedBookID:  orderedBookID,
    OrderID:        orderID,
    OrderStatus:    orderStatus,
    ListOrderFound: found,
    ListOrderTotal: listOrdersResp.GetTotalCount(),
  })
  if err != nil {
    fatalf("failed to marshal result json: %v", err)
  }

  fmt.Println(string(out))
}
GO

  if command -v go >/dev/null 2>&1; then
  log_info "Running gRPC business flow with local Go"
  json_output="$(cd "${REPO_ROOT}" && GRPC_USER_ADDR="${GRPC_USER_ADDR}" GRPC_BOOK_ADDR="${GRPC_BOOK_ADDR}" GRPC_ORDER_ADDR="${GRPC_ORDER_ADDR}" go run "${tmp_go_file}")"
  else
  log_info "Local Go not found, running gRPC business flow inside golang container"

  local compose_project
  local compose_network
  compose_project="${COMPOSE_PROJECT_NAME:-$(basename "$(dirname "${COMPOSE_FILE}")")}" 
  compose_network="${compose_project}_lms-network"

  json_output="$(docker run --rm \
    --network "${compose_network}" \
    -e GRPC_USER_ADDR="user-service:40041" \
    -e GRPC_BOOK_ADDR="book-service:40042" \
    -e GRPC_ORDER_ADDR="order-service:40043" \
    -v "${REPO_ROOT}:/app" \
    -v "${tmp_go_file}:/tmp/lms_e2e_grpc_flow.go:ro" \
    -w /app \
    golang:1.25-alpine \
    sh -lc 'go run /tmp/lms_e2e_grpc_flow.go')"
  fi

  rm -f "${tmp_go_file}"

  local clean_json
  clean_json="$(extract_last_json_object "${json_output}")" || {
    log_fail "gRPC business flow did not return valid JSON output"
    printf "%s\n" "${json_output}" >&2
    exit 1
  }

  printf "%s" "${clean_json}" > "${FLOW_RESULT_JSON}"
  assert_file_exists "${FLOW_RESULT_JSON}" "gRPC business flow output saved"
}

run_gateway_proxy_forward_check() {
  local tmp_go_file
  local tmp_go_dir
  local tmp_go_pkg
  local json_output
  local result_file

  if ! command -v go >/dev/null 2>&1; then
  log_info "Skipping proxy forward check: local Go not found"
  return
  fi

  result_file="/tmp/lms_e2e_proxy_forward_result.json"
  tmp_go_dir="$(mktemp -d "${REPO_ROOT}/services/gateway-service/cmd/e2e_proxy_forward_tmp_XXXX")"
  tmp_go_pkg="./services/gateway-service/cmd/$(basename "${tmp_go_dir}")"
  tmp_go_file="${tmp_go_dir}/main.go"

  cat > "${tmp_go_file}" <<'GO'
package main

import (
  "encoding/json"
  "fmt"
  "net/http"
  "net/http/httptest"
  "os"
  "time"

  "github.com/DgHnG36/lib-management-system/services/gateway-service/internal/middleware"
  "github.com/DgHnG36/lib-management-system/services/gateway-service/internal/proxy"
  gatewayRouter "github.com/DgHnG36/lib-management-system/services/gateway-service/internal/router"
  "github.com/DgHnG36/lib-management-system/services/gateway-service/pkg/logger"
  "github.com/redis/go-redis/v9"
)

type result struct {
  StatusCode        int    `json:"status_code"`
  ForwardedPath     string `json:"forwarded_path"`
  ForwardedAuth     string `json:"forwarded_auth"`
  ForwardedUserID   string `json:"forwarded_user_id"`
  ForwardedUserRole string `json:"forwarded_user_role"`
  ForwardedRequestID string `json:"forwarded_request_id"`
}

type closeNotifyRecorder struct {
  *httptest.ResponseRecorder
  closeCh chan bool
}

func newCloseNotifyRecorder() *closeNotifyRecorder {
  return &closeNotifyRecorder{
    ResponseRecorder: httptest.NewRecorder(),
    closeCh:          make(chan bool, 1),
  }
}

func (r *closeNotifyRecorder) CloseNotify() <-chan bool {
  return r.closeCh
}

func getenv(key, fallback string) string {
  if v := os.Getenv(key); v != "" {
    return v
  }
  return fallback
}

func fatalf(format string, args ...interface{}) {
  fmt.Fprintf(os.Stderr, format+"\n", args...)
  os.Exit(1)
}

func main() {
  redisAddr := getenv("REDIS_ADDR", "localhost:63379")
  jwtSecret := []byte(getenv("JWT_SECRET", "local-dev-secret-key"))

  var forwardedPath string
  var forwardedAuth string
  var forwardedUserID string
  var forwardedUserRole string
  var forwardedRequestID string

  backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    forwardedPath = r.URL.Path
    forwardedAuth = r.Header.Get("Authorization")
    forwardedUserID = r.Header.Get("X-User-ID")
    forwardedUserRole = r.Header.Get("X-User-Role")
    forwardedRequestID = r.Header.Get("X-Request-ID")
    w.WriteHeader(http.StatusOK)
    _, _ = w.Write([]byte(`{"ok":true}`))
  }))
  defer backend.Close()

  redisClient := redis.NewClient(&redis.Options{Addr: redisAddr})
  defer redisClient.Close()

  log := logger.DefaultNewLogger()
  authMiddleware := middleware.NewAuthMiddleware(jwtSecret, "HS256", log)
  corsMiddleware := middleware.NewCORSMiddleware(
    []string{"http://localhost:3000"},
    []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
    []string{"Origin", "Authorization", "Content-Type"},
    []string{"X-Request-ID", "X-Rate-Limit", "X-Rate-Limit-Remaining"},
    true,
    12*time.Hour,
  )
  rateLimitMiddleware := middleware.NewRateLimitMiddleware(redisClient, 1000, 60, log)
  reverseProxy := proxy.NewReverseProxy(map[string]string{
    "/api/v1/books": backend.URL,
  }, log)

  router := gatewayRouter.SetupRouter(authMiddleware, corsMiddleware, rateLimitMiddleware, reverseProxy, log)

  token, err := middleware.GenerateToken("e2e-proxy-user", "user", "e2e-proxy@example.com", jwtSecret, "HS256", 15)
  if err != nil {
    fatalf("failed to generate token: %v", err)
  }

  req := httptest.NewRequest(http.MethodGet, "/api/v1/books?limit=1", nil)
  req.Header.Set("Authorization", "Bearer "+token)
  req.Header.Set("Origin", "http://localhost:3000")
  req.Header.Set("X-Request-ID", "e2e-req-001")
  rec := newCloseNotifyRecorder()

  router.ServeHTTP(rec, req)

  out, err := json.Marshal(result{
    StatusCode:        rec.Code,
    ForwardedPath:     forwardedPath,
    ForwardedAuth:     forwardedAuth,
    ForwardedUserID:   forwardedUserID,
    ForwardedUserRole: forwardedUserRole,
    ForwardedRequestID: forwardedRequestID,
  })
  if err != nil {
    fatalf("marshal result failed: %v", err)
  }

  fmt.Println(string(out))
}
GO

  json_output="$(cd "${REPO_ROOT}" && REDIS_ADDR="localhost:63379" JWT_SECRET="${JWT_SECRET}" go run "${tmp_go_pkg}")"
  rm -rf "${tmp_go_dir}"

  local clean_json
  clean_json="$(extract_last_json_object "${json_output}")" || {
    log_fail "proxy forward check did not return valid JSON output"
    printf "%s\n" "${json_output}" >&2
    exit 1
  }

  printf "%s" "${clean_json}" > "${result_file}"

  assert_equal "200" "$(python3 -c 'import json;print(json.load(open("/tmp/lms_e2e_proxy_forward_result.json"))["status_code"])')" "proxy forward check status is 200"
  assert_equal "/books" "$(python3 -c 'import json;print(json.load(open("/tmp/lms_e2e_proxy_forward_result.json"))["forwarded_path"])')" "proxy forwards trimmed path"
  assert_contains "$(python3 -c 'import json;print(json.load(open("/tmp/lms_e2e_proxy_forward_result.json"))["forwarded_auth"])')" "Bearer " "proxy forwards authorization header"
  assert_equal "e2e-proxy-user" "$(python3 -c 'import json;print(json.load(open("/tmp/lms_e2e_proxy_forward_result.json"))["forwarded_user_id"])')" "proxy forwards X-User-ID"
  assert_equal "user" "$(python3 -c 'import json;print(json.load(open("/tmp/lms_e2e_proxy_forward_result.json"))["forwarded_user_role"])')" "proxy forwards X-User-Role"
  assert_equal "e2e-req-001" "$(python3 -c 'import json;print(json.load(open("/tmp/lms_e2e_proxy_forward_result.json"))["forwarded_request_id"])')" "proxy forwards X-Request-ID"
}

extract_json_field() {
  local field="$1"

  python3 - "${FLOW_RESULT_JSON}" "${field}" <<'PY'
import json
import sys

path = sys.argv[1]
field = sys.argv[2]

with open(path, "r", encoding="utf-8") as f:
    data = json.load(f)

value = data
for part in field.split("."):
    if part.isdigit():
        value = value[int(part)]
    else:
        value = value[part]

if isinstance(value, bool):
    print("true" if value else "false")
elif isinstance(value, (list, dict)):
    print(json.dumps(value))
else:
    print(value)
PY
}

validate_cross_db() {
  local user_id
  local username
  local email
  local order_id
  local ordered_book_id
  local book_ids_json

  user_id="$(extract_json_field user_id)"
  username="$(extract_json_field username)"
  email="$(extract_json_field email)"
  order_id="$(extract_json_field order_id)"
  ordered_book_id="$(extract_json_field ordered_book_id)"
  book_ids_json="$(extract_json_field book_ids)"

  assert_contains "$(extract_json_field login_token)" "." "login returned JWT token"
  assert_equal "true" "$(extract_json_field list_order_found)" "list my orders contains created order"

  eventually_equal "1" 20 "user exists in user_db.users" \
    query_postgres postgres-user user_db "SELECT COUNT(1) FROM users WHERE id='${user_id}' OR username='${username}';"

  eventually_equal "1" 20 "user email persisted in user_db.users" \
    query_postgres postgres-user user_db "SELECT COUNT(1) FROM users WHERE email='${email}';"

  local book_ids
  mapfile -t book_ids < <(python3 - "${book_ids_json}" <<'PY'
import json
import sys
for x in json.loads(sys.argv[1]):
    print(x)
PY
)

  local book_id
  for book_id in "${book_ids[@]}"; do
    eventually_equal "1" 20 "book ${book_id} exists in book_db.books" \
      query_postgres postgres-book book_db "SELECT COUNT(1) FROM books WHERE id='${book_id}';"
  done

  eventually_equal "1" 20 "order exists in order_db.orders" \
    query_postgres postgres-order order_db "SELECT COUNT(1) FROM orders WHERE id='${order_id}';"

  eventually_equal "${user_id}" 20 "order belongs to created user id" \
    query_postgres postgres-order order_db "SELECT user_id FROM orders WHERE id='${order_id}' LIMIT 1;"

  eventually_equal "1" 20 "order_books links order and ordered book" \
    query_postgres postgres-order order_db "SELECT COUNT(1) FROM order_books WHERE order_id='${order_id}' AND book_id='${ordered_book_id}';"

  eventually_equal "1" 20 "ordered book id maps to book_db.books" \
    query_postgres postgres-book book_db "SELECT COUNT(1) FROM books WHERE id='${ordered_book_id}';"
}

validate_notification_flow() {
  local logs
  local start_ts

  start_ts="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  sleep 1

  # Trigger another order.created event specifically for notification verification.
  run_grpc_business_flow

  eventually_command_success 25 "notification-service consumed order.created event" \
    sh -lc "docker logs --since '${start_ts}' lms-notification-service 2>&1 | grep -F 'Received event: order.created'"

  logs="$(docker logs --since "${start_ts}" lms-notification-service 2>&1 || true)"

  if [[ "${NOTIFICATION_REQUIRE_SUCCESS}" == "1" ]]; then
    assert_contains "${logs}" "Email sent to" "notification-service sent email successfully"
    return
  fi

  if [[ "${logs}" == *"Email sent to"* ]]; then
    log_pass "notification-service sent email successfully"
  elif [[ "${logs}" == *"Failed to send email"* || "${logs}" == *"Failed to process message"* ]]; then
    log_info "notification-service consumed event but email sending failed in local env"
  else
    log_fail "notification-service did not attempt email send"
    exit 1
  fi
}

cleanup() {
  if [[ "${KEEP_STACK}" == "1" ]]; then
    log_info "KEEP_STACK=1, skipping docker compose down"
    return
  fi

  log_info "Stopping local stack"
  "${COMPOSE_CMD[@]}" -f "${COMPOSE_FILE}" down >/dev/null
}

wait_for_gateway() {
  log_info "Waiting for gateway to become healthy at ${API_GATEWAY_URL}/healthy"
  local deadline=$((SECONDS + WAIT_TIMEOUT_SECONDS))

  while (( SECONDS < deadline )); do
    local code
    code="$(curl -sS -o /tmp/lms_e2e_health.txt -w "%{http_code}" "${API_GATEWAY_URL}/healthy" || true)"
    if [[ "${code}" == "200" ]]; then
      log_pass "Gateway healthy"
      return
    fi
    sleep 2
  done

  log_fail "Gateway did not become healthy within ${WAIT_TIMEOUT_SECONDS}s"
  exit 1
}

main() {
  trap cleanup EXIT

  echo "------------------------------------"
  echo "RUNNING FULL WORKFLOW TEST"
  echo "------------------------------------"

  if [[ ! -f "${COMPOSE_FILE}" ]]; then
    log_fail "Compose file not found: ${COMPOSE_FILE}"
    exit 1
  fi

  if [[ "${SKIP_START}" != "1" ]]; then
    log_info "Starting local stack from ${COMPOSE_FILE}"
    "${COMPOSE_CMD[@]}" -f "${COMPOSE_FILE}" up -d
  else
    log_info "SKIP_START=1, assuming services are already running"
  fi

  wait_for_gateway

  local status
  status="$(http_status GET "${API_GATEWAY_URL}/healthy")"
  assert_equal "200" "${status}" "GET /healthy returns 200"
  assert_contains "$(cat /tmp/lms_e2e_body.txt)" "healthy" "health response includes 'healthy'"

  status="$(http_status GET "${API_GATEWAY_URL}/ready")"
  assert_equal "200" "${status}" "GET /ready returns 200"

  status="$(http_status GET "${API_GATEWAY_URL}/api/v1/books")"
  assert_equal "401" "${status}" "GET /api/v1/books without token returns 401"
  assert_contains "$(cat /tmp/lms_e2e_body.txt)" "Missing authorization header" "unauthorized response message is correct"

  local token
  token="$(JWT_SECRET="${JWT_SECRET}" generate_jwt)"
  log_info "Generated JWT for authenticated gateway requests"

  status="$(http_status GET "${API_GATEWAY_URL}/api/v1/books" "Bearer ${token}")"
  assert_equal "${EXPECTED_PROXY_STATUS}" "${status}" "GET /api/v1/books with token returns expected upstream status"

  if [[ "${EXPECTED_PROXY_STATUS}" == "503" ]]; then
    assert_contains "$(cat /tmp/lms_e2e_body.txt)" "service unavailable" "proxy error message is returned"
  fi

  log_info "Validating rate-limit middleware after successful auth"
  local last_code=""
  for _ in $(seq 1 5); do
    last_code="$(http_status GET "${API_GATEWAY_URL}/api/v1/books" "Bearer ${token}")"
  done

  if [[ "${last_code}" == "429" ]]; then
    log_pass "Rate-limit is active (received 429)"
  else
    log_info "Rate-limit not reached in 5 quick requests (last status=${last_code})"
  fi

  log_info "Running business flow: register, login, create books, create order, list books, list orders"
  run_gateway_proxy_forward_check
  run_grpc_business_flow
  validate_notification_flow
  validate_cross_db

  log_pass "Full workflow test completed"
}

main "$@"

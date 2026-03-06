#!/usr/bin/env bash
# check-test-resources.sh
#
# Checks for leftover XO resources created by the xo-sdk-go integration test suite.
# Queries the Xen Orchestra REST API for every resource type that the tests create,
# filtering by the test name prefix.
#
# Usage:
#   ./scripts/check-test-resources.sh [--prefix PREFIX] [--url URL] [--token TOKEN]
#
# Environment variables (can be overridden by flags):
#   XOA_URL          XO base URL (e.g. http://10.1.0.222 or ws://10.1.0.222)
#   XOA_TOKEN        Authentication token
#   XOA_TEST_PREFIX  Resource name prefix used during tests (default: xo-go-sdk-)
#
# Exit codes:
#   0  No leftover resources found
#   1  Leftover resources found (or a query failed)

set -euo pipefail

# ---------------------------------------------------------------------------
# Defaults
# ---------------------------------------------------------------------------
PREFIX="${XOA_TEST_PREFIX:-xo-go-sdk-}"
RAW_URL="${XOA_URL:-}"
TOKEN="${XOA_TOKEN:-}"

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------
while [[ $# -gt 0 ]]; do
  case "$1" in
    --prefix)  PREFIX="$2";  shift 2 ;;
    --url)     RAW_URL="$2"; shift 2 ;;
    --token)   TOKEN="$2";   shift 2 ;;
    -h|--help)
      grep '^#' "$0" | grep -v '#!/' | sed 's/^# \{0,1\}//'
      exit 0
      ;;
    *) echo "Unknown option: $1" >&2; exit 1 ;;
  esac
done

# ---------------------------------------------------------------------------
# Validate inputs
# ---------------------------------------------------------------------------
if [[ -z "$RAW_URL" ]]; then
  echo "ERROR: XOA_URL is not set. Use --url or export XOA_URL." >&2
  exit 1
fi
if [[ -z "$TOKEN" ]]; then
  echo "ERROR: XOA_TOKEN is not set. Use --token or export XOA_TOKEN." >&2
  exit 1
fi

# Normalise URL: replace ws:// / wss:// with http:// / https://
BASE_URL="${RAW_URL/ws:\/\//http://}"
BASE_URL="${BASE_URL/wss:\/\//https://}"
BASE_URL="${BASE_URL%/}"   # strip trailing slash

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m'

FOUND=0   # set to 1 if any leftover resources are detected
ERRORS=0  # set to 1 if any API query fails

# query_collection <collection> <label>
#   Fetches /rest/v0/<collection>?fields=name_label and prints any item whose
#   name_label starts with PREFIX.
query_collection() {
  local collection="$1"
  local label="$2"

  local url="${BASE_URL}/rest/v0/${collection}?fields=name_label,id"
  local response

  if ! response=$(curl -sf \
      -b "authenticationToken=${TOKEN}" \
      -H "Accept: application/json" \
      "$url" 2>&1); then
    echo -e "${YELLOW}  WARN: failed to query ${collection} (${response})${NC}" >&2
    ERRORS=1
    return
  fi

  local matches
  matches=$(echo "$response" | jq -r --arg prefix "$PREFIX" '
    if type == "array" then
      .[] | select(.name_label | startswith($prefix))
      | "  - \(.name_label)  (id: \(.id // .href // ""))"
    else empty end
  ' 2>/dev/null) || true

  if [[ -n "$matches" ]]; then
    echo -e "${RED}  [LEAK] ${label}:${NC}"
    echo "$matches"
    FOUND=1
  else
    echo -e "${GREEN}  [OK]   ${label}${NC}"
  fi
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
echo -e "${BLUE}=====================================================${NC}"
echo -e "${BLUE}  XO SDK Integration Test — Resource Leak Checker${NC}"
echo -e "${BLUE}=====================================================${NC}"
echo ""
echo -e "  Target  : ${BASE_URL}"
echo -e "  Prefix  : ${PREFIX}"
echo ""
echo -e "${BLUE}Checking resource collections…${NC}"
echo ""

# Every resource type the integration tests can create
query_collection "vms"           "Virtual Machines (VMs)"
query_collection "vm-snapshots"  "VM Snapshots"
query_collection "vdis"          "Virtual Disk Images (VDIs)"
query_collection "vdi-snapshots" "VDI Snapshots"
query_collection "networks"      "Networks"
query_collection "vbds"          "Virtual Block Devices (VBDs)"
query_collection "srs"           "Storage Repositories (SRs)"
query_collection "hosts"         "Hosts"
query_collection "pools"         "Pools"

echo ""
echo -e "${BLUE}=====================================================${NC}"

if [[ "$FOUND" -eq 1 ]]; then
  echo -e "${RED}  RESULT: Leftover test resources detected!${NC}"
  echo -e "${BLUE}=====================================================${NC}"
  echo ""
  read -r -p "  Run cleanup now? [y/N] " confirm
  case "$confirm" in
    [yY][eE][sS]|[yY])
      echo ""
      SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
      exec "${SCRIPT_DIR}/cleanup-test-resources.sh"
      ;;
    *)
      echo ""
      SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
      echo -e "  To delete them later, run:"
      echo -e "  ${YELLOW}  ${SCRIPT_DIR}/cleanup-test-resources.sh${NC}"
      echo -e "  or:"
      echo -e "  ${YELLOW}  make cleanup-test-resources${NC}"
      echo -e "${BLUE}=====================================================${NC}"
      exit 1
      ;;
  esac
elif [[ "$ERRORS" -eq 1 ]]; then
  echo -e "${YELLOW}  RESULT: Check completed with warnings (some queries failed).${NC}"
  echo -e "${BLUE}=====================================================${NC}"
  exit 1
else
  echo -e "${GREEN}  RESULT: No leftover test resources found. Infrastructure is clean.${NC}"
  echo -e "${BLUE}=====================================================${NC}"
  exit 0
fi

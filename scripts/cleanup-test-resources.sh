#!/usr/bin/env bash
# cleanup-test-resources.sh
#
# Deletes leftover XO resources created by the xo-sdk-go integration test suite.
# Queries the Xen Orchestra REST API for every deletable resource type and removes
# any item whose name starts with the test prefix.
#
# Deletable resource types (in dependency order):
#   VMs       DELETE /rest/v0/vms/:id        (also deletes attached VBDs)
#   VDIs      DELETE /rest/v0/vdis/:id
#   Networks  DELETE /rest/v0/networks/:id
#
# Usage:
#   ./scripts/cleanup-test-resources.sh [--prefix PREFIX] [--url URL] [--token TOKEN] [--yes]
#
# Environment variables (can be overridden by flags):
#   XOA_URL          XO base URL (e.g. http://10.1.0.222 or ws://10.1.0.222)
#   XOA_TOKEN        Authentication token
#   XOA_TEST_PREFIX  Resource name prefix used during tests (default: xo-go-sdk-)
#
# Flags:
#   --yes   Skip confirmation prompt and delete immediately
#
# Exit codes:
#   0  All resources deleted successfully (or nothing to delete)
#   1  One or more deletions failed

set -euo pipefail

# ---------------------------------------------------------------------------
# Defaults
# ---------------------------------------------------------------------------
PREFIX="${XOA_TEST_PREFIX:-xo-go-sdk-}"
RAW_URL="${XOA_URL:-}"
TOKEN="${XOA_TOKEN:-}"
YES=0

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------
while [[ $# -gt 0 ]]; do
  case "$1" in
    --prefix)  PREFIX="$2";  shift 2 ;;
    --url)     RAW_URL="$2"; shift 2 ;;
    --token)   TOKEN="$2";   shift 2 ;;
    --yes|-y)  YES=1;        shift   ;;
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
# Colours
# ---------------------------------------------------------------------------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

DELETED=0
FAILED=0

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

# collect_ids <collection>
#   Prints "id|name_label" lines for every item in <collection> whose
#   name_label starts with PREFIX.
collect_ids() {
  local collection="$1"
  local url="${BASE_URL}/rest/v0/${collection}?fields=name_label,id"
  local response

  if ! response=$(curl -sf \
      -b "authenticationToken=${TOKEN}" \
      -H "Accept: application/json" \
      "$url" 2>&1); then
    echo -e "${YELLOW}  WARN: failed to query ${collection}: ${response}${NC}" >&2
    return
  fi

  echo "$response" | jq -r --arg prefix "$PREFIX" '
    if type == "array" then
      .[] | select(.name_label | startswith($prefix))
      | "\(.id)|\(.name_label)"
    else empty end
  ' 2>/dev/null || true
}

# delete_resource <collection> <id> <name>
#   Sends DELETE /rest/v0/<collection>/<id> and reports success/failure.
delete_resource() {
  local collection="$1"
  local id="$2"
  local name="$3"

  local http_code
  http_code=$(curl -s -o /dev/null -w "%{http_code}" \
      -X DELETE \
      -b "authenticationToken=${TOKEN}" \
      "${BASE_URL}/rest/v0/${collection}/${id}")

  if [[ "$http_code" == "204" || "$http_code" == "200" || "$http_code" == "202" ]]; then
    echo -e "${GREEN}  [DELETED]${NC} ${name} (${id})"
    (( DELETED++ )) || true
  elif [[ "$http_code" == "404" ]]; then
    echo -e "${YELLOW}  [GONE]   ${NC} ${name} (${id}) — already deleted"
  else
    echo -e "${RED}  [FAILED] ${NC} ${name} (${id}) — HTTP ${http_code}"
    (( FAILED++ )) || true
  fi
}

# delete_collection <collection> <label>
#   Collects and deletes all matching resources in a collection.
#   Populates the global TO_DELETE associative array for the preview.
delete_collection() {
  local collection="$1"
  local label="$2"
  local items

  items=$(collect_ids "$collection")

  if [[ -z "$items" ]]; then
    echo -e "${GREEN}  [OK]     ${label} — nothing to delete${NC}"
    return
  fi

  echo -e "${CYAN}  ${label}:${NC}"
  while IFS='|' read -r id name; do
    [[ -z "$id" ]] && continue
    delete_resource "$collection" "$id" "$name"
  done <<< "$items"
}

# preview_collection <collection> <label>
#   Lists matching resources without deleting them.
preview_collection() {
  local collection="$1"
  local label="$2"
  local items

  items=$(collect_ids "$collection")

  if [[ -z "$items" ]]; then
    return
  fi

  echo -e "${RED}  ${label}:${NC}"
  while IFS='|' read -r id name; do
    [[ -z "$id" ]] && continue
    echo -e "    - ${name}  ${YELLOW}(${id})${NC}"
    (( PREVIEW_COUNT++ )) || true
  done <<< "$items"
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
echo -e "${BLUE}=====================================================${NC}"
echo -e "${BLUE}  XO SDK Integration Test — Resource Cleanup${NC}"
echo -e "${BLUE}=====================================================${NC}"
echo ""
echo -e "  Target  : ${BASE_URL}"
echo -e "  Prefix  : ${PREFIX}"
echo ""

# --- Preview phase: show what will be deleted ---
echo -e "${BOLD}Resources to delete:${NC}"
echo ""

PREVIEW_COUNT=0
# VBDs are implicitly deleted with their VMs — no standalone delete needed
# Deletions must happen in dependency order:
#   1. VMs first (owns VBDs and VM snapshots)
#   2. VM snapshots (may outlive their parent VM if it was already deleted)
#   3. VDIs and VDI snapshots
#   4. Networks last
preview_collection "vms"           "Virtual Machines (VMs)"
preview_collection "vm-snapshots"  "VM Snapshots"
preview_collection "vdis"          "Virtual Disk Images (VDIs)"
preview_collection "vdi-snapshots" "VDI Snapshots"
preview_collection "networks"      "Networks"

echo ""

if [[ "$PREVIEW_COUNT" -eq 0 ]]; then
  echo -e "${GREEN}  Nothing to clean up. Infrastructure is already clean.${NC}"
  echo -e "${BLUE}=====================================================${NC}"
  exit 0
fi

echo -e "  ${BOLD}Total: ${PREVIEW_COUNT} resource(s) will be permanently deleted.${NC}"
echo ""

# --- Confirmation ---
if [[ "$YES" -eq 0 ]]; then
  echo -e "${YELLOW}  WARNING: This action is irreversible.${NC}"
  read -r -p "  Proceed with deletion? [y/N] " confirm
  case "$confirm" in
    [yY][eE][sS]|[yY]) ;;
    *)
      echo -e "${YELLOW}  Aborted. No resources were deleted.${NC}"
      echo -e "${BLUE}=====================================================${NC}"
      exit 0
      ;;
  esac
  echo ""
fi

# --- Deletion phase ---
echo -e "${BOLD}Deleting resources…${NC}"
echo ""

# VMs first (their VBDs are implicitly removed)
delete_collection "vms"           "Virtual Machines (VMs)"
# VM snapshots (may outlive their parent VM)
delete_collection "vm-snapshots"  "VM Snapshots"
# VDIs and VDI snapshots
delete_collection "vdis"          "Virtual Disk Images (VDIs)"
delete_collection "vdi-snapshots" "VDI Snapshots"
# Networks last
delete_collection "networks"      "Networks"

echo ""
echo -e "${BLUE}=====================================================${NC}"

if [[ "$FAILED" -gt 0 ]]; then
  echo -e "${RED}  RESULT: ${DELETED} deleted, ${FAILED} failed.${NC}"
  echo -e "${RED}  Review the errors above and retry or delete manually.${NC}"
  echo -e "${BLUE}=====================================================${NC}"
  exit 1
else
  echo -e "${GREEN}  RESULT: ${DELETED} resource(s) deleted successfully.${NC}"
  echo -e "${BLUE}=====================================================${NC}"
  exit 0
fi

#!/usr/bin/env sh
# End-to-end smoke test: drives the built stdio server with real MCP messages
# against the mock adapter. No network or credentials required.
set -eu
root=$(cd "$(dirname "$0")/.." && pwd)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
export TRIMBLE_MCP_ENABLE_MOCK=true TRIMBLE_CONNECT_ENABLED=false TRIMBLE_MCP_AUDIT_LOG="$tmp/audit.jsonl"
out=$(printf '%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"smoke","version":"1"}}}' \
  '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' \
  '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"trimble_list_projects","arguments":{"product":"mock","page_size":3}}}' \
  '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"trimble_get_file_metadata","arguments":{"product":"mock","project_id":"mock-prj-001","file_id":"mock-file-002-1"}}}' \
  | "$root/bin/trimble-mcp" -transport stdio 2>"$tmp/stderr")
echo "$out" | grep -q '"protocolVersion":"2025-11-25"' || { echo "FAIL initialize"; exit 1; }
echo "$out" | grep -q '"name":"trimble_list_folder_items"' || { echo "FAIL tools/list"; exit 1; }
echo "$out" | grep -q '"next_page_token"' || { echo "FAIL pagination"; exit 1; }
echo "$out" | grep -q '"code":"resource_not_found"' || { echo "FAIL IDOR check"; exit 1; }
"$root/bin/trimblectl" audit verify --file "$tmp/audit.jsonl"
"$root/bin/trimblectl" capabilities >/dev/null
"$root/bin/trimblectl" projects list --product mock --page-size 2 | grep -q '"complete": false' || { echo "FAIL cli"; exit 1; }
if "$root/bin/trimblectl" projects list --product trimble-connect >/dev/null 2>&1; then echo "FAIL unconfigured product accepted"; exit 1; fi
echo "smoke: OK"

#!/usr/bin/env sh
# Downloads every API definition in Trimble's official SwaggerHub organisation
# (Trimble-Connect) plus the /regions document, into $1 (default
# /tmp/trimble-specs). The raw definitions are not committed; `make catalog`
# regenerates internal/catalog/catalog.json.gz and docs/trimble-products/endpoints
# from them, and fails on any definition that has not been classified.
set -eu
out=${1:-/tmp/trimble-specs}
mkdir -p "$out"
curl -fsS --max-time 60 "https://api.swaggerhub.com/apis/Trimble-Connect?limit=100" -o "$out/index.json"
curl -fsS --max-time 60 "https://app.connect.trimble.com/tc/api/2.0/regions" -o "$out/regions.json"
python3 - "$out" <<'PY'
import json, sys, subprocess
out = sys.argv[1]
idx = json.load(open(f"{out}/index.json"))
for a in idx["apis"]:
    url = next(p["url"] for p in a["properties"] if p["type"] == "Swagger")
    slug = url.split("/Trimble-Connect/")[1].replace("/", "@", 1)
    subprocess.run(["curl", "-fsS", "--max-time", "60", "-o", f"{out}/{slug}.json", url], check=True)
    print(slug)
PY

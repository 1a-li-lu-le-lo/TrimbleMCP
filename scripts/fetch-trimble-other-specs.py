#!/usr/bin/env python3
"""Download every Trimble API definition outside the Trimble-Connect SwaggerHub
organisation, as classified in cmd/trimble-catalog/sources-other.json.

Discovery:
  * every developer.trimble.com product section's sitemap is crawled for
    pages that embed an OpenAPI viewer (spec-url="..."), so a definition
    Trimble links from its developer portal cannot be missed;
  * the manifest's direct sources are downloaded as listed;
  * Vista's per-operation OpenAPI fragments are collected from its llms.txt
    index and merged into one definition per module;
  * Trimble Identity's OpenID Connect discovery document is saved so every
    Identity endpoint is accounted for.

Output (in OUT, default /tmp/trimble-specs/other):
  <source-id>.json      normalised JSON definition (YAML converted)
  discovered.json       [{id, spec_url, doc_urls, kind}] for the generator

Requires Python 3 and PyYAML (for YAML definitions).
"""
import concurrent.futures as cf
import json
import os
import re
import sys
import urllib.parse
import urllib.request

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
MANIFEST = os.path.join(ROOT, "cmd", "trimble-catalog", "sources-other.json")
UA = "trimble-mcp-bridge-catalog/1.0"


def get(url, timeout=60):
    req = urllib.request.Request(url, headers={"User-Agent": UA, "Accept": "*/*"})
    with urllib.request.urlopen(req, timeout=timeout) as r:
        return r.read()


def to_json(raw, url):
    text = raw.decode("utf-8-sig")
    try:
        return json.loads(text)
    except ValueError:
        import yaml  # PyYAML

        return yaml.safe_load(text)


def slug(s):
    return re.sub(r"[^a-z0-9]+", "-", s.lower()).strip("-")


def discover_portal(base, sections):
    found = {}  # spec_url -> set(doc_url)
    pages = []
    for s in sections:
        try:
            idx = get(f"{base}/docs/{s}/sitemap-index.xml").decode()
        except Exception as e:  # section without a sitemap
            print(f"warn: {s}: no sitemap ({e})", file=sys.stderr)
            continue
        for sm in re.findall(r"<loc>([^<]+)</loc>", idx):
            try:
                body = get(sm).decode()
            except Exception as e:
                print(f"warn: {sm}: {e}", file=sys.stderr)
                continue
            pages += [u for u in re.findall(r"<loc>([^<]+)</loc>", body) if "/reference/" in u]

    def spec_urls(page):
        try:
            html = get(page).decode("utf-8", "replace")
        except Exception as e:
            print(f"warn: {page}: {e}", file=sys.stderr)
            return page, []
        return page, [urllib.parse.urljoin(page, u.replace("&amp;", "&")) for u in re.findall(r'spec-url="([^"]+)"', html)]

    with cf.ThreadPoolExecutor(16) as ex:
        for page, urls in ex.map(spec_urls, sorted(set(pages))):
            for u in urls:
                found.setdefault(u, set()).add(urllib.parse.urlparse(page).path)
    return found


def portal_id(doc_path):
    # /docs/<section>/reference/openapi/<rest>/ -> <section>/<rest>
    m = re.match(r"^/docs/([^/]+)/reference/openapi/(.*?)/?$", doc_path)
    if not m:
        return slug(doc_path)
    section, rest = m.group(1), m.group(2)
    rest = rest.replace("controllers/", "")
    return f"{section}/{slug(rest) or 'v1'}"


def vista(index_url):
    idx = get(index_url).decode()
    module, ops = None, []
    for line in idx.splitlines():
        h = re.match(r"^## API Reference: (Vista .+ Direct API)\s*$", line)
        if h:
            module = h.group(1)
            continue
        if line.startswith("## "):
            module = None
            continue
        m = re.match(r"^- \[[^\]]*\]\((https://direct-api\.xchange\.trimble\.com/reference/[^)]+\.md)\)", line)
        if module and m:
            ops.append((module, m.group(1)))

    def fragment(item):
        module, url = item
        try:
            md = get(url).decode("utf-8", "replace")
        except Exception as e:
            return module, url, None, str(e)
        m = re.search(r"```json\s*\n(\{.*?\})\s*\n```", md, re.S)
        if not m:
            return module, url, None, "no OpenAPI fragment"
        return module, url, json.loads(m.group(1)), None

    merged = {}
    errors = []
    with cf.ThreadPoolExecutor(16) as ex:
        for module, url, frag, err in ex.map(fragment, ops):
            if err:
                errors.append((url, err))
                continue
            spec = merged.setdefault(module, {"openapi": frag.get("openapi", "3.0.1"), "info": frag.get("info", {}),
                                              "servers": frag.get("servers", []), "paths": {}, "components": {},
                                              "x-doc-pages": []})
            spec["x-doc-pages"].append(url)
            for p, item in frag.get("paths", {}).items():
                spec["paths"].setdefault(p, {}).update(item)
            for section, entries in frag.get("components", {}).items():
                spec["components"].setdefault(section, {}).update(entries)
    return merged, len(ops), errors


def main():
    out = sys.argv[1] if len(sys.argv) > 1 else "/tmp/trimble-specs/other"
    os.makedirs(out, exist_ok=True)
    man = json.load(open(MANIFEST))
    discovered = []

    for spec_url, docs in sorted(discover_portal(man["portal"]["base"], man["portal"]["sections"]).items()):
        docs = sorted(docs)
        rec = {"id": portal_id(docs[0]), "spec_url": spec_url, "doc_urls": [man["portal"]["base"] + d for d in docs], "kind": "portal"}
        if spec_url.startswith("https://api.swaggerhub.com/apis/Trimble-Connect/"):
            discovered.append(rec)  # catalogued from the SwaggerHub organisation
            continue
        try:
            spec = to_json(get(spec_url), spec_url)
        except Exception as e:
            rec["error"] = str(e)
            discovered.append(rec)
            continue
        json.dump(spec, open(os.path.join(out, rec["id"].replace("/", "__") + ".json"), "w"))
        discovered.append(rec)

    for d in man["direct"]:
        rec = {"id": d["id"], "spec_url": d["url"], "doc_urls": [d["doc_url"]] if d["doc_url"] else [], "kind": "direct"}
        try:
            spec = to_json(get(d["url"]), d["url"])
            json.dump(spec, open(os.path.join(out, d["id"] + ".json"), "w"))
        except Exception as e:
            rec["error"] = str(e)
        discovered.append(rec)

    idn = man["identity"]
    rec = {"id": idn["id"], "spec_url": idn["url"], "doc_urls": [idn["doc_url"]], "kind": "oidc"}
    try:
        json.dump(to_json(get(idn["url"]), idn["url"]), open(os.path.join(out, idn["id"] + ".json"), "w"))
    except Exception as e:
        rec["error"] = str(e)
    discovered.append(rec)

    merged, pages, errors = vista(man["vista"]["index"])
    for module, spec in sorted(merged.items()):
        sid = "vista/" + slug(module.replace("Vista ", "").replace(" v2 Direct API", ""))
        json.dump(spec, open(os.path.join(out, sid.replace("/", "__") + ".json"), "w"))
        discovered.append({"id": sid, "spec_url": man["vista"]["index"], "doc_urls": spec["x-doc-pages"], "kind": "vista"})
    for url, err in errors:
        print(f"warn: vista {url}: {err}", file=sys.stderr)
        discovered.append({"id": "vista-page", "spec_url": url, "doc_urls": [url], "kind": "vista", "error": err})
    print(f"vista: {pages} operation pages, {len(merged)} modules, {len(errors)} errors", file=sys.stderr)

    json.dump(discovered, open(os.path.join(out, "discovered.json"), "w"), indent=1)
    print(f"{len(discovered)} definitions recorded in {out}/discovered.json", file=sys.stderr)


if __name__ == "__main__":
    main()

#!/usr/bin/env python3
"""Download every Trimble API definition outside the Trimble-Connect SwaggerHub
organisation, as classified in cmd/trimble-catalog/sources-other.json.

Discovery:
  * every page of every developer.trimble.com product section (from each
    section's sitemap) and every page of developer.trimblemaps.com (a
    same-site crawl) is scanned for embedded definitions: OpenAPI viewers
    (spec-url, in any quoting style) and links to OpenAPI, Swagger or
    AsyncAPI definition files. A definition Trimble links from these sites
    therefore cannot be missed;
  * the manifest's direct sources are downloaded as listed;
  * Vista's per-operation OpenAPI fragments are collected from its llms.txt
    index and merged into one definition per module;
  * App Xchange connector pages on help.trimble.com are scanned for their
    per-module Direct API definitions;
  * the manifest's "documented" entries (endpoints documented in prose with
    no definition) are turned into definitions, after checking that every
    path still appears on its documentation page;
  * Trimble Identity's OpenID Connect discovery document is saved so every
    Identity endpoint is accounted for.

Failures are never silent: a sitemap, page or definition that cannot be
retrieved after retries is recorded with an "error", the script exits
non-zero, and the generator refuses to build from a run that has errors.
Known-unavailable definitions are listed in the manifest's "unavailable".

Output (in OUT, default /tmp/trimble-specs/other):
  <source-id with / as __>.json   normalised JSON definition (YAML converted)
  discovered.json                 [{id, spec_url, doc_urls, kind, error?}]

Requires Python 3 and PyYAML (for YAML definitions).
"""
import concurrent.futures as cf
import html
import json
import os
import re
import sys
import time
import urllib.error
import urllib.parse
import urllib.request

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
MANIFEST = os.path.join(ROOT, "cmd", "trimble-catalog", "sources-other.json")
UA = "trimble-mcp-bridge-catalog/1.0"


class NotFound(Exception):
    pass


class Challenged(Exception):
    """The site answered with a bot challenge (HTTP 202). It is never evaded."""


def get(url, timeout=60, tries=3):
    """GET with retries for transient failures; 404/410 raise NotFound."""
    url = urllib.parse.quote(url, safe=":/?&=%#@+,;~!$'()*[]")
    last = None
    for i in range(tries):
        req = urllib.request.Request(url, headers={"User-Agent": UA, "Accept": "*/*"})
        try:
            with urllib.request.urlopen(req, timeout=timeout) as r:
                if r.status == 202:
                    raise Challenged(f"HTTP 202 challenge from {urllib.parse.urlparse(url).netloc}")
                return r.read()
        except Challenged:
            raise
        except urllib.error.HTTPError as e:
            if e.code in (404, 410):
                raise NotFound(f"HTTP {e.code}")
            last = e
            if e.code < 500 and e.code != 429:
                break
        except Exception as e:  # network errors, timeouts
            last = e
        time.sleep(2 ** i)
    raise last


def to_json(raw):
    """Parse JSON, JSON with trailing commas, or YAML."""
    text = raw.decode("utf-8-sig")
    try:
        return json.loads(text)
    except ValueError:
        pass
    try:
        return json.loads(re.sub(r",(\s*[}\]])", r"\1", text))
    except ValueError:
        pass
    # A Swagger UI page with the definition embedded inline.
    m = re.search(r'\{\s*"(?:openapi|swagger|asyncapi)"\s*:', text)
    if m and text.lstrip().startswith("<"):
        return json.JSONDecoder().raw_decode(text, m.start())[0]
    import yaml  # PyYAML

    spec = yaml.safe_load(text)
    if not isinstance(spec, dict):
        raise ValueError("not an API definition")
    return spec


def slug(s):
    return re.sub(r"[^a-z0-9]+", "-", s.lower()).strip("-")


SPEC_URL = re.compile(r"""spec-url\s*=\s*(?:"([^"]+)"|'([^']+)'|([^\s>"']+))""", re.I)
SPEC_LINK = re.compile(r"""href\s*=\s*["']?([^"'\s>]*(?:openapi|asyncapi|swagger)[^"'\s>]*\.(?:json|ya?ml))["'\s>]""", re.I)


def spec_links(page, body):
    out = [next(g for g in m.groups() if g) for m in SPEC_URL.finditer(body)]
    out += [m.group(1) for m in SPEC_LINK.finditer(body)]
    return sorted({urllib.parse.urljoin(page, html.unescape(u)) for u in out})


def scan_pages(pages, errors):
    """Fetch pages concurrently; return {spec_url: set(page_url)}."""
    found = {}

    def one(page):
        try:
            return page, spec_links(page, get(page).decode("utf-8", "replace")), None
        except NotFound as e:
            print(f"warn: {page}: {e} (listed in a sitemap but gone)", file=sys.stderr)
            return page, [], None
        except Exception as e:
            return page, [], str(e)

    with cf.ThreadPoolExecutor(16) as ex:
        for page, urls, err in ex.map(one, sorted(set(pages))):
            if err:
                errors.append({"id": "page:" + page, "spec_url": page, "doc_urls": [page], "kind": "page", "error": err})
            for u in urls:
                found.setdefault(u, set()).add(page)
    print(f"developer portal: {len(set(pages))} pages scanned", file=sys.stderr)
    return found


def portal_pages(base, sections, errors):
    pages = []
    for s in sections:
        index = f"{base}/docs/{s}/sitemap-index.xml"
        try:
            idx = get(index).decode()
        except Exception as e:
            errors.append({"id": "sitemap:" + s, "spec_url": index, "doc_urls": [], "kind": "sitemap", "error": str(e)})
            continue
        for sm in re.findall(r"<loc>([^<]+)</loc>", idx):
            try:
                pages += re.findall(r"<loc>([^<]+)</loc>", get(sm).decode())
            except Exception as e:
                errors.append({"id": "sitemap:" + sm, "spec_url": sm, "doc_urls": [], "kind": "sitemap", "error": str(e)})
    return pages


def crawl_site(base, seeds, errors, limit=6000):
    """Breadth-first crawl of one site's HTML pages (same host only)."""
    host = urllib.parse.urlparse(base).netloc
    seen, frontier, bodies = set(), [urllib.parse.urljoin(base, s) for s in seeds], {}
    skip = re.compile(r"\.(png|jpe?g|gif|svg|ico|css|js|pdf|zip|json|ya?ml|xml|txt|woff2?|ttf|eot|mp4|webm)$", re.I)

    def one(page):
        try:
            return page, get(page).decode("utf-8", "replace"), None
        except NotFound:
            return page, "", None
        except urllib.error.HTTPError as e:
            if e.code == 403:  # the site's CDN answers 403 for links to missing objects
                print(f"warn: {page}: HTTP 403 (broken link on the site)", file=sys.stderr)
                return page, "", None
            return page, "", str(e)
        except Exception as e:
            return page, "", str(e)

    with cf.ThreadPoolExecutor(16) as ex:
        while frontier and len(seen) < limit:
            batch = [p for p in dict.fromkeys(frontier) if p not in seen][: limit - len(seen)]
            frontier = []
            seen.update(batch)
            for page, body, err in ex.map(one, batch):
                if err:
                    errors.append({"id": "page:" + page, "spec_url": page, "doc_urls": [page], "kind": "page", "error": err})
                    continue
                bodies[page] = body
                for m in re.findall(r"""href\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>"']+))""", body):
                    href = html.unescape(m[0] or m[1] or m[2]).split("#")[0]
                    if not href or "\\" in href or href.startswith(("javascript:", "mailto:", "data:")):
                        continue  # escaped markup inside scripts, not a link
                    u = urllib.parse.urljoin(page, href).split("?")[0]
                    pu = urllib.parse.urlparse(u)
                    if pu.scheme == "https" and pu.netloc == host and not skip.search(pu.path) and u not in seen:
                        frontier.append(u)
    if frontier:
        errors.append({"id": "crawl:" + host, "spec_url": base, "doc_urls": [], "kind": "page",
                       "error": f"crawl stopped at {limit} pages; raise the limit"})
    found = {}
    for page, body in bodies.items():
        for u in spec_links(page, body):
            found.setdefault(u, set()).add(page)
    print(f"{host}: {len(bodies)} pages crawled", file=sys.stderr)
    return found


def portal_id(doc_path, spec_url):
    # /docs/<section>/reference/openapi/<rest>/ -> <section>/<rest>
    m = re.match(r"^/docs/([^/]+)/reference/openapi/(.*?)/?$", doc_path)
    if m:
        rest = m.group(2).replace("controllers/", "")
        return f"{m.group(1)}/{slug(rest) or 'v1'}"
    section = re.match(r"^/docs/([^/]+)/", doc_path)
    path = urllib.parse.urlparse(spec_url).path
    stem = slug(os.path.splitext(path.split(f"/docs/{section.group(1)}/", 1)[-1] if section else path)[0])
    return f"{section.group(1) if section else 'portal'}/{stem}"


def maps_id(spec_url):
    p = urllib.parse.urlparse(spec_url).path
    return "trimble-maps/" + slug(os.path.splitext(p.rsplit("/api/", 1)[-1])[0])


def vista(index_url, errors):
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
    with cf.ThreadPoolExecutor(16) as ex:
        for module, url, frag, err in ex.map(fragment, ops):
            if err:
                errors.append({"id": "vista-page:" + url, "spec_url": url, "doc_urls": [url], "kind": "vista", "error": err})
                continue
            spec = merged.setdefault(module, {"openapi": frag.get("openapi", "3.0.1"), "info": frag.get("info", {}),
                                              "servers": frag.get("servers", []), "paths": {}, "components": {},
                                              "x-doc-pages": []})
            spec["x-doc-pages"].append(url)
            for p, item in frag.get("paths", {}).items():
                spec["paths"].setdefault(p, {}).update(item)
            for section, entries in frag.get("components", {}).items():
                spec["components"].setdefault(section, {}).update(entries)
    print(f"vista: {len(ops)} operation pages, {len(merged)} modules", file=sys.stderr)
    return merged


DIRECT_UI = re.compile(r"""https://api\.xchange\.trimble\.com/connect/v1/(?:direct|appnetwork)/[^"'\s<>?#]+?/swagger/index\.html""")


def app_xchange(cfg, errors):
    """Per-module Direct API definitions linked from App Xchange connector pages.

    help.trimble.com answers unknown clients with a bot challenge, which is
    never evaded. The manifest therefore lists every connector page and its
    definitions (captured with curl on the date in "checked"); when the index
    is readable, any connector page missing from the manifest is an error.
    """
    listed = {c["page"] for c in cfg["connectors"]}
    try:
        body = html.unescape(get(cfg["index"]).decode("utf-8", "replace"))
        prefix = cfg["index"].rstrip("/") + "/"
        pages = {u for u in (urllib.parse.urljoin(cfg["index"], h).split("#")[0].split("?")[0]
                             for h in re.findall(r"""href\s*=\s*["']([^"']+)["']""", body)) if u.startswith(prefix)}
        for p in sorted(pages - listed):
            errors.append({"id": "app-xchange:" + p, "spec_url": p, "doc_urls": [p], "kind": "page",
                           "error": "connector page not listed in the manifest's app_xchange.connectors; add it with its definitions"})
    except Challenged as e:
        print(f"warn: {cfg['index']}: {e}; using the manifest's connector list (checked {cfg['checked']})", file=sys.stderr)
    found = {}
    for c in cfg["connectors"]:
        for u in c["definitions"]:
            found.setdefault(u, set()).add(c["page"])
    # Trimble's connector directory (appxchange.trimble.com) lists every
    # connector's Direct API and App Network definitions; it is read
    # directly and merged with the manifest's list.
    try:
        body = html.unescape(get(cfg["directory"]).decode("utf-8", "replace"))
        for ui in sorted(set(DIRECT_UI.findall(body))):
            found.setdefault(ui[: -len("index.html")] + "openapi.json", set()).add(cfg["directory"])
    except Exception as e:
        errors.append({"id": "app-xchange-directory", "spec_url": cfg["directory"], "doc_urls": [], "kind": "page", "error": str(e)})
    print(f"app xchange: {len(listed)} connector pages, {len(found)} definitions", file=sys.stderr)
    return found


DEFINITION = re.compile(r"(swagger|openapi|api-docs|asyncapi|/documentation/)|\.ya?ml$", re.I)


def confluence(cfg, errors):
    """Definition links on every page of a public Confluence documentation
    space (Transporeon), read through Confluence's REST API."""
    nxt = "/rest/api/content/search?" + urllib.parse.urlencode(
        {"cql": f"space={cfg['space']} and type=page", "limit": "50", "expand": "body.storage"})
    found, inline, pages = {}, {}, 0
    while nxt:
        try:
            d = json.loads(get(cfg["base"] + nxt))
        except Exception as e:
            errors.append({"id": "confluence:" + cfg["space"], "spec_url": cfg["base"] + nxt, "doc_urls": [], "kind": "page", "error": str(e)})
            break
        for r in d.get("results", []):
            pages += 1
            page = cfg["base"] + r["_links"]["webui"]
            body = r["body"]["storage"]["value"]
            for u in re.findall(r"""https?://[^"'<>\s\]]+""", body):
                u = html.unescape(u)
                if DEFINITION.search(u) and not any(x in u for x in cfg["ignore"]):
                    found.setdefault(u, set()).add(page)
            # Definitions embedded inline in the open-api macro (CDATA body).
            macros = re.findall(r'<ac:structured-macro[^>]*ac:name="open-api".*?</ac:structured-macro>', body, re.S)
            bodies = [m for m in (re.search(r"<!\[CDATA\[(.*?)\]\]>", x, re.S) for x in macros) if m]
            for n, m in enumerate(bodies):
                sid = cfg["id_prefix"] + "inline-" + slug(r["title"]) + (f"-{n + 1}" if len(bodies) > 1 else "")
                inline[sid] = (m.group(1), page)
            # Interfaces specified only as attached PDFs are reported for review.
            if re.search(r'ac:name="(viewpdf|view-file)"', body):
                print(f"note: {page}: specification attached as a file (classify in capability-matrix.md)", file=sys.stderr)
        nxt = d.get("_links", {}).get("next")
    print(f"{cfg['space']}: {pages} pages, {len(found)} linked and {len(inline)} inline definitions", file=sys.stderr)
    return found, inline


def documented(entry, errors):
    """A definition for endpoints documented only in prose, checked against the page."""
    try:
        page = html.unescape(get(entry["doc_url"]).decode("utf-8", "replace"))
    except Exception as e:
        errors.append({"id": entry["id"], "spec_url": entry["doc_url"], "doc_urls": [entry["doc_url"]], "kind": "doc", "error": str(e)})
        return None
    paths = {}
    for op in entry["ops"]:
        if op["path"] not in page:
            errors.append({"id": entry["id"], "spec_url": entry["doc_url"], "doc_urls": [entry["doc_url"]], "kind": "doc",
                           "error": f"path {op['path']} no longer appears on the documentation page; review the entry"})
        params = [{"name": p["name"], "in": p["in"], "required": p.get("required", p["in"] == "path"),
                   "schema": {"type": p.get("type", "string")}} for p in op.get("params", [])]
        o = {"summary": op["summary"], "parameters": params}
        if op.get("body"):
            o["requestBody"] = {"content": {"application/json": {"schema": {"type": "object"}}}}
        paths.setdefault(op["path"], {})[op["method"].lower()] = o
    return {"openapi": "3.0.3", "info": {"title": entry["title"], "version": "documented"},
            "servers": [{"url": s} for s in entry["servers"]], "paths": paths,
            "x-note": "Built from the documentation page by scripts/fetch-trimble-other-specs.py; Trimble publishes no definition."}


def save(out, sid, spec):
    json.dump(spec, open(os.path.join(out, sid.replace("/", "__") + ".json"), "w"), sort_keys=True)


def main():
    out = sys.argv[1] if len(sys.argv) > 1 else "/tmp/trimble-specs/other"
    os.makedirs(out, exist_ok=True)
    man = json.load(open(MANIFEST))
    unavailable = {u["url"]: u["reason"] for u in man.get("unavailable", [])}
    discovered, errors = [], []

    def fetch_defs(found, idf, kind):
        for spec_url, docs in sorted(found.items()):
            docs = sorted(docs)
            rec = {"id": idf(spec_url, docs), "spec_url": spec_url, "doc_urls": docs, "kind": kind}
            if spec_url.startswith("https://api.swaggerhub.com/apis/Trimble-Connect/"):
                discovered.append(rec)  # catalogued from the SwaggerHub organisation
                continue
            try:
                save(out, rec["id"], to_json(get(spec_url)))
            except Exception as e:
                if spec_url in unavailable:
                    rec["unavailable"] = f"{unavailable[spec_url]} (retrieval failed: {e})"
                else:
                    rec["error"] = str(e)
            discovered.append(rec)

    pages = portal_pages(man["portal"]["base"], man["portal"]["sections"], errors)
    # Name a definition after its OpenAPI reference page when it has one, so
    # ids stay stable when guides also link the same definition.
    ref_page = lambda d: next((x for x in d if "/reference/openapi/" in x), d[0])
    fetch_defs(scan_pages(pages, errors), lambda u, d: portal_id(urllib.parse.urlparse(ref_page(d)).path, u), "portal")
    maps = man["maps"]
    fetch_defs(crawl_site(maps["base"], maps["seeds"], errors), lambda u, d: maps_id(u), "maps")
    fetch_defs(app_xchange(man["app_xchange"], errors),
               lambda u, d: ("xchange-connector/" + slug(u.split("/direct/", 1)[1]) if "/direct/" in u else
                             "xchange-appnetwork/" + slug(u.split("/appnetwork/", 1)[1])).removesuffix("-swagger-openapi-json"), "app-xchange")

    for c in man["confluence"]:
        linked, inline = confluence(c, errors)
        fetch_defs(linked, lambda u, d, c=c: c["id_prefix"] + slug(urllib.parse.unquote(
            urllib.parse.urlparse(u).netloc.split(".")[-2] + urllib.parse.urlparse(u).path)), "confluence")
        for sid, (text, page) in sorted(inline.items()):
            rec = {"id": sid, "spec_url": page, "doc_urls": [page], "kind": "confluence"}
            try:
                save(out, sid, to_json(text.encode()))
            except Exception as e:
                rec["error"] = f"inline definition does not parse: {e}"
            discovered.append(rec)

    for d in man["direct"]:
        rec = {"id": d["id"], "spec_url": d["url"], "doc_urls": [d["doc_url"]] if d["doc_url"] else [], "kind": "direct"}
        try:
            save(out, d["id"], to_json(get(d["url"])))
        except Exception as e:
            rec["error"] = str(e)
        discovered.append(rec)

    for entry in man["documented"]:
        spec = documented(entry, errors)
        if spec:
            save(out, entry["id"], spec)
            discovered.append({"id": entry["id"], "spec_url": entry["doc_url"], "doc_urls": [entry["doc_url"]], "kind": "doc"})

    idn = man["identity"]
    rec = {"id": idn["id"], "spec_url": idn["url"], "doc_urls": [idn["doc_url"]], "kind": "oidc"}
    try:
        save(out, idn["id"], to_json(get(idn["url"])))
    except Exception as e:
        rec["error"] = str(e)
    discovered.append(rec)

    for module, spec in sorted(vista(man["vista"]["index"], errors).items()):
        sid = "vista/" + slug(module.replace("Vista ", "").replace(" v2 Direct API", ""))
        save(out, sid, spec)
        discovered.append({"id": sid, "spec_url": man["vista"]["index"], "doc_urls": spec["x-doc-pages"], "kind": "vista"})

    discovered += errors
    ids = [r["id"] for r in discovered if not r.get("error")]
    for i in sorted({i for i in ids if ids.count(i) > 1}):
        discovered.append({"id": i, "spec_url": "", "doc_urls": [], "kind": "page", "error": "two definitions map to this id"})
    json.dump(discovered, open(os.path.join(out, "discovered.json"), "w"), indent=1)
    bad = [r for r in discovered if r.get("error")]
    for r in bad:
        print(f"error: {r['id']}: {r['error']}", file=sys.stderr)
    print(f"{len(discovered)} records ({len(bad)} errors) in {out}/discovered.json", file=sys.stderr)
    sys.exit(1 if bad else 0)


if __name__ == "__main__":
    main()

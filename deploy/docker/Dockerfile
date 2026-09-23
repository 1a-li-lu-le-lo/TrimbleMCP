# syntax=docker/dockerfile:1
FROM golang:1.24 AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/bin/trimble-mcp ./cmd/trimble-mcp \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/bin/trimblectl ./cmd/trimblectl \
 && mkdir -p /out/data

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/bin/ /usr/local/bin/
# The audit directory must exist and be writable by nonroot; mount a
# persistent volume here so audit evidence survives restarts.
COPY --from=build --chown=nonroot:nonroot /out/data /var/lib/trimble-mcp
VOLUME ["/var/lib/trimble-mcp"]
USER nonroot:nonroot
# Remote deployments never expose the synthetic mock product by default.
ENV TRIMBLE_MCP_HTTP_ADDR=0.0.0.0:8787 TRIMBLE_MCP_AUDIT_LOG=/var/lib/trimble-mcp/audit.jsonl TRIMBLE_MCP_ENABLE_MOCK=false
EXPOSE 8787
ENTRYPOINT ["/usr/local/bin/trimble-mcp", "-transport", "http"]

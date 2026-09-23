# syntax=docker/dockerfile:1
FROM golang:1.24 AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/trimble-mcp ./cmd/trimble-mcp \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/trimblectl ./cmd/trimblectl

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/ /usr/local/bin/
USER nonroot:nonroot
ENV TRIMBLE_MCP_HTTP_ADDR=0.0.0.0:8787 TRIMBLE_MCP_AUDIT_LOG=/var/lib/trimble-mcp/audit.jsonl
EXPOSE 8787
ENTRYPOINT ["/usr/local/bin/trimble-mcp", "-transport", "http"]

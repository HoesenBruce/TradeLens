# Separate demo image: the production Dockerfiles and publishing stay unchanged.
FROM golang:1.27-bookworm AS api
WORKDIR /src
COPY api/go.mod api/go.sum ./
RUN go mod download
COPY api/ ./
COPY VERSION /VERSION
RUN CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/tradermemos/api/internal/version.Version=$(cat /VERSION)" -o /server ./cmd/server

FROM --platform=$BUILDPLATFORM node:24-bookworm-slim AS web
WORKDIR /web
ENV CI=true
RUN corepack enable && corepack prepare pnpm@11.16.0 --activate
COPY web/package.json web/pnpm-lock.yaml web/pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
COPY VERSION /VERSION
RUN pnpm run build

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates nginx python3 \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --system --uid 10001 --create-home demo
COPY --from=api /server /server
COPY --from=web /web/dist /app/web
COPY scripts/seed-demo.py scripts/seed-showcase.py scripts/showcase-market.py /app/scripts/
COPY deploy/demo/ /app/demo/
USER demo
ENV PORT=10000 PYTHONUNBUFFERED=1 PYTHONDONTWRITEBYTECODE=1
EXPOSE 10000
HEALTHCHECK --interval=30s --timeout=5s --start-period=120s CMD python3 /app/demo/runtime.py --healthcheck
ENTRYPOINT ["python3", "/app/demo/runtime.py"]

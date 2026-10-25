# Mandalore server image: mandalore serve --config /config/server.json.
#
# Development: docker build -f packaging/server/Dockerfile .
# Release: bin/publish-server-image replaces the build stage with the accepted
# candidate's own Linux binaries (--build-context build=DIR), so the image
# carries the released bytes rather than a fresh build.
FROM golang:1.26.10-bookworm AS build
ARG TARGETARCH
ARG VERSION=0.0.0-dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd cmd
COPY internal internal
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -trimpath -buildvcs=false \
      -ldflags "-buildid= -X main.version=$VERSION" -o /out/mandalore-linux-$TARGETARCH ./cmd/mandalore

FROM debian:bookworm-slim
ARG TARGETARCH
# The server runs git (checkpoint and push over SSH or HTTPS); curl serves the
# health check, since /healthz answers loopback peers only.
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl git openssh-client \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --gid 10001 mandalore \
    && useradd --uid 10001 --gid 10001 --home-dir /var/lib/mandalore --create-home --shell /usr/sbin/nologin mandalore \
    && install -d -o 10001 -g 10001 -m 0700 /state \
    && install -d -m 0755 /config
COPY --from=build --chmod=0755 /out/mandalore-linux-$TARGETARCH /usr/local/bin/mandalore
USER 10001:10001
ENV HOME=/var/lib/mandalore
VOLUME ["/state"]
EXPOSE 8730
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD curl -fsS --max-time 4 http://127.0.0.1:8730/healthz || exit 1
ENTRYPOINT ["mandalore"]
CMD ["serve", "--config", "/config/server.json"]

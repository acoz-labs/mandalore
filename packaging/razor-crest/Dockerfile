FROM --platform=$BUILDPLATFORM golang:1.26.4-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGETOS
ARG TARGETARCH
ARG SOURCE_COMMIT
RUN test "${#SOURCE_COMMIT}" = 40 && echo "$SOURCE_COMMIT" | grep -Eq '^[0-9a-f]+$' && \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags "-X main.version=$(cat VERSION) -X main.sourceCommit=$SOURCE_COMMIT" -o /mandalore ./cmd/mandalore

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates git openssh-client curl && rm -rf /var/lib/apt/lists/* && useradd --uid 10001 --home-dir /app --create-home mandalore
COPY --from=build /mandalore /usr/local/bin/mandalore
USER 10001:10001
WORKDIR /app
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s CMD curl --fail --silent http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["mandalore", "razor-crest", "serve", "--config", "/config/service.json"]

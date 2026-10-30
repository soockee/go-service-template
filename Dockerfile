# syntax=docker/dockerfile:1

# Build on the native platform and cross-compile, so multi-arch builds
# need no emulation.
FROM --platform=$BUILDPLATFORM golang:1.27 AS build
ARG BUILDARCH
ARG TARGETOS
ARG TARGETARCH
ARG TAILWIND_VERSION=v4.1.11
ARG VERSION=dev

RUN case "$BUILDARCH" in arm64) tw=linux-arm64 ;; *) tw=linux-x64 ;; esac && \
    curl -fsSL -o /usr/local/bin/tailwindcss \
      "https://github.com/tailwindlabs/tailwindcss/releases/download/${TAILWIND_VERSION}/tailwindcss-${tw}" && \
    chmod +x /usr/local/bin/tailwindcss

WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go tool templ generate && \
    tailwindcss -i assets/css/input.css -o assets/css/output.css --minify && \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/server ./cmd/server

# Assets and migrations are embedded; the image is just the binary.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /server
ENV APP_ENV=prod ADDR=:8080
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/server"]

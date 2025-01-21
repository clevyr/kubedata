#syntax=docker/dockerfile:1.13

FROM --platform=$BUILDPLATFORM golang:1.23.3-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Set Golang build envs based on Docker platform string
ARG TARGETPLATFORM
RUN <<EOT
  set -eux
  case "$TARGETPLATFORM" in
    'linux/amd64') export GOARCH=amd64 ;;
    'linux/arm/v6') export GOARCH=arm GOARM=6 ;;
    'linux/arm/v7') export GOARCH=arm GOARM=7 ;;
    'linux/arm64') export GOARCH=arm64 ;;
    *) echo "Unsupported target: $TARGETPLATFORM" && exit 1 ;;
  esac
  go build -ldflags='-w -s'
EOT


FROM alpine:3.21
LABEL org.opencontainers.image.source="https://github.com/clevyr/kubedb"
WORKDIR /data

COPY --from=builder /app/kubedb /usr/local/bin/

ENV KUBECONFIG /.kube/config
ENTRYPOINT ["kubedb"]

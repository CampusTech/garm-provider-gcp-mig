# Image holding only the provider binary, to copy into a GARM image:
#   COPY --from=ghcr.io/ut0mt8/garm-provider-gcp-mig:v0.1.0 /garm-provider-gcp-mig /opt/garm/providers.d/
FROM golang:1.26 AS build
ARG VERSION=v0.0.0-dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go test ./... && CGO_ENABLED=0 go build -trimpath \
    -ldflags "-s -w -X github.com/ut0mt8/garm-provider-gcp-mig/provider.Version=${VERSION}" \
    -o /garm-provider-gcp-mig .

FROM scratch
COPY --from=build /garm-provider-gcp-mig /garm-provider-gcp-mig
ENTRYPOINT ["/garm-provider-gcp-mig"]

FROM golang:1.17.8-alpine3.15 as builder
RUN apk add --no-cache gcc=~10.3 pkgconfig=~1.7 musl-dev=~1.2 libgit2-dev=~1.3
WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download
COPY main.go main.go
COPY pkg/ pkg/
RUN CGO_ENABLED=1 go build -o gitops-promotion main.go

FROM alpine:3.15.0
LABEL org.opencontainers.image.source="https://github.com/XenitAB/gitops-promotion"
# hadolint ignore=DL3017,DL3018
RUN apk upgrade --no-cache && \
    apk add --no-cache ca-certificates tini=~0.19 libgit2=~1.3
COPY --from=builder /workspace/gitops-promotion /usr/local/bin/
COPY ./action-entrypoint.sh /usr/local/bin/
WORKDIR /workspace
ENTRYPOINT [ "/sbin/tini", "--", "gitops-promotion" ]

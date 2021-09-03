FROM golang:1.16-alpine as builder
RUN apk add --no-cache gcc=~10.3 pkgconfig=~1.7 libc-dev=~0.7 musl=~1.2 libgit2-dev=~1.1
WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download
COPY main.go main.go
COPY pkg/ pkg/
RUN CGO_ENABLED=1 go build -o gitops-promotion main.go

FROM alpine:3.14.0
LABEL org.opencontainers.image.source="https://github.com/XenitAB/gitops-promotion"
# hadolint ignore=DL3018
RUN apk add --no-cache ca-certificates tini=~0.19 libgit2=~1.1 musl=~1.2
COPY --from=builder /workspace/gitops-promotion /usr/local/bin/
WORKDIR /workspace
ENTRYPOINT [ "/sbin/tini", "--", "gitops-promotion" ]

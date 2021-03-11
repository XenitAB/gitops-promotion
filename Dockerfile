FROM golang:1.16-alpine as builder

RUN apk add gcc pkgconfig libc-dev
RUN apk add --no-cache musl~=1.2 libgit2-dev~=1.1

WORKDIR /workspace

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY main.go main.go
COPY pkg/ pkg/

RUN CGO_ENABLED=1 go build -o gitops-promotion main.go

FROM alpine:3.13
LABEL org.opencontainers.image.source="https://github.com/xenitab/gitops-promotion"

RUN apk add --no-cache ca-certificates tini libgit2~=1.1 musl~=1.2

COPY --from=builder /workspace/gitops-promotion /usr/local/bin/

WORKDIR /workspace

ENTRYPOINT [ "/sbin/tini", "--", "gitops-promotion" ]

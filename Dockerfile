FROM golang:1.16-alpine as builder
RUN apk add --no-cache gcc~=10.2.1_pre1-r3 pkgconfig~=1.7.3-r0 libc-dev~=0.7.2-r3
RUN apk add --no-cache musl~=1.2 libgit2-dev~=1.1
WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download
COPY main.go main.go
COPY pkg/ pkg/
RUN CGO_ENABLED=1 go build -o gitops-promotion main.go

FROM alpine:3.14.1
LABEL org.opencontainers.image.source="https://github.com/XenitAB/gitops-promotion"
RUN apk add --no-cache ca-certificates=20191127-r5 tini~=0.19.0-r0 libgit2~=1.1 musl~=1.2
COPY --from=builder /workspace/gitops-promotion /usr/local/bin/
WORKDIR /workspace
ENTRYPOINT [ "/sbin/tini", "--", "gitops-promotion" ]

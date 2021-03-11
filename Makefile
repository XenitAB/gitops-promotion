TAG = latest
IMG ?= docker.io/phillebaba/gitops-promotion:$(TAG)

lint:
	golangci-lint run -E misspell

fmt:
	go fmt ./...

vet:
	go vet ./...

test: fmt vet
	go test ./...

docker-build:
	docker build -t ${IMG} .

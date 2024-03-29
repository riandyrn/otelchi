.PHONY: *

GO_VERSIONS="1.15 1.16 1.17"

go-build:
	go build .

go-test:
	go test ./...

test:
	docker build \
		-t go-test \
		--build-arg GO_VERSIONS=${GO_VERSIONS} \
		-f ./test/infras/Dockerfile . && \
		docker run --rm go-test
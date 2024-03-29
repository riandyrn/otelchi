.PHONY: *

GO_VERSIONS="1.15 1.16 1.17"

# This is the command that will be used to run the tests
go-test:
	go build .
	go test ./...

# This is the command that will be used to run the tests in a Docker container, useful when executing the test locally
test:
	docker build \
		-t go-test \
		--build-arg GO_VERSIONS=${GO_VERSIONS} \
		-f ./test/infras/Dockerfile . && \
		docker run --rm go-test
.PHONY: *

# GO_VERSIONS="1.15 1.16 1.17"

# debug
GO_VERSIONS="1.15"

test:
	docker build \
		-t go-test \
		--build-arg GO_VERSIONS=${GO_VERSIONS} \
		-f ./test/infras/linux/Dockerfile . && \
		docker run --rm go-test

test-windows:
	docker build \
		-t go-test \
		--build-arg GO_VERSIONS=${GO_VERSIONS} \
		-f ./test/infras/windows/Dockerfile . && \
		docker run --rm go-test
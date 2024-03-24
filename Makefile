.PHONY: *

test:
	docker build -t go-test -f ./deploy/test/Dockerfile .
	docker run --rm -it go-test
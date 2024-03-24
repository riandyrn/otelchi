.PHONY: *

test:
	docker build -t go-test -f ./deploy/test/Dockerfile .
	docker run --rm -it -v ${PWD}:/go/src/github.com/riandyrn/otelchi go-test
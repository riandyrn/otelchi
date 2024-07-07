FROM golang:1.21-alpine3.20
WORKDIR /go/src/github.com/riandyrn/otelchi

RUN apk --no-cache add curl
COPY . .

WORKDIR /go/src/github.com/riandyrn/otelchi/examples/multi-services
RUN go mod download -x

WORKDIR /go/src/github.com/riandyrn/otelchi/examples/multi-services/back-svc
RUN go build -o server

ENTRYPOINT ["./server"]

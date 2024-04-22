FROM golang:1.18-alpine3.14

WORKDIR /go/src/github.com/riandyrn/otelchi
COPY . .

WORKDIR /go/src/github.com/riandyrn/otelchi/examples/basic
RUN go mod download -x
RUN go build -o server

ENTRYPOINT ["./server"]
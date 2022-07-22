FROM golang:1.15.6-alpine3.12
WORKDIR /go/src/github.com/riandyrn/otelchi/examples/basic

COPY . .

RUN go build -o server
CMD ["./server"]
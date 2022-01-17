FROM golang:1.15.6-alpine3.12
WORKDIR /go/src/github.com/riandyrn/otelchi

COPY . .

WORKDIR /go/src/github.com/riandyrn/otelchi/example
RUN go build -o server
CMD ["./server"]
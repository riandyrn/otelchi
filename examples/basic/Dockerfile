FROM golang:1.15-alpine3.12
WORKDIR /go/src/github.com/riandyrn/otelchi/examples/basic

COPY go.mod go.sum ./
RUN go mod download -x

COPY . .

RUN go build -o server
ENTRYPOINT ["./server"]
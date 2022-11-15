FROM golang:1.19.3-alpine3.16
WORKDIR /go/src/github.com/riandyrn/otelchi/examples/multi-services

RUN apk --no-cache add curl

COPY go.mod go.sum ./
RUN go mod download -x

COPY ./back-svc ./back-svc
COPY ./utils ./utils
WORKDIR /go/src/github.com/riandyrn/otelchi/examples/multi-services/back-svc
RUN go build -o server

CMD ["./server"]
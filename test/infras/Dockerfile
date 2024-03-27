FROM golang:1.22.1

# Set Working Directory
WORKDIR /go/src/github.com/riandyrn/otelchi

# Install Dependencies
COPY go.mod go.sum ./
RUN go mod download -x

# Copy Source Code
COPY . .

ENTRYPOINT ["go", "test", "./..."]
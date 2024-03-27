FROM golang:1.22.1

# Set Shell to CMD for Windows
SHELL ["cmd", "/S", "/C"]

# Set Working Directory
WORKDIR /go/src/github.com/riandyrn/otelchi

# Install Dependencies
COPY go.mod go.sum ./
RUN go mod download -x

# Install Golang Versions
RUN go install golang.org/dl/go1.15@latest && go1.15 download && \
	go install golang.org/dl/go1.16@latest && go1.16 download && \
	go install golang.org/dl/go1.17@latest && go1.17 download

# Copy Source Code
COPY . .

ENTRYPOINT echo "Testing with Go 1.15" && go1.15 build && go1.15 test ./...
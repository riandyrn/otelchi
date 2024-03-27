FROM golang:1.22.1

# Set Working Directory
WORKDIR /go/src/github.com/riandyrn/otelchi

# Install Dependencies
COPY go.mod go.sum ./
RUN go mod download -x

# Copy Entrypoint File
COPY ./test/infras/entrypoint.sh ./test/infras/
RUN chmod +x ./test/infras/entrypoint.sh

# Copy Source Code
COPY . .

ENTRYPOINT [ "./test/infras/entrypoint.sh" ]
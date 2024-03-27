ARG GO_VERSION=1.22
FROM golang:${GO_VERSION}

# Install Gobrew
ENV PATH="/root/.gobrew/current/bin:/root/.gobrew/bin:$PATH"
RUN curl -sLk https://raw.githubusercontent.com/kevincobain2000/gobrew/master/git.io.sh | sh

# Set Working Directory
WORKDIR /go/src/github.com/riandyrn/otelchi

# Install Multiple Go Versions
ARG GO_VERSIONS
COPY ./test/infras/linux/install_go.sh ./test/infras/linux/
RUN chmod +x ./test/infras/linux/install_go.sh && sh ./test/infras/linux/install_go.sh

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download -x

# Copy entrypoint
COPY ./test/infras/linux/entrypoint.sh ./test/infras/linux/
RUN chmod +x ./test/infras/linux/entrypoint.sh

ENV GO_VERSIONS=$GO_VERSIONS

COPY . .

ENTRYPOINT [ "./test/infras/linux/entrypoint.sh" ]

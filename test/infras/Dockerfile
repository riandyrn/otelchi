ARG GO_VERSION=1.22

# linux
FROM golang:${GO_VERSION} as test-linux

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


# windows
FROM golang:${GO_VERSION} as test-windows

# Install Gobrew
RUN Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://raw.githubusercontent.com/kevincobain2000/gobrew/master/git.io.ps1'))

# Install Multiple Go Versions
ARG GO_VERSIONS
COPY ./test/infras/windows/install_go.ps1 /test/infras/windows/
RUN Set-ExecutionPolicy Bypass -Scope Process -Force -File /test/infras/windows/install_go.ps1

# Install dependencies
COPY go.mod go.sum /
RUN go mod download -x

# Copy entrypoint
COPY ./test/infras/windows/entrypoint.ps1 /test/infras/windows/
RUN Set-ExecutionPolicy Bypass -Scope Process -Force -File /test/infras/windows/entrypoint.ps1

ENV GO_VERSIONS=$GO_VERSIONS

COPY . .

ENTRYPOINT [ "powershell", "-File", "/test/infras/windows/entrypoint.ps1" ]

ARG TARGETOS
FROM test-${TARGETOS}

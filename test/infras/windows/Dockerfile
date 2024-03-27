ARG GO_VERSION=1.22
FROM golang:${GO_VERSION} as test-windows

# Install Gobrew
RUN Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://raw.githubusercontent.com/kevincobain2000/gobrew/master/git.io.ps1'))

# Install Multiple Go Versions
ARG GO_VERSIONS
COPY ./test/infras/windows/install_go.ps1 /test/infras/windows/
RUN Set-ExecutionPolicy Bypass -Scope Process -Force; iex /test/infras/windows/install_go.ps1

# Install dependencies
COPY go.mod go.sum /
RUN go mod download -x

# Copy entrypoint
COPY ./test/infras/windows/entrypoint.ps1 /test/infras/windows/
RUN Set-ExecutionPolicy Bypass -Scope Process -Force; iex /test/infras/windows/entrypoint.ps1

ENV GO_VERSIONS=$GO_VERSIONS

COPY . .

ENTRYPOINT [ "iex", "/test/infras/windows/entrypoint.ps1" ]

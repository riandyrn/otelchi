FROM debian:buster

RUN apt-get update && apt-get install -y \
	gcc \
	curl \
	git \
	unzip \
	wget \
	&& rm -rf /var/lib/apt/lists/*

# Install Gobrew
ENV PATH="/root/.gobrew/current/bin:/root/.gobrew/bin:$PATH"
RUN curl -sLk https://raw.githubusercontent.com/kevincobain2000/gobrew/master/git.io.sh | sh

# Set Working Directory
WORKDIR /go/src/github.com/riandyrn/otelchi

# Install Multiple Go Versions
ARG GO_VERSIONS
COPY ./test/infras/install_go.sh ./test/infras/
RUN chmod +x ./test/infras/install_go.sh && sh ./test/infras/install_go.sh

COPY go.mod go.sum ./
RUN go mod download -x

COPY ./test/infras/entrypoint.sh ./test/infras/
RUN chmod +x ./test/infras/entrypoint.sh

ENV GO_VERSIONS=$GO_VERSIONS

COPY . .

ENTRYPOINT [ "./test/infras/entrypoint.sh" ]
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
ARG GO_VERSIONS="1.16 1.17 1.18 1.19 1.20"
COPY ./deploy/test/install_go.sh ./deploy/test/
RUN chmod +x ./deploy/test/install_go.sh && sh ./deploy/test/install_go.sh

COPY go.mod go.sum ./
RUN go mod download -x

COPY ./deploy/test/entrypoint.sh ./deploy/test/
RUN chmod +x ./deploy/test/entrypoint.sh

ENV GO_VERSIONS=$GO_VERSIONS

COPY . .

ENTRYPOINT [ "./deploy/test/entrypoint.sh" ]
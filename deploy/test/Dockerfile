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

# Install Multiple Go Versions
RUN gobrew install 1.16
RUN gobrew install 1.17
RUN gobrew install 1.18
RUN gobrew install 1.19
RUN gobrew install 1.20

WORKDIR /go/src/github.com/riandyrn/otelchi

COPY go.mod go.sum ./
RUN go mod download -x

COPY . .

RUN chmod a+rx ./deploy/test/entrypoint.sh

ENTRYPOINT [ "./deploy/test/entrypoint.sh" ]
FROM python:3.14-slim

RUN groupadd --gid 1000 vscode \
    && useradd --uid 1000 --gid 1000 -m vscode -s /bin/bash \
    && apt-get update \
    && apt-get install -y sudo \
    && echo "vscode ALL=(root) NOPASSWD:ALL" > /etc/sudoers.d/vscode \
    && chmod 0440 /etc/sudoers.d/vscode

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    ca-certificates \
    git \
    build-essential \
    protobuf-compiler \
    postgresql-client \
    && apt-get clean && \
    rm -rf /var/lib/apt/lists/*

RUN curl -fsSL https://go.dev/dl/go1.26.0.linux-amd64.tar.gz | tar -C /usr/local -xzf -
ENV PATH=$PATH:/usr/local/go/bin

ENV GOPATH=/go
ENV PATH=$PATH:/go/bin
RUN mkdir -p /go/bin

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

RUN pip install --no-cache-dir mypy numpy pytest pytest-cov nats-py python-dotenv protobuf mypy-protobuf types-protobuf

RUN curl -L https://github.com/golang-migrate/migrate/releases/download/v4.18.2/migrate.linux-amd64.tar.gz | \
    tar xvz && \
    mv migrate /usr/local/bin/migrate

RUN echo "export PS1='\\[\\033[01;32m\\]\\u@\\h\\[\\033[00m\\]:\\[\\033[01;34m\\]\\w\\[\\033[00m\\]\\$ '" >> /home/vscode/.bashrc && \
    echo "alias ls='ls --color=auto'" >> /home/vscode/.bashrc && \
    chown vscode:vscode /home/vscode/.bashrc

RUN chown -R vscode:vscode /go

WORKDIR /workspace
RUN chown vscode:vscode /workspace

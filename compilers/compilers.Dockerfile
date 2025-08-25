# Base image
FROM ubuntu:22.04
ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=Etc/UTC

# Dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    pkg-config \
    git \
    curl \
    ca-certificates \
    libcap-dev \
    cgroup-tools \
    libssl-dev \
    zlib1g-dev \
    libbz2-dev \
    libsystemd-dev \
    libreadline-dev \
    libsqlite3-dev \
    libncursesw5-dev \
    xz-utils \
    tk-dev \
    libxml2-dev \
    libxmlsec1-dev \
    libffi-dev \
    liblzma-dev \
    locales \
    && rm -rf /var/lib/apt/lists/*

RUN set -xe && \
    echo "en_US.UTF-8 UTF-8" > /etc/locale.gen && \
    locale-gen
ENV LANG=en_US.UTF-8 LANGUAGE=en_US:en LC_ALL=en_US.UTF-8

# Language runtimes
# Golang
ENV GO_VERSION=1.24.6
RUN curl -fsSl "https://storage.googleapis.com/golang/go${GO_VERSION}.linux-amd64.tar.gz" -o /tmp/go-${GO_VERSION}.tar.gz && \
    mkdir /usr/local/go-${GO_VERSION} && \
    tar -xf /tmp/go-${GO_VERSION}.tar.gz -C /usr/local/go-${GO_VERSION} --strip-components=1 && \
    rm -rf /tmp/*;

# Python
ENV PYTHON_VERSION=3.13.6
RUN curl -fSsL "https://www.python.org/ftp/python/${PYTHON_VERSION}/Python-${PYTHON_VERSION}.tar.xz" -o /tmp/python-${PYTHON_VERSION}.tar.xz && \
    mkdir /tmp/python-${PYTHON_VERSION} && \
    tar -xf /tmp/python-${PYTHON_VERSION}.tar.xz -C /tmp/python-${PYTHON_VERSION} --strip-components=1 && \
    rm /tmp/python-${PYTHON_VERSION}.tar.xz && \
    cd /tmp/python-${PYTHON_VERSION} && \
    ./configure \
    --prefix=/usr/local/python-${PYTHON_VERSION} && \
    make -j$(nproc) && \
    make -j$(nproc) install && \
    rm -rf /tmp/*;


RUN set -xe && \
    git clone "https://github.com/ioi/isolate.git" /tmp/isolate && \
    cd /tmp/isolate && \
    make -j$(nproc) install && \
    rm -rf /tmp/*

ENV BOX_ROOT=/var/local/lib/isolate

LABEL maintainer="Ung Nguyen Song Phuc <songphucungnguyen@gmail.com>"
LABEL version="1"

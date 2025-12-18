FROM ubuntu:24.04

# Avoid interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive

# Install build dependencies
RUN apt-get update && apt-get install -y \
    # Build essentials
    build-essential \
    pkg-config \
    git \
    # Go (we'll install manually for latest version)
    wget \
    # SDL2 development libraries
    libsdl2-dev \
    libsdl2-ttf-dev \
    libsdl2-image-dev \
    libsdl2-mixer-dev \
    # OpenGL
    libgl1-mesa-dev \
    # SQLite3
    libsqlite3-dev \
    # Debian packaging
    dpkg-dev \
    && rm -rf /var/lib/apt/lists/*

# Install Go 1.24
RUN wget -q https://go.dev/dl/go1.24.2.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go1.24.2.linux-amd64.tar.gz \
    && rm go1.24.2.linux-amd64.tar.gz

ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/go"
ENV CGO_ENABLED=1

WORKDIR /workspace

# Default command shows Go version to verify setup
CMD ["go", "version"]

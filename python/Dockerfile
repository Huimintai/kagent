### STAGE 1: base image
ARG BASE_IMAGE=alpine:3.23
FROM $BASE_IMAGE AS base-os

ENV LANG=C.UTF-8
ENV LC_ALL=C.UTF-8

# Install system packages + uv (not in Alpine repos, use standalone installer)
RUN --mount=type=cache,target=/var/cache/apk,rw \
    apk update && apk add --no-cache \
        curl openssl bash git ca-certificates libstdc++ \
    && curl -LsSf https://astral.sh/uv/install.sh | sh \
    && cp /root/.local/bin/uv /usr/local/bin/uv \
    && cp /root/.local/bin/uvx /usr/local/bin/uvx \
    && chmod 755 /usr/local/bin/uv /usr/local/bin/uvx \
    && rm -rf /root/.local

### STAGE 2: python
FROM base-os AS python-os
ARG TOOLS_PYTHON_VERSION=3.13

ENV PYTHONOPTIMIZE=2
ENV PYTHONUNBUFFERED=1

# Optimize malloc for containerized Python workloads
# 256KB threshold balances memory efficiency with performance
ENV MALLOC_TRIM_THRESHOLD_=262144
ENV MALLOC_ARENA_MAX=2

ENV GIT_LFS_SKIP_SMUDGE=1

ENV UV_LINK_MODE=copy
ENV UV_COMPILE_BYTECODE=1
ENV UV_COMPILE_BYTECODE_TIMEOUT=300
ENV UV_SYSTEM_PYTHON=1
ENV UV_NO_PROGRESS=1
ENV UV_HTTP_TIMEOUT=60
ENV UV_CONCURRENT_DOWNLOADS=10

# Configure the Python directories
ENV UV_CACHE_DIR=/.kagent/cache/packages
ENV UV_TOOL_DIR=/.kagent/cache/tools
ENV UV_PYTHON_DOWNLOADS_DIR=/.kagent/cache/downloads
ENV UV_PROJECT_ENVIRONMENT=/.kagent/.venv

ENV UV_PYTHON_INSTALL_DIR=/python
ENV UV_PYTHON_PREFERENCE=only-managed

RUN addgroup -g 1001 pythongroup                           && \
    adduser  -u 1001 -G pythongroup -s /bin/bash -D python -h /.kagent/  && \
    mkdir    -p $UV_PYTHON_DOWNLOADS_DIR                   && \
    mkdir    -p $UV_TOOL_DIR                               && \
    mkdir    -p $UV_CACHE_DIR                              && \
    mkdir    -p /python                                    && \
    chown    -vR 1001:1001 /.kagent /python

# Install anthropic sandbox runtime and dependencies
RUN --mount=type=cache,target=/var/cache/apk,rw \
    apk add --no-cache \
    nodejs npm python3 make g++ bubblewrap socat ripgrep

# Make bwrap setuid-root so it can create user/network namespaces when invoked
# by non-root users on hosts with kernel.apparmor_restrict_unprivileged_userns=1
# (Ubuntu 23.10+). See https://github.com/openai/codex/issues/14919
RUN chmod u+s /usr/bin/bwrap

# Install sandbox runtime from a specific commit of the GitHub repo without using global prefix
# This avoids scope-related rename issues in global node_modules
# Using BuildKit cache for npm to speed up rebuilds
RUN --mount=type=cache,target=/root/.npm \
    mkdir -p /opt && \
    cd opt && \
    git clone --depth 1 --revision=ef4afdef4d711ba21a507d7f7369e305f7d3dbfa https://github.com/anthropic-experimental/sandbox-runtime.git && \
    cd sandbox-runtime && \
    npm install && \
    npm run build && \
    # CVE-2026-26996: all minimatch instances (3.1.2, 9.0.5) are transitive dev
    # deps (eslint, typescript-eslint). Prune dev deps after build to remove them.
    npm prune --omit=dev && \
    npm install -g --ignore-scripts

# Ensure the sandbox runtime binaries are on PATH
ENV PATH="/opt/sandbox-runtime/node_modules/.bin:$PATH"

USER python
WORKDIR /.kagent

### STAGE 3: final
FROM python-os AS builder
ARG TOOLS_PYTHON_VERSION

WORKDIR /.kagent

ENV PATH=$PATH:/.kagent/bin:/.kagent/.venv/bin

# Copy dependency files first for better layer caching
COPY --chown=python:pythongroup pyproject.toml .
COPY --chown=python:pythongroup .python-version .
COPY --chown=python:pythongroup uv.lock .
COPY --chown=python:pythongroup packages/kagent-adk packages/kagent-adk
COPY --chown=python:pythongroup packages/kagent-core packages/kagent-core
COPY --chown=python:pythongroup packages/kagent-skills packages/kagent-skills
COPY --chown=python:pythongroup packages/agentsts-adk packages/agentsts-adk
COPY --chown=python:pythongroup packages/agentsts-core packages/agentsts-core
COPY --chown=python:pythongroup README.md .

ARG VERSION

# Install dependencies - make sure /.kagent/.venv/bin in path and not in cache mount
RUN --mount=type=cache,target=/.kagent/cache,uid=1001,gid=1001                \
    echo "Creating virtual environment and installing dependencies..."        \
    && uv venv --python=python$TOOLS_PYTHON_VERSION                           \
    && uv lock && uv sync --package kagent-adk                                \
    && uv cache prune                                                         \
    && echo "Installation complete."

# Create a separate venv for bash tool commands (sandbox environment)
# This venv does not have pip installed
RUN --mount=type=cache,target=/.kagent/cache,uid=1001,gid=1001                \
    echo "Creating bash tool sandbox environment..."                          \
    && mkdir -p /.kagent/sandbox-venv                                         \
    && uv venv --python=python$TOOLS_PYTHON_VERSION /.kagent/sandbox-venv     \
    && echo "Bash tool sandbox environment created."

ENV PATH="/.kagent/.venv/bin:$PATH"
ENV UV_PROJECT_ENVIRONMENT=/app/.venv
ENV BASH_VENV_PATH=/.kagent/sandbox-venv
ENV VIRTUAL_ENV=/.kagent/.venv

WORKDIR /app

ENTRYPOINT ["kagent-adk", "static", "--host", "0.0.0.0", "--port", "8080"]

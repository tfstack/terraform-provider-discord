# Terraform Provider Discord Development Container
FROM docker.io/golang:1.24-bullseye

# [Option] Install zsh
ARG INSTALL_ZSH="true"
# [Option] Upgrade OS packages to their latest versions
ARG UPGRADE_PACKAGES="true"

# Install needed packages and setup non-root user
ARG USERNAME=vscode
ARG USER_UID=1000
ARG USER_GID=$USER_UID

COPY scripts/*.sh /tmp/scripts/
COPY bashrc /tmp/bashrc
RUN apt-get clean \
    && rm -rf /var/lib/apt/lists/* \
    && echo "Acquire::Check-Valid-Until false;" > /etc/apt/apt.conf.d/99no-check-valid-until \
    && (apt-get update -y --allow-releaseinfo-change 2>&1 || true) \
    && apt-get install -y --allow-unauthenticated ca-certificates gnupg lsb-release || true \
    && apt-get update -y --allow-releaseinfo-change \
    && /bin/bash /tmp/scripts/common-debian.sh "${INSTALL_ZSH}" "${USERNAME}" "${USER_UID}" "${USER_GID}" "${UPGRADE_PACKAGES}" "true" "true" \
    # Install additional development tools
    && apt-get install -y \
        git \
        curl \
        wget \
        make \
        ca-certificates \
        gnupg \
        lsb-release \
        software-properties-common \
        apt-transport-https \
        bash-completion \
    # Install golangci-lint (latest)
    && curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin \
    # Install goimports (latest - GOTOOLCHAIN=auto will download compatible Go toolchain if needed)
    && GOTOOLCHAIN=auto go install golang.org/x/tools/cmd/goimports@latest \
    # Install gopls (latest - GOTOOLCHAIN=auto will download compatible Go toolchain if needed)
    && GOTOOLCHAIN=auto go install golang.org/x/tools/gopls@latest \
    # Configure bash history and completion for root
    && cat /tmp/bashrc >> /root/.bashrc \
    # Configure bash history and completion for vscode user if it exists
    && if [ -d /home/${USERNAME} ]; then cat /tmp/bashrc >> /home/${USERNAME}/.bashrc; fi \
    # Also add .env loading to system-wide bashrc for VS Code terminals
    && echo "" >> /etc/bash.bashrc \
    && echo "# Auto-load .env file if it exists (for Discord bot token and Terraform variables)" >> /etc/bash.bashrc \
    && echo "if [ -f /workspaces/terraform-provider-discord/.env ]; then" >> /etc/bash.bashrc \
    && echo "    set -a" >> /etc/bash.bashrc \
    && echo "    source /workspaces/terraform-provider-discord/.env" >> /etc/bash.bashrc \
    && echo "    set +a" >> /etc/bash.bashrc \
    && echo "elif [ -f /workspace/.env ]; then" >> /etc/bash.bashrc \
    && echo "    set -a" >> /etc/bash.bashrc \
    && echo "    source /workspace/.env" >> /etc/bash.bashrc \
    && echo "    set +a" >> /etc/bash.bashrc \
    && echo "fi" >> /etc/bash.bashrc \
    # Clean up
    && apt-get autoremove -y && apt-get clean -y && rm -rf /var/lib/apt/lists/* /tmp/scripts/ /tmp/bashrc

# Set up Go environment
ENV GOPATH=/go
ENV PATH=$GOPATH/bin:$PATH

# Keep as root user to avoid permission issues
# USER ${USERNAME}

# Set working directory
WORKDIR /workspace

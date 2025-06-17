#!/bin/bash
set -e

# Download golang-migrate binary if not present
if ! command -v migrate &>/dev/null; then
    echo "Downloading golang-migrate binary..."
    ARCH=$(uname -m)
    OS=$(uname | tr '[:upper:]' '[:lower:]')
    case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64 | arm64) ARCH="arm64" ;;
    *)
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
    esac
    VERSION=$(curl -s https://api.github.com/repos/golang-migrate/migrate/releases/latest | grep tag_name | cut -d '"' -f 4)
    URL="https://github.com/golang-migrate/migrate/releases/download/${VERSION}/migrate.${OS}-${ARCH}.tar.gz"
    TMP_DIR=$(mktemp -d)
    curl -L "$URL" -o "$TMP_DIR/migrate.tar.gz"
    tar -xzf "$TMP_DIR/migrate.tar.gz" -C "$TMP_DIR"
    install "$TMP_DIR/migrate" "$HOME/.local/bin/migrate"
    export PATH="$HOME/.local/bin:$PATH"
    rm -rf "$TMP_DIR"
fi

# Run migrations (migrations directory is always 'migrations')
echo "Running migrations..."
migrate -database "$POSTGRESQL_DB_CONNECT_STRING" -path migrations up

echo "Migration completed."

#!/bin/bash
set -e

echo "📦 Instalando k-flow-spec..."

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo "Arquitectura no soportada: $ARCH"; exit 1 ;;
esac

# Check for existing package managers
if command -v brew &> /dev/null && [[ "$OS" == "darwin" ]]; then
    echo "🍺 Instalando via Homebrew..."
    echo "  Homebrew tap no disponible. Usando Go..."
    go install github.com/AlvaroMrJack/k-flow-spec/cmd/kfs@latest
elif command -v go &> /dev/null; then
    echo "🔧 Instalando via Go..."
    go install github.com/AlvaroMrJack/k-flow-spec/cmd/kfs@latest
else
    echo "⚠️  No se encontró Homebrew ni Go."
    echo "   Instala Go desde https://go.dev/dl/ o usa Docker:"
    echo "   docker run ghcr.io/AlvaroMrJack/k-flow-spec:latest kfs run --mock"
    exit 1
fi

# Ensure GOPATH/bin is in PATH
GOPATH_BIN="$(go env GOPATH)/bin"
if [[ ":$PATH:" != *":$GOPATH_BIN:"* ]]; then
    SHELL_RC="$HOME/.zshrc"
    [[ "$SHELL" == */bash* ]] && SHELL_RC="$HOME/.bashrc"
    echo "" >> "$SHELL_RC"
    echo "# k-flow-spec" >> "$SHELL_RC"
    echo "export PATH=\"\$PATH:$GOPATH_BIN\"" >> "$SHELL_RC"
    echo "✅ $GOPATH_BIN añadido a PATH en $SHELL_RC"
    echo "   Recarga: source $SHELL_RC"
    export PATH="$PATH:$GOPATH_BIN"
fi

echo ""
echo "✅ k-flow-spec instalado correctamente!"
echo ""
echo "Para empezar:"
echo "  cd tu-proyecto/"
echo "  kfs init"
echo "  kfs run --mock"

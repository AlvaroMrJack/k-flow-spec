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
    go install github.com/AlvaroMrJack/k-flow-spec@latest
elif command -v go &> /dev/null; then
    echo "🔧 Instalando via Go..."
    go install github.com/AlvaroMrJack/k-flow-spec@latest
else
    echo "⚠️  No se encontró Homebrew ni Go."
    echo "   Instala Go desde https://go.dev/dl/ o usa Docker:"
    echo "   docker run ghcr.io/AlvaroMrJack/k-flow-spec:latest kfs run --mock"
    exit 1
fi

echo ""
echo "✅ k-flow-spec instalado correctamente!"
echo ""
echo "Para empezar:"
echo "  cd tu-proyecto/"
echo "  kfs init"
echo "  kfs run --mock"

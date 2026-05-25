#!/bin/bash
set -e

echo "📦 Instalando k-flow-spec..."

# Install via Go (Homebrew tap coming soon)
if ! command -v go &> /dev/null; then
    echo "⚠️  Go no está instalado."
    echo "   Instálalo desde https://go.dev/dl/ o usa Docker:"
    echo "   docker run ghcr.io/AlvaroMrJack/k-flow-spec:latest kfs run --mock"
    exit 1
fi

echo "🔧 Instalando via Go..."
go install github.com/AlvaroMrJack/k-flow-spec/cmd/kfs@latest

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
echo "  kfs init --configure   # Crea proyecto + asistente setup"
echo "  kfs generate -i        # Genera specs interactivamente"
echo "  kfs run --mock         # Ejecuta pruebas en modo simulado"

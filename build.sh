#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FRONT_DIR="$ROOT_DIR/front"
STATIC_DIR="$ROOT_DIR/cmd/dilmeterapi/static"
BUILD_DIR="$ROOT_DIR/build"

GOOS_VALUE="${GOOS:-$(go env GOOS)}"
GOARCH_VALUE="${GOARCH:-$(go env GOARCH)}"
OUTPUT_NAME="${OUTPUT_NAME:-Midir-${GOOS_VALUE}-${GOARCH_VALUE}}"
if [[ "$GOOS_VALUE" == "windows" ]]; then
  OUTPUT_NAME="${OUTPUT_NAME%.exe}.exe"
fi

echo "[1/5] Installing frontend dependencies..."
cd "$FRONT_DIR"
npm install

echo "[2/5] Building frontend..."
npm run build

echo "[3/5] Moving static files to backend..."
rm -rf "$STATIC_DIR"
mkdir -p "$STATIC_DIR"
cp -R "$FRONT_DIR/dist/"* "$STATIC_DIR/"

echo "[4/5] Tidying Go modules..."
cd "$ROOT_DIR"
go mod tidy

echo "[5/5] Building Go executable for ${GOOS_VALUE}/${GOARCH_VALUE}..."
mkdir -p "$BUILD_DIR"
GOOS="$GOOS_VALUE" GOARCH="$GOARCH_VALUE" go build -ldflags="-s -w" -trimpath -v -o "$BUILD_DIR/$OUTPUT_NAME" ./cmd/dilmeterapi

echo
echo "Build complete: $BUILD_DIR/$OUTPUT_NAME"

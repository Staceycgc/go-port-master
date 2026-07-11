#!/usr/bin/env bash
set -eo pipefail

ROOT="${GITHUB_WORKSPACE:-$(cd "$(dirname "$0")/.." && pwd)}"
VERSION="${GITHUB_REF_NAME#v}"
DIST="$ROOT/dist/release"

if [ -z "$VERSION" ] || [ "$VERSION" = "$GITHUB_REF_NAME" ]; then
  echo "Unable to resolve release version from GITHUB_REF_NAME=${GITHUB_REF_NAME:-<empty>}"
  exit 1
fi

mkdir -p "$DIST"
cd "$ROOT/backend"
go mod download

build_one() {
  local goos="$1"
  local goarch="$2"
  local ext="$3"
  local base="port-master-${VERSION}-${goos}-${goarch}"
  local out="$DIST/${base}${ext}"
  local stage="$DIST/${base}"
  local archive=""

  echo "==> Building ${base}"
  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 \
    go build -ldflags="-s -w" -o "$out" ./cmd/port-master

  if [ ! -f "$out" ]; then
    echo "Build output missing: $out"
    exit 1
  fi

  rm -rf "$stage"
  mkdir -p "$stage"

  if [ "$goos" = "windows" ]; then
    cp "$out" "$stage/port-master${ext}"
    archive="$DIST/${base}.zip"
  else
    cp "$out" "$stage/port-master"
    archive="$DIST/${base}.tar.gz"
  fi

  cp "$ROOT/README.md" "$stage/"

  if [ "$goos" = "windows" ]; then
    (cd "$stage" && zip -r "$archive" .)
  else
    tar -czf "$archive" -C "$stage" .
  fi

  rm -rf "$stage" "$out"

  if [ ! -f "$archive" ]; then
    echo "Archive missing: $archive"
    exit 1
  fi

  echo "==> Packaged $(basename "$archive")"
}

build_one windows amd64 .exe
build_one windows arm64 .exe
build_one linux amd64 ""
build_one linux arm64 ""
build_one darwin amd64 ""
build_one darwin arm64 ""

echo "==> Release artifacts"
ls -lh "$DIST"

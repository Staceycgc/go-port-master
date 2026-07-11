#!/usr/bin/env bash
set -eo pipefail

ROOT="${GITHUB_WORKSPACE:-$(cd "$(dirname "$0")/.." && pwd)}"
VERSION="${GITHUB_REF_NAME#v}"
DIST="$ROOT/dist/release"

mkdir -p "$DIST"
cd "$ROOT/backend"

build_one() {
  local goos="$1"
  local goarch="$2"
  local ext="$3"
  local base="port-master-${VERSION}-${goos}-${goarch}"
  local out="$DIST/${base}${ext}"
  local stage="$DIST/${base}"

  echo "==> Building ${base}"
  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 \
    go build -ldflags="-s -w" -o "$out" ./cmd/port-master

  rm -rf "$stage"
  mkdir -p "$stage"

  if [ "$goos" = "windows" ]; then
    cp "$out" "$stage/port-master${ext}"
  else
    cp "$out" "$stage/port-master"
  fi

  cp "$ROOT/README.md" "$stage/"

  if [ "$goos" = "windows" ]; then
    python3 - "$stage" "$DIST/${base}.zip" <<'PY'
import os
import sys
import zipfile

stage, archive = sys.argv[1], sys.argv[2]
with zipfile.ZipFile(archive, "w", zipfile.ZIP_DEFLATED) as zf:
    for root, _, files in os.walk(stage):
        for name in files:
            path = os.path.join(root, name)
            zf.write(path, os.path.relpath(path, stage))
PY
  else
    tar -czf "$DIST/${base}.tar.gz" -C "$stage" .
  fi

  rm -rf "$stage" "$out"
  ls -lh "$DIST/${base}".*
}

build_one windows amd64 .exe
build_one windows arm64 .exe
build_one linux amd64 ""
build_one linux arm64 ""
build_one darwin amd64 ""
build_one darwin arm64 ""

echo "==> Release artifacts:"
ls -lh "$DIST"

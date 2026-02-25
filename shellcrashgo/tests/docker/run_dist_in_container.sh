#!/usr/bin/env bash

set -euo pipefail

# One-click helper for developers:
# - Ensures dist/shellcrashgo-linux-amd64 exists (runs build_linux_package.sh if needed)
# - Starts a simple Linux container
# - Copies the dist tree into /app inside the container
# - Leaves the container running so you can exec into it

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# ROOT_DIR 指向 shellcrashgo 根目录
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

DIST_ROOT="${DIST_ROOT:-"$ROOT_DIR/dist/shellcrashgo-linux-amd64"}"
IMAGE="${IMAGE:-debian:stable-slim}"
CONTAINER_NAME="${CONTAINER_NAME:-shellcrashgo-dist-test}"

echo "==> Using DIST_ROOT:       $DIST_ROOT"
echo "==> Using base IMAGE:      $IMAGE"
echo "==> Using CONTAINER_NAME:  $CONTAINER_NAME"
echo

if [ ! -d "$DIST_ROOT" ]; then
  echo "dist directory not found at $DIST_ROOT"
  echo "Running build_linux_package.sh to create it..."
  (
    cd "$ROOT_DIR"
    ./build_linux_package.sh
  )
  echo
fi

if [ ! -d "$DIST_ROOT" ]; then
  echo "ERROR: dist directory still not found at $DIST_ROOT after build." >&2
  exit 1
fi

echo "==> Preparing container $CONTAINER_NAME (image: $IMAGE)"
docker rm -f "$CONTAINER_NAME" >/dev/null 2>&1 || true

docker pull "$IMAGE"

CID="$(docker create --name "$CONTAINER_NAME" -w /app "$IMAGE" tail -f /dev/null)"
echo "Created container with ID: $CID"

echo "==> Copying dist contents into /app inside container..."
docker cp "$DIST_ROOT/." "$CONTAINER_NAME:/app"

echo "==> Starting container..."
docker start "$CONTAINER_NAME" >/dev/null

echo
echo "Container '$CONTAINER_NAME' is running with ShellCrash Go dist at /app."
echo
echo "To open an interactive shell inside the container, run (one of):"
echo "  docker exec -it $CONTAINER_NAME bash"
echo "  docker exec -it $CONTAINER_NAME sh"
echo
echo "When you are done, you can remove the container with:"
echo "  docker rm -f $CONTAINER_NAME"


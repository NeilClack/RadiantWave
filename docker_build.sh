#!/bin/bash

set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "Usage: docker_build.sh [OPTIONS] RELEASE_TYPE"
  echo "  RELEASE_TYPE: dev | release"
  echo "  OPTIONS: --local (passed through to build.sh)"
  exit 1
fi

podman run -it --rm \
  -v "$(pwd)":/workspace:Z \
  -v ~/.ssh:/root/.ssh:ro,Z \
  -w /workspace \
  localhost/radiantwave-build ./build.sh "$@"

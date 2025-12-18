!#/bin/sh
podman run -it --rm \
  -v $(pwd):/workspace:Z \
  -v ~/.ssh:/root/.ssh:ro,Z \
  localhost/radiantwave-build ./build.sh dev

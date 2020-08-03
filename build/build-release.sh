#!/usr/bin/env bash
set -euo pipefail

BOOTKUBE_ROOT=$(git rev-parse --show-toplevel)
GOLANG_IMAGE=${GOLANG_IMAGE:-golang:1.13.11}
SRCDIR="/usr/src/bootkube"

docker run -v ${BOOTKUBE_ROOT}:${SRCDIR} ${GOLANG_IMAGE} make -C ${SRCDIR} release

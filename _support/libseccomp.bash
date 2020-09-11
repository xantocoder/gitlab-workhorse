#!/usr/bin/env bash

set -xeuo pipefail

SCMP_ARCHIVE="libseccomp-${SCMP_VERSION}.tar.gz"
SCMP_SRC_DIR="${SCMP_BUILD_DIR}/libseccomp-${SCMP_VERSION}"

if [[ ! -d "${SCMP_BUILD_DIR}" ]]; then
  echo "Cannot install libseccomp into ${SCMP_BUILD_DIR}; directory does not exist"
  exit 1
fi

pushd "${SCMP_BUILD_DIR}"
wget -qO- "https://github.com/seccomp/libseccomp/releases/download/v${SCMP_VERSION}/${SCMP_ARCHIVE}" | tar zvx
popd

pushd "${SCMP_SRC_DIR}"
./configure --prefix="${SCMP_BUILD_DIR}"

make -j$(nproc) install
popd

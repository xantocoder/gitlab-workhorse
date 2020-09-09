#!/usr/bin/env bash

set -xeuo pipefail

GM_ARCHIVE="GraphicsMagick-${GM_VERSION}.tar.gz"

if [[ ! -d "${GM_PREFIX}" ]]; then
  echo "Cannot install GraphicsMagick into ${GM_PREFIX}; directory does not exist"
  exit 1
fi

pushd "${BUILD_DIR}"
wget -qO- "https://sourceforge.net/projects/graphicsmagick/files/graphicsmagick/${GM_VERSION}/${GM_ARCHIVE}/download" | tar zvx
popd

pushd "${GM_SRC_DIR}"
./configure \
  --prefix="${GM_PREFIX}" \
  --disable-openmp \
  --without-xml \
  --without-ttf \
  --without-trio \
  --without-lcms2 \
  --without-wmf \
  --without-x \
  --without-dps \
  --without-bzlib \
  --without-tiff \
  --without-zstd \
  --without-jbig \
  --without-lzma \
  --without-threads \
  --without-magick-plus-plus

make -j$(nproc) install
popd

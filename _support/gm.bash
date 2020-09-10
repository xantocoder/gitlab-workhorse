#!/usr/bin/env bash

set -xeuo pipefail

GM_ARCHIVE="GraphicsMagick-${GM_VERSION}.tar.gz"
GM_SRC_DIR="${GM_BUILD_DIR}/GraphicsMagick-${GM_VERSION}"

if [[ ! -d "${GM_BUILD_DIR}" ]]; then
  echo "Cannot install GraphicsMagick into ${GM_BUILD_DIR}; directory does not exist"
  exit 1
fi

pushd "${GM_BUILD_DIR}"
wget -qO- "https://sourceforge.net/projects/graphicsmagick/files/graphicsmagick/${GM_VERSION}/${GM_ARCHIVE}/download" | tar zvx
popd

pushd "${GM_SRC_DIR}"
./configure \
  --prefix="${GM_BUILD_DIR}" \
  --disable-openmp \
  --with-perl=no \
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
  --without-jp2 \
  --without-lzma \
  --without-gslib \
  --without-fpx \
  --without-threads \
  --without-magick-plus-plus

make -j$(nproc) install
popd

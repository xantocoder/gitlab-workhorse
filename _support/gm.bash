#!/usr/bin/env bash

set -xeuo pipefail

GM_PREFIX="${BUILD_PATH}/pref"
GM_SRC="${BUILD_PATH}/GraphicsMagick-${GM_VERSION}"
GM_ARCHIVE="GraphicsMagick-${GM_VERSION}.tar.gz"

mkdir -p "${BUILD_PATH}"

if [[ ! -d "$GM_SRC" ]]; then
  pushd "${BUILD_PATH}"
  wget -qO- "https://sourceforge.net/projects/graphicsmagick/files/graphicsmagick/${GM_VERSION}/${GM_ARCHIVE}/download" | tar zvx
  popd
fi

if [[ ! -d "${GM_PREFIX}" ]]; then
  pushd "${GM_SRC}"
  ./configure --prefix="${GM_PREFIX}" \
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
fi

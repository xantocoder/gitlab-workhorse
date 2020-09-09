#!/usr/bin/env bash

set -xeuo pipefail

GM_PREFIX="${BUILD_PATH}/GraphicsMagick-Prefix"
GM_SRC="${BUILD_PATH}/GraphicsMagick-${GM_VERSION}"
GM_ARCHIVE="GraphicsMagick-${GM_VERSION}.tar.xz"

mkdir -p "${BUILD_PATH}"

if [[ ! -d "$GM_SRC" ]]; then
  pushd "${BUILD_PATH}"
  wget -qO- "https://deac-ams.dl.sourceforge.net/project/graphicsmagick/graphicsmagick/${GM_VERSION}/${GM_ARCHIVE}" | tar Jvx
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

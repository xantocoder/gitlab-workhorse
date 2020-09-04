#!/bin/bash

export GM_VERSION="1.3.35"

set -xeo pipefail

BUILD_PATH="$PWD/_build"
GM_PREFIX="$PWD/_build/GraphicsMagick-Prefix"
GM_SRC="$BUILD_PATH/GraphicsMagick-$GM_VERSION"

if [[ ! -d "$GM_SRC" ]]; then
  pushd "$BUILD_PATH"
  wget "https://deac-ams.dl.sourceforge.net/project/graphicsmagick/graphicsmagick/$GM_VERSION/GraphicsMagick-$GM_VERSION.tar.xz"
  tar -Jxf "GraphicsMagick-$GM_VERSION.tar.xz"
  rm "GraphicsMagick-$GM_VERSION.tar.xz"
  popd
fi

if [[ ! -d "$GM_PREFIX" ]]; then
  pushd "$GM_SRC"
  ./configure --prefix="$GM_PREFIX" \
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

export PKG_CONFIG_PATH="$GM_PREFIX/lib/pkgconfig:$PKG_CONFIG_PATH"

go clean --cache
go build -tags resizer_static_build ./cmd/gitlab-resize-image
ldd gitlab-resize-image

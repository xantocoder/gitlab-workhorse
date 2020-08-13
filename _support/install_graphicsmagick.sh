#!/usr/bin/env sh 
set -ex

gm_version="${GM_VERSION:-1.3.34}"

original_dir=$(pwd)
build_dir=$(mktemp -d)
trap "cd ${original_dir}; rm -rf ${build_dir}" EXIT

cd "${build_dir}"

curl -L "ftp://ftp.graphicsmagick.org/pub/GraphicsMagick/1.3/GraphicsMagick-${gm_version}.tar.gz" | tar xvz

cd "GraphicsMagick-${gm_version}"

./configure \
  --prefix="${build_dir}" \
  --disable-installed \
  --enable-shared=no \
  --enable-static=yes \
  --disable-openmp \
  --without-magick-plus-plus \
  --with-perl=no \
  --without-bzlib \
  --without-dps \
  --without-fpx \
  --without-gslib \
  --without-jbig \
  --without-webp \
  --without-jp2 \
  --without-lcms2 \
  --without-trio \
  --without-ttf \
  --without-umem \
  --without-wmf \
  --without-xml \
  --without-x \
  --with-tiff=yes \
  --with-lzma=yes \
  --with-jpeg=yes \
  --with-zlib=yes \
  --with-png=yes

make

./utilities/gm version

cp ./utilities/gm  "${original_dir}/vendor/gm"

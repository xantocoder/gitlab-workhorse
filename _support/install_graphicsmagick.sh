#!/usr/bin/env sh 
set -ex

original_dir=$(pwd)
export mydir=$(mktemp -d)
trap "cd $original_dir; rm -rf ${mydir}" EXIT
export CPPFLAGS="-I$mydir/include"
export LDFLAGS="-L$mydir/lib"
cd $mydir

# download sources
curl -L ftp://ftp.graphicsmagick.org/pub/GraphicsMagick/1.3/GraphicsMagick-1.3.34.tar.gz | tar xvz

# zlib needed for png
curl -L ftp://ftp.graphicsmagick.org/pub/GraphicsMagick/delegates/zlib-1.2.11.tar.gz | tar xvz
curl -L ftp://ftp.graphicsmagick.org/pub/GraphicsMagick/delegates/libpng-1.6.37.tar.gz | tar xvz
curl -L ftp://ftp.graphicsmagick.org/pub/GraphicsMagick/delegates/libwebp-1.0.0.tar.gz | tar xvz
curl -L ftp://ftp.graphicsmagick.org/pub/GraphicsMagick/delegates/jpegsrc.v6b2.tar.gz | tar xvz

# use `less zlib-<TAB>/configure` to discover configure options
(cd zlib-*           && ./configure --static                             --prefix=$mydir && make install)
(cd libpng-*         && ./configure --disable-shared                     --prefix=$mydir && make install)
(cd libwebp-*        && ./configure --disable-shared --enable-libwebpmux --prefix=$mydir && make install)
(cd jpeg-*           && ./configure --disable-shared                     --prefix=$mydir && make install)
(cd GraphicsMagick-* && ./configure --disable-installed                  --prefix=$mydir && make install)
./bin/gm version
cp ./bin/gm  "${original_dir}/vendor"

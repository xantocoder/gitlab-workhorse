#!/usr/bin/env sh 
set -ex

original_dir=$(pwd)
export mydir=$(mktemp -d)
trap "cd $original_dir; rm -rf ${mydir}" EXIT
export CPPFLAGS="-I$mydir/include"
export LDFLAGS="-L$mydir/lib"
cd $mydir

# download sources
curl -L http://ftp.icm.edu.pl/pub/unix/graphics/GraphicsMagick/1.3/GraphicsMagick-1.3.31.tar.gz | tar xvz
# zlib needed for png
curl -L http://ftp.icm.edu.pl/pub/unix/graphics/GraphicsMagick/delegates/zlib-1.2.11.tar.gz | tar xvz
curl -L http://ftp.icm.edu.pl/pub/unix/graphics/GraphicsMagick/delegates/libpng-1.6.28.tar.gz | tar xvz
curl -L http://ftp.icm.edu.pl/pub/unix/graphics/GraphicsMagick/delegates/libwebp-1.0.0.tar.gz | tar xvz
curl -L http://ftp.icm.edu.pl/pub/unix/graphics/GraphicsMagick/delegates/jpegsrc.v6b2.tar.gz | tar xvz

# use `less zlib-<TAB>/configure` to discover configure options
(cd zlib-*           && ./configure --static                             --prefix=$mydir && make install)
(cd libpng-*         && ./configure --disable-shared                     --prefix=$mydir && make install)
(cd libwebp-*        && ./configure --disable-shared --enable-libwebpmux --prefix=$mydir && make install)
(cd jpeg-*           && ./configure --disable-shared                     --prefix=$mydir && make install)
(cd GraphicsMagick-* && ./configure --disable-installed                  --prefix=$mydir && make install)
./bin/gm version
cp ./bin/gm  "${original_dir}/vendor"

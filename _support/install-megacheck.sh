#!/bin/sh
set -e
set -x

REPO=https://github.com/dominikh/go-tools.git
PKG_BASE=honnef.co/go/tools
PKG="${PKG_BASE}/cmd/megacheck"
INSTALL_DIR="${GOPATH}/src/${PKG_BASE}"

rm -rf -- "${INSTALL_DIR}"
mkdir -p "${INSTALL_DIR}"
git clone -b 2019.1 "${REPO}" "${INSTALL_DIR}"
go get -v "${PKG}"

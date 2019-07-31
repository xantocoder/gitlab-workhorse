#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

version="${1}"
location="${2}"

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

cd "$SCRIPT_DIR/.."
rm -rf  _build/staticcheck_tmp
mkdir -p _build/staticcheck_tmp
cd _build/staticcheck_tmp

case "$(uname -s)" in
    Linux*)     goos=linux;;
    Darwin*)    goos=darwin;;
    *) echo "Unknown operation system: $(uname -s)"; exit 1;;
esac

case "$(uname -m)" in
    i386)     goarch=386;;
    x86_64)    goarch=amd64;;
    *) echo "Unknown architecture: $(uname -m)"; exit 1;;
esac

archive_file="staticcheck_${goos}_${goarch}.tar.gz"
checksum_file="${archive_file}.sha256"

base_url="https://github.com/dominikh/go-tools/releases/download/${version}"

curl --fail --location -o "${archive_file}" "$base_url/${archive_file}"
curl --fail --location -o "${checksum_file}" "$base_url/${checksum_file}"

shasum -a 256 -c ${checksum_file}

tar -xzf ${archive_file}

chmod 755 staticcheck/staticcheck
mv staticcheck/staticcheck "${location}"

cd "$SCRIPT_DIR/.."
rm -rf  _build/staticcheck_tmp


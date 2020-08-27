#!/bin/sh

set -e

target=${CI_MERGE_REQUEST_TARGET_BRANCH_NAME:-master}

if echo "$CI_MERGE_REQUEST_TITLE" | grep 'NO CHANGELOG$'; then
    echo "Changelog not needed"

    exit 0
fi

if git diff --name-only "origin/$target" -- changelogs/ | wc -l | grep -E -v "\b0\b" > /dev/null; then
    echo "Changelog included"
else
    echo "Please add a changelog running '_support/changelog'"
    echo "or disable this check adding 'NO CHANGELOG' at the end of the merge request title"
    echo "/title $CI_MERGE_REQUEST_TITLE NO CHANGELOG"

    exit 1
fi


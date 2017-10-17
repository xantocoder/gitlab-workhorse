#!/bin/sh

git grep 'context.\(Background\|TODO\)' | grep -v -e '^[^:]*_test.go:' -e '^vendor/' -e '^_support/' | awk '{
  print "Found disallowed use of context.Background or TODO"
  print
  exit 1
}'
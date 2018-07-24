#!/bin/sh

export GOPATH="$(mktemp -d "/tmp/dep.XXXXXXXX")"
trap "rm -rf \"${GOPATH}\"" EXIT HUP INT

SRCDIR="${GOPATH}/src/github.com/EdSchouten/bazel-buildbarn"
mkdir -p "$(dirname "${SRCDIR}")"
ln -s "$(pwd)" "${SRCDIR}"
(cd "${SRCDIR}"; dep "$@")

rm -rf vendor
bazel run //:gazelle -- update-repos -from_file=Gopkg.lock
bazel run //:gazelle

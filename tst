#! /bin/sh

TEST="$1"
shift

GOPATH=$PWD go test -v -c -gcflags "-N -l" -o "bin/$TEST" "$TEST"
exec gdb -tui -d "$GOROOT" "$@" bin/$TEST

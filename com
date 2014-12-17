#! /bin/sh
rm -f bin/testnotify
GOPATH=$PWD go install -compiler gccgo -gccgoflags '-g' testnotify
# GOPATH=$PWD go install -compiler gc -gcflags '-N -l' testnotify

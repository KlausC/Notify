#! /bin/sh
GOPATH=$PWD go install -compiler gccgo -gccgoflags '-g' testnotify

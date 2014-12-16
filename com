#! /bin/sh
go install -compiler gccgo -gccgoflags '-g' fswatch/testnotify

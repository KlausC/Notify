package filesync

import (
	"testing"
	"path/filepath"
	"os"	
)

func TestMain(m *testing.M) {
	
	os.Chdir("../..")
	os.Exit(m.Run())
}


func TestCreate(t *testing.T) {
	T := "Create"
	dir := "../a/b/c/../d"
	fs, err := Create(dir)
	if err != nil {
		t.Error(T, err, dir) 
	}
	if fs == nil {
		t.Error(T, "fs nil", dir)
	}
	idir, err := filepath.Abs(dir)
	if fs.baseDir != idir {
		t.Error(T, "internal", idir, "!=", fs.baseDir)
	}
	return
}

func TestSyncName(t *testing.T) {
	T:= "SyncName"
	dir := "../../../../../synchronized/"
	fs, err := Create(dir)
	if err != nil {
		t.Error(T, err)
	}
	orig := "/home/user/xyz/.."
	exp := "/home/synchronized/home/user"
	target := fs.SyncName(orig)
	if target != exp {
		t.Error(T, dir, orig, exp, target)
	}
	return
}

func TestRenameOrig1(t *testing.T) {
	T := "RenameOrig"
	orig := "/a/b/c/x/y/z"
	olddir := "/a/b/c"
	newdir := "/A/"
	expected := "/A/x/y/z"
	target := RenameOrig(newdir, olddir, orig)
	
	if target != expected {
		t.Error(T, "expected", expected, "!=", target)
	}
	return
}

func TestRenameOrig2(t *testing.T) {
	T := "RenameOrig"
	orig := "/a/b/x/y/z"
	olddir := "/a/b/c"
	newdir := "/A/"
	expected := orig
	target := RenameOrig(newdir, olddir, orig)
	
	if target != expected {
		t.Error(T, "expected", expected, "!=", target)
	}
	return
}


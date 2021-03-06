package filesync

import (
	"path/filepath"
	"strings"
)

type FileSync struct {
	baseDir string
}

func NewFileSync(pa string) (fs *FileSync) {
	pabs, _ := filepath.Abs(filepath.Clean(pa))
	fs = &FileSync{baseDir: pabs}
	return
}


func (fs *FileSync) SyncName(orig string) (sync string) {
	if orig == "" {
		return
	}
	return filepath.Join(fs.baseDir, orig)
}

// if orig is starting with olddir, replace oldir part with newdir
func RenameOrig(newdir, olddir, orig string) (renamed string) {
	renamed = orig
	if orig == "" {
		return
	}
	olddira, err := filepath.Abs(olddir)
	if err != nil {
		return
	}
	origa, err := filepath.Abs(orig)
	if err != nil {
		return
	}
	if !strings.HasPrefix(origa, olddira) {
		return
	}
	rel, err := filepath.Rel(olddira, origa)
	if err != nil {
		return
	}
	newdira, err := filepath.Abs(newdir)
	if err != nil {
		return
	}
	renamed = filepath.Join(newdira, rel)
	return
}

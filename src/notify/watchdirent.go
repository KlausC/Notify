package notify

import (
	"path"
)


/*
	WatchDirent represents a directory entry internally.
	
*/
type WatchDirent struct {
	wd       uint32			// watch descriptor if this is a directory
	name     string			// name within parent directory (NAME_MAX)
	parent   *WatchDirent	// pointer to parent directory
	next     *WatchDirent	// pointer to next file with same inode - nil for directory
	statid   *Statid		// pointer to file status information (per inode)
	cookie   uint32			// transiently used between move-to and moved-from events
	elements map[string]*WatchDirent	// collection of all directory elements for directory
}

func (wde *WatchDirent) Cleanup() {
	wde.name = ""
	wde.parent = nil
	wde.elements = nil
}

// Cookie searches first cookie in hierarchy
func (wde *WatchDirent) Cookie() uint32 {
	if wde.parent == nil || wde.cookie != 0 {
		return wde.cookie
	}
	return wde.parent.Cookie()
}

// Path constructs complete path of hierarchy
func (wde *WatchDirent) Path() (pa string) {
	pa = wde.name
	if wde.parent == nil {
		return
	}
	dir := wde.parent.Path()
	if len(dir) > 0 {
		pa = path.Join(dir, pa)
	}
	return
}

func (wde *WatchDirent) Path2(name string) (pa string) {
	dir := wde.Path()
	pa = name
	if len(dir) > 0 {
		pa = path.Join(dir, name)
	}
	return
}

// createWatchDirent constructor
func createWatchDirent(parent *WatchDirent, name string, isdir bool) (wdenew *WatchDirent) {
	wdenew = &WatchDirent{name: name, parent: parent}
	if isdir {
		wdenew.elements = make(map[string]*WatchDirent)
	}
	return
}

/*
	Dequeue removes this entry from the linked list
*/
func (wde *WatchDirent) Dequeue() {
	previous := &wde.statid.first
	for next := *previous; next != nil; next = *previous {
		if next == wde {
			*previous = next.next
			next.next = nil
			break
		}
		previous = &next.next
	}
}
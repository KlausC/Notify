package notify

import (
	"bytes"
	"fmt"
	"os"
	"syscall"
	"path"
)

const NAME_MAX = 255

/*
NotifyCallbacks is a list of callback function provided by the user
*/
type NotifyCallbacks struct {
	Report  func(string, *Event)
	Created func(string, uint32)
	Deleted func(string, uint32)
	Changed func(string, uint32)
	Linked  func(string, string, uint32)
	Moved   func(string, string, uint32)
	Removed func(string, uint32)
	Attribute func(string, uint32)
}

/*
	WatchDirent represents a directory entry
*/
type WatchDirent struct {
	wd       uint32
	name     string
	parent   *WatchDirent
	next     *WatchDirent
	statid   *Statid
	cookie   uint32
	elements map[string]*WatchDirent
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

/* WT - Central watchtable object definition.
Note that dictionary objects are all included in this structure.
*/
type WT struct {
	data     map[uint32]*WatchDirent  // map of wd to watchDirents
	inodes   map[Stat_key]*Statid    // set of stat by inode
	excludes map[string]bool         // set of path names to be excluded
	moved    map[uint32]*WatchDirent // wachDirents moved away from dir
	reader	 EventReader			// Event reader
	root     WatchDirent             // directory entry containing all root paths
	ncb      *NotifyCallbacks        // functions to be called
}

// createWatchTable constructor
func createWatchTable(mask uint32) (wt *WT) {
	wt = &WT{}
	wt.reader.Init(mask)
	wt.data = make(map[uint32]*WatchDirent)
	wt.inodes = make(map[Stat_key]*Statid)
	wt.moved = make(map[uint32]*WatchDirent)
	wt.excludes = make(map[string]bool)
	wt.root = WatchDirent{elements: make(map[string]*WatchDirent)}
	return
}

/* destroy and free watchtable */
func (wt *WT) cleanup() {
	wt.reader.Close()
	wt.data = nil
	wt.inodes = nil
	wt.excludes = nil
	wt.moved = nil
	wt.root.Cleanup()
}

/* print error text to stdout and perform error return. */
func reporterror(err error, text string, ec int) {
	fmt.Println(text, err, ec)
	if ec != 0 {
		panic(ec)
	}
}

/* print error text to stdout and perform error return. */
func report(err error, a, b string, ec int) {
	text := fmt.Sprintf("%s(\"%s\")", a, b)
	reporterror(err, text, ec)
}

/*
 * Call the callback function for each directory entry.
 * The pseudo subdirectories "." and ".." are excluded.
*/
func (wt *WT) walkDirectory(wde *WatchDirent, action func(*WatchDirent, string, *WT)) (err error) {
	dir := wde.Path()
	file, err := os.Open(dir)
	if err != nil {
		report(err, "Open", dir, 0)
		return
	}
	fis, err := file.Readdirnames(0)
	if err != nil {
		report(err, "Readdirnames", dir, 0)
		return
	}
	for _, name := range fis {
		if name != "." && name != ".." {
			action(wde, name, wt)
		}
	}
	return
}

/*
 * Add a path to observed objects.
 * Dict<int, char*> stores the association from watch id to pathname
 * of observed object.
 * Events according to mask will be delivered.
 */
func (wt *WT) addWatch(wde *WatchDirent) {
	path := wde.Path()
	wd, err := wt.reader.addWatch(path)
	wde.wd = wd
	if err != nil {
		report(err, "inotifyAddWatch", path, 0)
		return
	}
	wt.data[wde.wd] = wde
	fmt.Printf("node+ %d %s\n", wd, path)
	return
}

/*
 * Remove individual watch descriptor.
 */
func (wt *WT) removeWatch(wde *WatchDirent) {
	wd := wde.wd
	if wd > 0 {
		fmt.Printf("node- %d %s\n", wd, wde.Path())
		err := wt.reader.removeWatch(wd)
		if err != nil {
			// report(err, "inotify_rm_watch", strconv.FormatInt(int64(wd), 10), 0)
		}
	}
	delete(wt.data, wd)
	return
}

/*
 * Add path name to the set of excluded path names.
 * Set<char*> excludes.
 */
func (wt *WT) addExclude(path string) {
	wt.excludes[path] = true
}

func (wt *WT) dequeueAndMaybeFreeStatus(wde *WatchDirent) {
	wde.Dequeue()
	if wde.statid.first == nil {
		delete(wt.inodes, key(&wde.statid.filestat))
	}
}

/*
 * Remove wde hierarchy from cookie dictionary and
 * remove corresponding watches.
 */
func (wt *WT) removeHierarchyRec(wde *WatchDirent) {
	if wde.wd > 0 {
		wt.removeWatch(wde)
	}
	if wde.elements != nil {
		for k, wdechild := range wde.elements {
			wt.removeHierarchyRec(wdechild)
			delete(wde.elements, k)
		}
		wde.elements = nil
	}
	wt.dequeueAndMaybeFreeStatus(wde)
	wt.destroyAndUnlink(wde)
}

func (wt *WT) removeHierarchy(wde *WatchDirent) {
	wt.removeHierarchyRec(wde)

	if wde.cookie != 0 {
		delete(wt.moved, wde.cookie)
	} else if wde.parent != nil && wde.parent.elements != nil {
		delete(wde.parent.elements, wde.name)
	}
}

const WATCHED = syscall.S_IFREG | syscall.S_IFDIR | syscall.S_IFLNK

/*
 * Insert all data for a newly detected file, directory, or soft link.
 * Set<stateid> inodes obtains a new or actualised entry for the inode.
 * Dict<char*,struct property> data2 obtains new entry for path name.
 * The pathname entries for the same inode maintain a linked list.
 */
func (wt *WT) statNewFile(wde *WatchDirent, name string) *WatchDirent {
	/* if wde is in moved directory */
	if wde.Cookie() != 0 {
		wt.removeWatch(wde)
		return nil
	}

	path := wde.Path2(name)
	statidBuffer := Statid{}

	if err := syscall.Lstat(path, &statidBuffer.filestat); err != nil {
		report(err, "statNewFilel.stat", path, 0)
		return nil
	}
	if statidBuffer.filestat.Mode&WATCHED != 0 {
		var savedfirst *WatchDirent = nil
		statkey := statidBuffer.key()
		statid, ok := wt.inodes[statkey]
		if ok {
			savedfirst = statid.first
			statid.filestat = statidBuffer.filestat
		} else {
			wt.inodes[statkey] = &statidBuffer
			statid = &statidBuffer
		}
		wdenew := createWatchDirent(wde, name, statid.filestat.Mode&syscall.S_IFDIR != 0)
		wdenew.statid = statid
		wdenew.next = savedfirst
		wde.elements[name] = wdenew
		statid.first = wdenew
		return wdenew
	}
	return nil
}

/*
 * Callback function for directory walk loop.
 * Build full path name,
 * If path is a directory, recursively walk this subdirectory and
 * finally add inotify watch to it.
 *
 */
func addWatches(wde *WatchDirent, name string, wt *WT) {
	wdenew := wt.statNewFile(wde, name)
	if wdenew != nil && wdenew.statid.filestat.Mode&syscall.S_IFDIR != 0 {
		if wt.walkDirectory(wdenew, addWatches) == nil {
			wt.addWatch(wdenew)
		}
	}
}

func addWatches2(wde *WatchDirent, name string, wt *WT) {
	wdenew := wt.statNewFile(wde, name)
	if wdenew == nil {
		return
	}
	wt.callback(wt.ncb.Created, &Event{}, wdenew)
	if wdenew != nil && wdenew.statid.filestat.Mode&syscall.S_IFDIR != 0 {
		if wt.walkDirectory(wdenew, addWatches2) == nil {
			wt.addWatch(wdenew)
		}
	}
}

func byteToString(b []byte, n uint32) string {

	leng := bytes.IndexByte(b[:n], byte(0))
	if leng < 0 {
		leng = int(n)
	}
	return string(b[:leng])
}

/*
 * Call the external callbacks.
 */
func (wt *WT) debug(event *Event) {
	name := event.Name
	buffer := fmt.Sprintf("debug: wd= %d name=\"%s\"", event.Wd, name)
	if wt.ncb != nil && wt.ncb.Report != nil {
		(wt.ncb.Report)(buffer, event)
	}
}

func (wt *WT) callback(cb func(string, uint32), event *Event, wde *WatchDirent) {

	if cb != nil {
		path := wde.Path()
		cb(path, event.Mask)
	}
}

func (wt *WT) callback2(cb func(string, string, uint32), event *Event, wde *WatchDirent, altpath string) {

	if cb != nil {
		path := wde.Path()
		cb(path, altpath, event.Mask)
	}
}

func (wt *WT) processSelf(event *Event, wde *WatchDirent) int {
	mask := event.Mask

	switch {
	case mask&syscall.IN_IGNORED != 0:
		path := wde.Path()
		fmt.Printf("node- %d %s\n", event.Wd, path)
		delete(wt.data, event.Wd)
	case mask&syscall.IN_MOVE_SELF != 0:
		// move-from is missing or not subfile of supervised directory */
		if wde.cookie > 0 || wde.parent.wd == 0 {
			wt.removeHierarchy(wde)
		}
	case mask&syscall.IN_DELETE_SELF != 0:
		if wde.parent.wd == 0 {
			wde.wd = 0
			wt.removeHierarchy(wde)
		}
	}

	return 0
}

func (wde *WatchDirent)child(event *Event) (wdenew *WatchDirent) {
	name := event.Name
	wdenew, ok := wde.elements[name]
	if !ok || wdenew == nil {
		report(nil, "missing element", wde.Path2(name), 64)
	}
	return
}

func (wt *WT) processCreate(event *Event, wde *WatchDirent) int {
	mask := event.Mask
	name := event.Name
	wdenew := wt.statNewFile(wde, name)
	if wdenew == nil {
		return 0
	}
	wt.callback(wt.ncb.Created, event, wdenew)
	if mask&syscall.IN_ISDIR != 0 {
		if wt.walkDirectory(wdenew, addWatches2) == nil {
			wt.addWatch(wdenew)
		}
		//D wt.printTable("p create")
	}
	return 0
}

func (wt *WT) processMovedFrom(event *Event, wdenew *WatchDirent) int {
	if wdenew == nil {
		return 0
	}
	wdenew.cookie = event.Cookie
	delete(wdenew.parent.elements, wdenew.name)
	wt.moved[wdenew.cookie] = wdenew
	wdenew.Dequeue()
	return 0
}

func (wt *WT) processMovedTo(event *Event, wde *WatchDirent) int {

	wdenew, ok := wt.moved[event.Cookie]
	if !ok {
		return wt.processCreate(event, wde)
	} else {
		oldpath := wdenew.Path()
		wdenew.cookie = 0
		wdenew.name = event.Name
		wdenew.parent = wde
		wde.elements[wdenew.name] = wdenew
		statid := wdenew.statid
		wdenew.next = statid.first
		statid.first = wdenew
		wt.callback2(wt.ncb.Moved, event, wdenew, oldpath)
	}
	return 0
}

func (wt *WT) destroyAndUnlink(wde *WatchDirent) {
	if wde.wd > 0 {
		delete(wt.data, wde.wd)
	}
	if wde.cookie > 0 {
		delete(wt.moved, wde.cookie)
	} else if wde.parent != nil && wde.parent.elements != nil && len(wde.name) > 0 {
		delete(wde.parent.elements, wde.name)
	}
}

func (wt *WT) processDelete(event *Event, wdenew *WatchDirent) int {

	if wdenew == nil {
		return 0
	}
	wt.callback(wt.ncb.Deleted, event, wdenew)
	wt.removeHierarchy(wdenew)
	delete(wdenew.parent.elements, wdenew.name)
	return 0
}

func (wt *WT) modifyComplete(event *Event, wde *WatchDirent) (res int) {
	if wde != nil && wde.statid.isChangeComplete() {
		wt.callback(wt.ncb.Changed, event, wde)
		wde.statid.resetChanged()
	}
	return
}

func (wt *WT) attributeComplete(event *Event, wde *WatchDirent) (res int) {
	if wde != nil && wde.statid.isAttributeComplete() {
		wt.callback(wt.ncb.Attribute, event, wde)
		wde.statid.resetAttribute()
	}
	return
}

func (wt *WT) processModify(event *Event, wdenew *WatchDirent) (res int) {
	wdenew.statid.smask |= syscall.IN_MODIFY
	return
}
func (wt *WT) processClose(event *Event, wdenew *WatchDirent) (res int) {
	if wdenew.statid.smask & syscall.IN_MODIFY != 0 {
		wdenew.statid.smask |= syscall.IN_CLOSE_WRITE
		res = wt.modifyComplete(event, wdenew)
}
	return
}
func (wt *WT) processAttribute(event *Event, wdenew *WatchDirent) (res int) {
	wdenew.statid.smask |= syscall.IN_ATTRIB
	res = wt.attributeComplete(event, wdenew)
	return
}

func (wt *WT) processSubfile(event *Event, wde *WatchDirent) (res int) {
	mask := event.Mask
	switch {
	case mask&syscall.IN_CREATE != 0:
		res = wt.processCreate(event, wde)
	case mask&syscall.IN_MOVED_FROM != 0:
		res = wt.processMovedFrom(event, wde.child(event))
	case mask&syscall.IN_MOVED_TO != 0:
		res = wt.processMovedTo(event, wde)
	case mask&syscall.IN_DELETE != 0:
		res = wt.processDelete(event, wde.child(event))
	case mask&syscall.IN_MODIFY != 0:
		res = wt.processModify(event, wde.child(event))
	case mask&syscall.IN_CLOSE_WRITE != 0:
		res = wt.processClose(event, wde.child(event))
	case mask&syscall.IN_ATTRIB != 0:
		res = wt.processAttribute(event, wde.child(event))
	}
	//D fmt.Printf("%s%s %#x %#x \n", "subfile", wde.name,wde.statid.address(),  wde.statid.smask)
	return
}

/*
 * Process a single event.
 * The function returns positive integer return value to indicate the
 * processing loop to be stopped.
 * return 1: No more directories/files to be watched
 * return 2: For overflow of event queue
 * return 3: Event with zero watch descriptor
 * return 4: Registered path is NULL
 */
func (wt *WT) processEvent(event *Event) (res int) {

	name := event.Name
	mask := event.Mask
	wt.debug(event)
	if mask&syscall.IN_Q_OVERFLOW != 0 {
		report(nil, "processEvent", "IN_Q_OVERFLOW", 0)
		return 2
	}
	if event.Wd == 0 {
		report(nil, "processEvent", "wd == 0", 0)
		return 3
	}

	wde, ok := wt.data[event.Wd]
	if !ok {
		return 0 // silently ignore
	}
	if len(name) == 0 {
		res = wt.processSelf(event, wde)
	} else {
		res = wt.processSubfile(event, wde)
	}
	//D wt.printTable("after event")
	if res == 0 && len(wt.data) <= 0 {
		res = 1
	}
	return res
}

/*
 * Walk dictionary of registered path names.
 */
func (wt *WT) printWalk(action func(*WatchDirent), action2 func(*WatchDirent, int), text string) {

	fmt.Printf("%s:\n", text)
	for _, wde := range wt.data {
		action(wde)
	}
	wt.root.Walk(action2, 0)
}

/*
 * Callback function to print watched path names.
 */
func actionPrintWatchDescs(wde *WatchDirent) {
	if wde != nil {
		path := wde.Path()
		fmt.Printf("node: %d => %s\n", wde.wd, path)
	}
}

func actionPrintHierarchy(wde *WatchDirent, depth int) {
	if wde == nil || wde.statid == nil {
		return
	}
	if depth <= 0 {
		depth = 1
	}
	fmt.Printf("%*s%s %#x %#x (", (depth-1)*2, "", wde.name, wde.statid.address(), wde.statid.smask)
	for i, wden := 0, wde.statid.first; wden != nil && i < 100; i, wden = i+1, wden.next {
		path := wden.Path()
		fmt.Printf("%s ", path)
	}
	fmt.Printf(")\n")
}

func (wde *WatchDirent) Walk(action func(*WatchDirent, int), depth int) {
	if wde  != nil && wde.statid != nil {
		action(wde, depth)
	}
	for _, wdenew := range wde.elements {
		wdenew.Walk(action, depth+1)
	}
}

/*
 * Print table contents walking watchtable.
 */
func (wt *WT) printTable(text string) {
	wt.printWalk(actionPrintWatchDescs, actionPrintHierarchy, text)
}

/*
 * Setup processing.
 * Return initialised watchtable object on heap (to be freed at exit.
 */
func fillWatchTable(inv []string, exv []string, mask uint32, ncb *NotifyCallbacks) (wt *WT) {

	wt = createWatchTable(mask)
	if wt == nil {
		return
	}
	wt.ncb = ncb
	for _, pa := range inv {
		ppath := path.Join(pa)
		wde := wt.statNewFile(&wt.root, ppath)
		if wde != nil && wt.walkDirectory(wde, addWatches) == nil {
			wt.addWatch(wde)
		}
	}
	for _, pa := range exv {
		ppath := path.Join(pa)
		wt.addExclude(ppath)
	}
	wt.printTable("init watchtable")
	return wt
}

func (wt *WT) cleanupWatchtable() {

}

/*
 * Perform processing loop.
 */
func (wt *WT) internalProcessNotify() (stop int) {

	if len(wt.data) == 0 {
		stop = 1
	}
	for stop == 0 {
		ev, err := wt.reader.NextEvent()
		if err != nil {
			return 1
		}
		stop = wt.processEvent(ev)
	}

	wt.reader.Close()
	if stop <= 1 {
		return 0
	}
	return
}

/*
 * Initialise processing and perform processing loop.
 * Shutdown processing upon error or normal return.
 * Callback function called whenever notify event asks for special activity.
 */
func ProcessNotifyEvents(inv []string, exv []string, mask uint32, ncb *NotifyCallbacks) (res int) {

	wt := fillWatchTable(inv, exv, mask, ncb)
	if wt == nil {
		return 1
	}
	defer func() {
		err := recover()
		switch err := err.(type) {
		case int:
			res = err
		case nil:
			res = 0
		default:
			fmt.Println(err, "returning", 99)
			res = 99
		}
	}()

	return wt.internalProcessNotify()
}

func MaskToString(mask uint32) (s string) {
	names := []string{
		"ACCESS", "MODIFY", "ATTRIB", "CLOSE_WRITE", "CLOSE_NOWRITE",
		"OPEN", "MOVED_FROM", "MOVED_TO", "MOVE_SELF",
		"CREATE", "DELETE", "DELETE_SELF", "UNMOUNT", "Q_OVERFLOW", "IGNORED",
		"DIR",
	}
	masks := []uint32{
		syscall.IN_ACCESS, syscall.IN_MODIFY, syscall.IN_ATTRIB,
		syscall.IN_CLOSE_WRITE, syscall.IN_CLOSE_NOWRITE,
		syscall.IN_OPEN, syscall.IN_MOVED_FROM, syscall.IN_MOVED_TO,
		syscall.IN_MOVE_SELF, syscall.IN_CREATE, syscall.IN_DELETE,
		syscall.IN_DELETE_SELF, syscall.IN_UNMOUNT, syscall.IN_Q_OVERFLOW,
		syscall.IN_IGNORED, syscall.IN_ISDIR,
	}

	for i := 0; i < len(names); i++ {
		if mask&masks[i] != 0 {
			if len(s) > 0 {
				s += ","
			}
			s += names[i]
		}
	}
	return s
}

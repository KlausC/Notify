package notify

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

type Event struct {
	EventType    EventType
	IsDir        bool
	DataModified bool
	Path         string
	Path2        string
	Key          StatKey
}

/*
NotifyCallbacks is a list of callback function provided by the user
*/
type NotifyCallbacks struct {
	Init   InitCallback
	Report ReportCallback
	Event  EventCallback
}

type (
	InitCallback   func()
	ReportCallback func(string, *EventIntern)
	EventCallback  func(ev *Event)
)
type EventType uint8

const (
	CREATE    = EventType(1)
	DELETE    = EventType(2)
	MOVE      = EventType(3)
	LINK      = EventType(4)
	ATTRIBUTE = EventType(5)
	CHANGE    = EventType(6)
)

func (et EventType) String() (out string) {
	switch et {
	case CREATE:
		out = "CREATE"
	case DELETE:
		out = "DELETE"
	case MOVE:
		out = "MOVE"
	case LINK:
		out = "LINK"
	case ATTRIBUTE:
		out = "ATTRIBUTE"
	case CHANGE:
		out = "CHANGE"
	default:
		out = "NOP"
	}
	return
}

/* WT - Central watchtable object definition.
Note that dictionary objects are all included in this structure.
*/
type WT struct {
	data          map[uint32]*WatchDirent // map of wd to watchDirents
	inodes        map[StatKey]*Statid     // set of stat by inode
	excludes      map[string]bool         // set of path names to be excluded
	moved         map[uint32]*WatchDirent // wachDirents moved away from dir
	reader        EventReader             // Event reader
	root          WatchDirent             // directory entry containing all root paths
	ncb           *NotifyCallbacks        // functions to be called
	pendingCookie uint32                  // cookie form last movedFrom event
}

// createWatchTable constructor
func createWatchTable(mask uint32) (wt *WT) {
	wt = &WT{}
	wt.reader.Init(mask)
	wt.data = make(map[uint32]*WatchDirent)
	wt.inodes = make(map[StatKey]*Statid)
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
	//D fmt.Printf("node+ %d %s\n", wd, path)
	return
}

/*
 * Remove individual watch descriptor.
 */
func (wt *WT) removeWatch(wde *WatchDirent) {
	wd := wde.wd
	if wd > 0 {
		//D fmt.Printf("node- %d %s\n", wd, wde.Path())
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

// DequeueAndMaybeFreeStatus calls Dequeue on wde
// and deletes the Statid if that has no more reference in directory.
func (wt *WT) dequeueAndMaybeFreeStatus(wde *WatchDirent) {
	wde.Dequeue()
	if wde.statid.first == nil {
		delete(wt.inodes, wde.statid.key())
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

// unlink deletes wde from moved directory of from parent directory.
func (wt *WT) unlink(wde *WatchDirent) {
	if wde.cookie != 0 {
		delete(wt.moved, wde.cookie)
	} else if wde.parent != nil && wde.parent.elements != nil {
		delete(wde.parent.elements, wde.name)
	}
}

// removeHierarchy recursively removes all elements form wde.
// then deletes wde from moved or parent elements.
func (wt *WT) removeHierarchy(wde *WatchDirent) {
	wt.removeHierarchyRec(wde)
	wt.unlink(wde)
}

// WATCHED is binary mask for all file modes as observed in struct stat
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

	path := wde.Path(name)
	statidBuffer := Statid{}

	if err := syscall.Lstat(path, &statidBuffer.filestat); err != nil {
		report(err, "statNewFilel.stat", path, 0)
		return nil
	}
	if statidBuffer.filestat.Mode&WATCHED != 0 {
		var savedfirst *WatchDirent = nil
		statkey := statidBuffer.key()
		statid, ok := wt.inodes[statkey] // check if there is already and entry for this inode
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

// addWatches2 adds watches recursively after a directory has been created.
func addWatches2(wde *WatchDirent, name string, wt *WT) {
	wdenew := wt.statNewFile(wde, name)
	if wdenew == nil {
		return
	}
	isDir := wdenew.statid.filestat.Mode&syscall.S_IFDIR != 0
	wt.callback(CREATE, &EventIntern{}, wdenew, false)
	if wdenew != nil && isDir {
		if wt.walkDirectory(wdenew, addWatches2) == nil {
			wt.addWatch(wdenew)
		}
	}
}

/*
 * Call the external callbacks.
 */
func (wt *WT) debug(event *EventIntern) {
	name := event.Name
	buffer := fmt.Sprintf("debug: wd= %d name=\"%s\"", event.Wd, name)
	if wt.ncb != nil && wt.ncb.Report != nil {
		(wt.ncb.Report)(buffer, event)
	}
}

// callback calls a callback function with an additional string parameter
func (wt *WT) callback(et EventType, event *EventIntern, wde *WatchDirent, data bool, altpath ...string) {

	if wt.ncb != nil && wt.ncb.Event != nil {
		var ev Event
		ev.EventType = et
		ev.IsDir = wde.statid.filestat.Mode&syscall.S_IFDIR != 0
		ev.DataModified = data
		ev.Path = wde.Path()
		if len(altpath) > 0 {
			ev.Path2 = altpath[0]
		}
		ev.Key = wde.statid.key()
		wt.ncb.Event(&ev)
	}
}

// process the IN_..._SELF events (which have no Name in InotifyEvent).
func (wt *WT) processSelf(event *EventIntern, wde *WatchDirent) int {
	mask := event.Mask

	switch {
	case mask&syscall.IN_IGNORED != 0:
		//D fmt.Printf("node- %d %s\n", event.Wd, wde.Path())
		delete(wt.data, event.Wd)
	case mask&syscall.IN_MOVE_SELF != 0:
		// move-to is missing or not subfile of supervised directory */
		if wde.cookie > 0 || wde.parent.wd == 0 {
			wt.removeHierarchy(wde)
			event.Mask |= syscall.IN_ISDIR
			wt.pendingCookie = 0
			wt.callbackDelete(event, wde)
		}
	case mask&syscall.IN_DELETE_SELF != 0:
		if wde.parent.wd == 0 {
			wde.wd = 0
			wt.removeHierarchy(wde)
			event.Mask |= syscall.IN_ISDIR
			wt.callback(DELETE, event, wde, true)
		}
	case mask&syscall.IN_ATTRIB != 0:
		if wde.parent.wd == 0 {
			wde.wd = 0
			return wt.processAttribute(event, wde)
		}
	}

	return 0
}

// processCreate event
func (wt *WT) processCreate(event *EventIntern, wde *WatchDirent) int {
	mask := event.Mask
	name := event.Name
	wdenew := wt.statNewFile(wde, name)
	if wdenew == nil {
		return 0
	}
	createEvent := event.Mask&syscall.IN_CREATE != 0
	if wdenew.next != nil && createEvent {
		wt.callback(LINK, event, wdenew, false, wdenew.next.Path())
	} else {
		wt.callback(CREATE, event, wdenew, true)
	}
	if mask&syscall.IN_ISDIR != 0 {
		if wt.walkDirectory(wdenew, addWatches2) == nil {
			wt.addWatch(wdenew)
		}
		//D wt.printTable("p create")
	}
	return 0
}

// processMovedFrom event
func (wt *WT) processMovedFrom(event *EventIntern, wdenew *WatchDirent) int {
	if wdenew == nil {
		return 0
	}
	wdenew.cookie = event.Cookie
	wt.pendingCookie = wdenew.cookie
	delete(wdenew.parent.elements, wdenew.name)
	wt.moved[wdenew.cookie] = wdenew
	wdenew.Dequeue()
	return 0
}

// processMovedTo event
func (wt *WT) processMovedTo(event *EventIntern, wde *WatchDirent) int {

	wdenew, ok := wt.moved[event.Cookie]
	if !ok {
		// no corresponding movedFrom
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
		wt.callback(MOVE, event, wdenew, false, oldpath)
	}
	return 0
}

// destroyAndUnlink deletes this wde from all wt dictionaries.
func (wt *WT) destroyAndUnlink(wde *WatchDirent) {
	if wde.wd > 0 {
		delete(wt.data, wde.wd)
	}
	wt.unlink(wde)
}

// processDelete event
func (wt *WT) processDelete(event *EventIntern, wdenew *WatchDirent) int {

	if wdenew == nil {
		return 0
	}
	wt.callbackDelete(event, wdenew)
	wt.removeHierarchy(wdenew)
	delete(wdenew.parent.elements, wdenew.name)
	return 0
}

// call callback for delete event
// additional alternative path when file content preserved
func (wt *WT) callbackDelete(event *EventIntern, wde *WatchDirent) {
	alt := wde.Alternative()
	if alt == nil {
		wt.callback(DELETE, event, wde, true)
	} else {
		wt.callback(DELETE, event, wde, false, alt.Path())
	}
}

// modifyComplete is called after a file contents change is concluded.
func (wt *WT) modifyComplete(event *EventIntern, wde *WatchDirent) (res int) {
	if wde != nil && wde.statid.isChangeComplete() {
		wt.callback(CHANGE, event, wde, true)
		wde.statid.resetChanged()
	}
	return
}

// attributeComplete is called after each attribute change event
func (wt *WT) attributeComplete(event *EventIntern, wde *WatchDirent) (res int) {
	if wde != nil && wde.statid.isAttributeComplete() {
		wt.callback(ATTRIBUTE, event, wde, false)
		wde.statid.resetAttribute()
	}
	return
}

// processModify event - only smask bit is set
func (wt *WT) processModify(event *EventIntern, wdenew *WatchDirent) (res int) {
	wdenew.statid.smask |= syscall.IN_MODIFY
	return
}

// processClose Event - eventually conclude modification of file contents
func (wt *WT) processClose(event *EventIntern, wdenew *WatchDirent) (res int) {
	if wdenew.statid.smask&syscall.IN_MODIFY != 0 {
		wdenew.statid.smask |= syscall.IN_CLOSE_WRITE
		res = wt.modifyComplete(event, wdenew)
	}
	return
}

// processAttribute event
func (wt *WT) processAttribute(event *EventIntern, wdenew *WatchDirent) (res int) {
	wdenew.statid.smask |= syscall.IN_ATTRIB
	res = wt.attributeComplete(event, wdenew)
	return
}

// processSubfile seledct the proper event processing function
func (wt *WT) processSubfile(event *EventIntern, wde *WatchDirent) (res int) {
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
 * return 3: EventIntern with zero watch descriptor
 * return 4: Registered path is NULL
 */
func (wt *WT) processEvent(event *EventIntern) (res int) {

	wt.simulateMovedToEvent(event) // test for missing movedTo event when file moved out
	if event == nil {
		return
	}
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

func (wt *WT) simulateMovedToEvent(event *EventIntern) {

	if wt.pendingCookie != 0 &&
		(event == nil || event.Mask&(syscall.IN_MOVED_TO|syscall.IN_MOVE_SELF) == 0) {

		newevent := &EventIntern{Mask: syscall.IN_MOVE_SELF, Wd: 0, Name: "", Cookie: wt.pendingCookie}
		wt.processSelf(newevent, wt.moved[wt.pendingCookie])
	}
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
	if wde != nil && wde.statid != nil {
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
		ppath, err := filepath.Abs(filepath.Clean(pa))
		if err != nil {
			report(nil, "No include file", pa, 2)
			return
		}
		wde := wt.statNewFile(&wt.root, ppath)
		if wde != nil && wt.walkDirectory(wde, addWatches) == nil {
			fmt.Printf("Include %q\n", ppath)
			wt.addWatch(wde)
		}
	}
	for _, pa := range exv {
		ppath, err := filepath.Abs(filepath.Clean(pa))
		if err != nil {
			report(nil, "No exclude file", pa, 2)
			return
		}
		fmt.Printf("Exclude %q\n", ppath)
		wt.addExclude(ppath)
	}
	//D wt.printTable("init watchtable")
	return wt
}

/*
 * Perform processing loop.
 */
func (wt *WT) internalProcessNotify() (stop int) {

	if len(wt.data) == 0 {
		stop = 1
	}
	for stop == 0 {
		ev, err := wt.reader.NextEventWait(time.Second * 5)
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
 Initialise processing and perform processing loop.
 Shutdown processing upon error or normal return.
 Callback function called whenever notify event asks for special activity.
 Catch all system panics generated while waiting for events.
*/
func ProcessNotifyEvents(inv []string, exv []string, mask uint32, ncb *NotifyCallbacks) (res int) {

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

	wt := fillWatchTable(inv, exv, mask, ncb)

	if wt == nil || len(wt.data) == 0 {
		return 1
	} else {
		if wt.ncb.Init != nil {
			wt.ncb.Init()
		}
	}
	return wt.internalProcessNotify()
}

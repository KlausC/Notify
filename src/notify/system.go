package notify

import (
	"bytes"
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"
)

// Event convenient variant of InotifyEvent
type EventIntern struct {
	Wd     uint32
	Mask   uint32
	Cookie uint32
	Name   string
}

// maximal size of file name
const NAME_MAX = 255

/* inotify event masks -- events that are ignored. */
const IN_IGN uint32 = syscall.IN_ACCESS |
	syscall.IN_ATTRIB |
	syscall.IN_CLOSE_WRITE |
	syscall.IN_CLOSE_NOWRITE |
	syscall.IN_CREATE |
	syscall.IN_DELETE |
	syscall.IN_MODIFY |
	syscall.IN_MOVED_FROM |
	syscall.IN_MOVED_TO |
	syscall.IN_OPEN |
	syscall.IN_ISDIR

/* all interesting events returned by inotify system */
const IN_ALL uint32 = // syscall.IN_ACCESS |
syscall.IN_ATTRIB |
	syscall.IN_CLOSE_WRITE |
	// syscall.IN_CLOSE_NOWRITE |
	syscall.IN_CREATE |
	syscall.IN_DELETE |
	syscall.IN_MODIFY |
	syscall.IN_MOVED_FROM |
	syscall.IN_MOVED_TO |
	// syscall.IN_OPEN |
	syscall.IN_ISDIR |
	syscall.IN_DELETE_SELF |
	syscall.IN_MOVE_SELF |
	syscall.IN_Q_OVERFLOW |
	syscall.IN_IGNORED

// StatID is the key of an inode
type StatKey struct {
	Dev uint64
	Ino uint64
}

func (key StatKey) String() string {
	return fmt.Sprintf("%x.%d", key.Dev, key.Ino)
}

/*
Statid represents an inode
*/
type Statid struct {
	smask    uint32         // aggregation of status changes ATTRIB, MODIFY, CLOSE_WRITE
	first    *WatchDirent   // first in list of directory entries with same inode
	filestat syscall.Stat_t // file status as read from syscall.Lstat
}

// address converts a Statid address into an integer
func (addr *Statid) address() uintptr {
	return uintptr(unsafe.Pointer(addr))
}

// key returns the key part of the Statid (dev, ino)
func (s *Statid) key() StatKey {
	return StatKey{s.filestat.Dev, s.filestat.Ino}
}

// reset the MODIFY and CLOSE bits
func (s *Statid) resetChanged() {
	const mask uint32 = syscall.IN_MODIFY | syscall.IN_CLOSE_WRITE
	s.smask &= ^mask
}

// check if modify is compled by CLOSE
func (s *Statid) isChangeComplete() bool {
	const mask uint32 = syscall.IN_MODIFY | syscall.IN_CLOSE_WRITE
	return (s.smask & mask) == mask
}

// reset ATTRUBTE bit
func (s *Statid) resetAttribute() {
	s.smask &= ^uint32(syscall.IN_ATTRIB)
}

// check ATTRIBUTE bit
func (s *Statid) isAttributeComplete() bool {
	const mask uint32 = syscall.IN_ATTRIB
	return s.smask&mask == mask
}

/*
	EventReader delivers notify.Events after it has been initialized
*/
type EventReader struct {
	mask       uint32
	file       *os.File
	readbuffer []byte
	pos        uint32
	max        uint32
	channel    chan *EventIntern
}

// Init initialise EventReader
// obtain file descripto from inotifyInit and store mask to be used for addWatch calls
func (er *EventReader) Init(mask uint32) (err error) {
	er.mask = (syscall.IN_ALL_EVENTS & mask) | syscall.IN_DONT_FOLLOW | syscall.IN_EXCL_UNLINK
	fd, err := syscall.InotifyInit1(syscall.IN_CLOEXEC)
	if err != nil {
		report(err, "inotify_init1", "", 0)
		return nil
	}
	er.file = os.NewFile(uintptr(fd), "inotify")
	if err != nil {
		return
	}
	const eventsize = uint32(syscall.SizeofInotifyEvent)
	size := uint32(eventsize + NAME_MAX + 1)
	//size := uint32(eventsize+16)
	er.readbuffer = make([]byte, size)
	return
}

// addWatch call InotifyAddWatch for path, using the fd and mask of EventReader
func (er *EventReader) addWatch(path string) (wd uint32, err error) {
	fd := int(er.file.Fd())
	wd1, err := syscall.InotifyAddWatch(fd, path, er.mask)
	if err != nil {
		report(err, "inotifyAddWatch", path, 0)
	}
	wd = uint32(wd1)
	return
}

// call InotifyRmWatch for watch descriptor , using fd from EventReader
func (er *EventReader) removeWatch(wd uint32) (err error) {
	fd := int(er.file.Fd())
	_, err = syscall.InotifyRmWatch(fd, wd)
	return
}

// Close EventReader by closing underlying file
func (er *EventReader) Close() {
	er.file.Close()
}

/*
	ReadEvent reads the next event from inotify file descriptor.
	The readbuffer size must be able to contain at least one maximal size InotifyEvent
*/
func (er *EventReader) NextEvent() (ev *EventIntern, err error) {

	const eventsize = uint32(syscall.SizeofInotifyEvent)
	for true {
		if er.pos+eventsize <= er.max {
			event := eventPointer(&er.readbuffer[er.pos])
			if er.pos+eventsize+event.Len <= er.max {
				ev = &EventIntern{uint32(event.Wd), event.Mask, event.Cookie, eventName(event)}
				er.pos += eventsize + event.Len
				break
			}
		}
		copy(er.readbuffer[0:er.max-er.pos], er.readbuffer[er.pos:er.max])
		er.max -= er.pos
		er.pos = 0
		n, err := er.file.Read(er.readbuffer[er.max:])
		if err != nil {
			report(err, "Read", er.file.Name(), 0)
			return ev, err
		}
		er.max += uint32(n)
	}
	return
}

func (er *EventReader) NextEventWait(d time.Duration) (event *EventIntern, err error) {
	if er.channel == nil {
		er.channel = make(chan *EventIntern, 1)
		go func() {
			var event *EventIntern
			for event, err = er.NextEvent(); err == nil; event, err = er.NextEvent() {
				er.channel <- event
			}
			close(er.channel)
			er.channel = nil
		}()
	}
	select {
	case event = <-er.channel:
		return
	case <-time.After(d):
		return
	}
}

// eventName extracts the name string from the Name byte slice of the InotifyEvent
func eventName(event *syscall.InotifyEvent) string {
	size := event.Len
	b := make([]byte, event.Len)
	p0 := uintptr(unsafe.Pointer(&event.Name))
	for i := uint32(0); i < event.Len; i++ {
		b[i] = *(*byte)(unsafe.Pointer(uintptr(i) + p0))
		if b[i] == 0 {
			size = i
			break
		}
	}
	//D fmt.Printf("eventName: size=%d Len=%d %s\n", size, event.Len, b)
	return string(b[:size])
}

// eventPointer converts a byte address into a InotifyEvent address
func eventPointer(ba *byte) (event *syscall.InotifyEvent) {
	event = (*syscall.InotifyEvent)(unsafe.Pointer(ba))
	return event
}

// MaskToString produces a readable string form the Inotify bit mask
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

// byteToString converts a byte slice to a string assuming UTF-8 encoding with NUL termination.
func byteToString(b []byte, n uint32) string {

	leng := bytes.IndexByte(b[:n], byte(0))
	if leng < 0 {
		leng = int(n)
	}
	return string(b[:leng])
}

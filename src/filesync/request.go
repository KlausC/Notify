package filesync

import (
	"notify"
	"time"
)

// Request represents a destination side file system modification request
type Request struct {
	eventType   notify.EventType
	source      string
	sourceAlt   string
	dest string
	destAlt string
	key         notify.StatKey
	fileSync    *FileSync
	eventTime   time.Time
}

func NewRequest(fs *FileSync, ev *notify.Event) (res *Request) {
	res = &Request{
		eventType:   ev.EventType,
		source:      ev.Path,
		sourceAlt:   ev.Path2,
		dest: fs.SyncName(ev.Path),
		destAlt: fs.SyncName(ev.Path2),	
		key:         ev.Key,
		fileSync:    fs,
		eventTime:   time.Now(),
	}
	return
}

// RewriteSource
func RewriteSource(r *Request, newpath, oldpath string) {
	rewrite(newpath, oldpath, &r.source)
	rewrite(newpath, oldpath, &r.sourceAlt)
}

func RewriteDestination(r *Request, newpath, oldpath string) {
	rewrite(newpath, oldpath, &r.dest)
	rewrite(newpath, oldpath, &r.destAlt)
}

// RequestFifo is a type-safe incarnation of Fifo
type RequestFifo struct {
	queue Fifo
}

func (f RequestFifo) RewriteSource(newpath, oldpath string, predicates ...func(*Request) bool) {
	f.rewrite(newpath, oldpath, RewriteSource, predicates)
}

func (f RequestFifo) RewriteDestination(newpath, oldpath string, predicates ...func(*Request) bool) {
	f.rewrite(newpath, oldpath, RewriteDestination, predicates)
}

// Wrappers to functions of Fifo
func (f *RequestFifo) Insert(d *Request) {
	f.queue.Insert(d)
}

func (f *RequestFifo) Retrieve() (res *Request) {
	return interface2Request(f.queue.Retrieve())
}

func (f RequestFifo) First() (res *Request) {
	return interface2Request(f.queue.First())
}

func (f RequestFifo) Last() (res *Request) {
	return interface2Request(f.queue.Last())
}

func (f RequestFifo) Len() uint {
	return f.queue.Len()
}

func (f RequestFifo) Walk(cb func(*Request)) {
	cbi := func(x interface{}) {
		cb(x.(*Request))
	}
	f.queue.Walk(cbi)
}

// Implementation details

func rewrite(newpath, oldpath string, mod *string) {
	if mod != nil {
		*mod = RenameOrig(newpath, oldpath, *mod)
	}
}

func andArray(predicates []func(*Request) bool) (fand func(*Request) bool) {
	fand = func(req *Request) (res bool) {
		res = true
		for _, p := range predicates {
			res = p(req)
			if !res {
				return
			}
		}
		return
	}
	return
}

func (f RequestFifo) rewrite(newpath, oldpath string,
	fun func(*Request, string, string), predicates []func(*Request) bool) {

	predicate := andArray(predicates)
	cb := func(req *Request) {
		if predicate(req) {
			fun(req, newpath, oldpath)
		}
	}
	f.Walk(cb)
}

func interface2Request(in interface{}) (out *Request) {
	if in != nil {
		out = in.(*Request)
	}
	return
}

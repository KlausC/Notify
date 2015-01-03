package filesync

import (
	"fmt"
	"notify"
	"time"
)

func StartAll(target string, includes []string, excludes []string) {
	chan1 := make(chan *Request)
	chan2 := make(chan *Request)
	chan2a := make(chan *Request)
	chan3 := make(chan *Request)
	go startFrontend(chan1, target, includes, excludes)
	go startQueue(chan1, chan2, time.Second*5)
	go startBackend(chan3)
	startMainThread(chan2, chan2a, chan3, time.Second*15)
}

func startFrontend(evchan chan<- *Request, target string, includes []string, excludes []string) {

	doInit := func() {
		fmt.Printf("All watches are set up - starting events processing.\n")
	}

	doReport := func(path string, event *notify.EventIntern) {
		fmt.Printf("%s %s %d\n", path, notify.MaskToString(event.Mask), event.Cookie)
	}

	doEvent := func(ev *notify.Event) {
		fileSync := NewFileSync(target)
		evchan <- NewRequest(fileSync, ev)
	}

	var callbacks = notify.NotifyCallbacks{
		doInit,
		doReport,
		doEvent,
	}

	notify.ProcessNotifyEvents(includes, excludes, notify.IN_ALL, &callbacks)
}

func startQueue(inchan <-chan *Request, outchan chan<- *Request, delay time.Duration) {
	queue := &RequestFifo{}
	var timechan <-chan time.Time
	for {
		select {
		case ev := <-inchan:
			queue.Insert(ev)
			modifyQueue(queue, ev)
			evl := queue.First()
			timechan = setTimer(evl, delay)
		case t := <-timechan:
			timechan = nil
			for evl := queue.First(); evl != nil; evl = queue.First() {
				//D fmt.Printf("Time now: %s first %s\n", t, evl.eventTime.Add(delay))
				if t.After(evl.eventTime.Add(delay)) {
					evl = queue.Retrieve()
					outchan <- evl
				} else {
					timechan = setTimer(evl, delay)
					break
				}
			}
		}

	}
}

func startMainThread(inchan <-chan *Request, inchana <-chan *Request, outchan chan<- *Request, delay time.Duration) {
	queue := &RequestFifo{}
	var timechan <-chan time.Time
	for {
		select {
		case ev := <-inchan:
			queue.Insert(ev)
			modifyQueue(queue, ev)
			evl := queue.First()
			timechan = setTimer(evl, delay)
			fmt.Printf("Main in: %s\n", ev)
		case ev := <-inchana:
			queue.Insert(ev)
			modifyQueue(queue, ev)
			evl := queue.First()
			timechan = setTimer(evl, delay)
			fmt.Printf("Main oob: %s\n", ev)
		case t := <-timechan:
			timechan = nil
			for evl := queue.First(); evl != nil; evl = queue.First() {
				//D fmt.Printf("Time now: %s first %s\n", t, evl.eventTime.Add(delay))
				if t.After(evl.eventTime.Add(delay)) {
					evl = queue.Retrieve()
					outchan <- evl
					fmt.Printf("Main out: %s\n", evl)
				} else {
					timechan = setTimer(evl, delay)
					break
				}
			}
		}

	}
}

func startBackend(inchan <-chan *Request) {

	for {
		select {
		case ev := <-inchan:
			fmt.Printf("Backend: %v\n", *ev)
		}
	}
}

func setTimer(ev *Request, delay time.Duration) (timechan <-chan time.Time) {
	if ev != nil {
		timechan = time.After(delay - time.Since(ev.eventTime))
	} else {
		timechan = nil
	}
	return
}

func modifyQueue(queue *RequestFifo, ev *Request) {
	switch ev.eventType {
	case notify.MOVE:
		newpath := ev.source
		oldpath := ev.sourceAlt
		pred := func(req *Request) bool {
			rt := req.eventType
			return rt == notify.CHANGE || rt == notify.CREATE
		}
		queue.RewriteSource(newpath, oldpath, pred)

	case notify.DELETE:
		delpath := ev.source
		altpath := ev.sourceAlt
		fun := func(req *Request) {
			if req.eventType != notify.DELETE {
				if req.source == delpath {
					req.source = altpath
					req.sourceAlt = ""
				}
			}
		}
		queue.Walk(fun)
	}
}

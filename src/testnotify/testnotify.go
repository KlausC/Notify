package main

import (
	"fmt"
	"notify"
	"os"
)

func maskToString(mask uint32) string {
	return notify.MaskToString(mask)
}

func doReport(path string, event *notify.EventIntern) {
	fmt.Printf("event: %s %s %d\n", path, maskToString(event.Mask), event.Cookie)
}
func doEvent(ev *notify.Event) {
	fmt.Printf("%v %v %v %s %s %v\n", ev.EventType, ev.IsDir, ev.DataModified, ev.Path, ev.Path2, ev.Key)
}

var callbacks = notify.NotifyCallbacks{
	doReport,
	doEvent,
}

func main() {
	var res int
	defer func() {
		os.Exit(res)
	}()
	args := os.Args[1:]
	res = notify.ProcessNotifyEvents(args, nil, notify.IN_ALL, &callbacks)
}

package main

import (
	"notify"
	"fmt"
	"os"
)

func maskToString(mask uint32) string {
	return notify.MaskToString(mask)
}

func doEvent(path string , event *notify.Event) {
	fmt.Printf("event: %s %s %d\n", path, maskToString(event.Mask), event.Cookie)
}
func doCreated(path string , mask uint32) {
	fmt.Printf("created: %s %s\n", path, maskToString(mask))
}
func doDeleted(path string , mask uint32) {
	fmt.Printf("deleted: %s %s\n", path, maskToString(mask))
}
func doChanged(path string , mask uint32) {
	fmt.Printf("changed contents: %s %s\n", path, maskToString(mask))
}
func doLinked(path string , path2 string, mask uint32) {
	fmt.Printf("linked: %s %s %s\n", path, path2, maskToString(mask))
}
func doMoved(path string , path2 string, mask uint32) {
	fmt.Printf("moved: %s %s %s\n", path, path2, maskToString(mask)) 
}
func doRemoved(path string , mask uint32) {
	fmt.Printf("removed: %s %s\n", path, maskToString(mask))
}
func doAttribute(path string , mask uint32) {
	fmt.Printf("changed attributes: %s %s\n", path, maskToString(mask))
}

var callbacks = notify.NotifyCallbacks{
	doEvent,
	doCreated,
	doDeleted,
	doChanged,
	doLinked,
	doMoved,
	doRemoved,
	doAttribute,
}

func main() {
	var res int
	defer func() {
		os.Exit(res)
	}()
	args := os.Args[1:]
	res = notify.ProcessNotifyEvents(args, nil, notify.IN_ALL, &callbacks);
}


package main 

import (
	"filesync"
    "flag"
    "fmt"
)

const APP_VERSION = "0.1"

type FileList []string

func (inc *FileList)Set(s string) error {
	*inc = append(*inc, s)
	return nil
}
func (inc *FileList)String() string {
	return fmt.Sprintf("%v", []string(*inc))
}

var includes FileList
var excludes FileList

func init() {
	flag.Var(&includes, "I", "directory to be watched")
	flag.Var(&excludes, "X", "directory or file to be excluded")
}

func main() {
	  
	var target string
	versionFlag := flag.Bool("v", false, "Print the version number.")
	flag.StringVar(&target, "t", "", "target dir of the synchronisation")
	
    flag.Parse() // Scan the arguments list 

    if *versionFlag {
        fmt.Println("Version:", APP_VERSION)
    }
    
    filesync.StartAll(target, includes, excludes)
}


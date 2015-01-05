package main 

import (
	"bwt"
	"bytes"
	"fmt"
)

func trivialtest(str string) {
	
	b := bytes.NewBufferString(str)
	input := b.Bytes()
	index, coded := bwt.EncodeBwt(input)
	fmt.Printf("%d#%s\n", index, coded)

	decoded := bwt.DecodeBwt(coded, index)
	fmt.Printf("%s\n", decoded)
}

func trivialtests() {
	trivialtest("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
	trivialtest("~ananasbanane~")
	trivialtest("Wikipedia!")
	trivialtest("Große Änderungen kämen!")
	trivialtest("Große Änderungen kämen! kämmen Än der ßeße")
}

func mtftests() {
	
	r1 := bytes.NewBufferString("ananas zwerg fadensamt ananas")
	w1 := new(bytes.Buffer)
	bwt.EncodeBwtStream(r1, w1)
	
	r := bytes.NewReader(w1.Bytes())
	w := new(bytes.Buffer)
	bwt.EncodeMtfStream(r,w)
	coded := w.Bytes()
	fmt.Printf("coded: %v\n", coded)
	
	rr := bytes.NewReader(coded)
	ww := new(bytes.Buffer)
	bwt.DecodeMtfStream(rr,ww)
	
	r2 := bytes.NewReader(ww.Bytes())
	w2 := new(bytes.Buffer)
	bwt.DecodeBwtStream(r2,w2)
	
	decoded := w2.Bytes()
	fmt.Printf("decoded: %s\n", decoded)
}


func main() {
	trivialtests()
	mtftests()
}


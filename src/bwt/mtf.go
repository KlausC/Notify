package bwt

import (
	"bytes"
	"io"
)

// move forward compression algorithm

type Alphabet []byte

func FindAndMoveToFront(a *Alphabet, in byte) byte {
	pos := bytes.IndexByte(*a, in)
	for i := pos; i > 0; i-- {
		(*a)[i] = (*a)[i-1]
	}
	(*a)[0] = in
	return byte(pos)
}

func LookupAndMoveToFront(a *Alphabet, in byte) byte {
	pos := (*a)[int(in)]
	for i := int(in); i > 0; i-- {
		(*a)[i] = (*a)[i-1]
	}
	(*a)[0] = pos
	return pos
}

func createAlphabet() *Alphabet {
	var a Alphabet = make([]byte, 256)
	for i := 0; i < 256; i++ {
		a[i] = byte(i)
	}
	return &a
}

func EncodeMtfStream(reader io.ByteReader, writer io.ByteWriter) {
	processStream(reader, writer, FindAndMoveToFront)
}

func DecodeMtfStream(reader io.ByteReader, writer io.ByteWriter) {
	processStream(reader, writer, LookupAndMoveToFront)
}

func processStream(reader io.ByteReader, writer io.ByteWriter, p func(*Alphabet, byte) byte) {
	alpha := createAlphabet()
	for in, err := reader.ReadByte(); err == nil; in, err = reader.ReadByte() {
		out := p(alpha, in)
		writer.WriteByte(out)
	}
}

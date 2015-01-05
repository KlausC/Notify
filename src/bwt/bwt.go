package bwt

import (
	"fmt"
	"io"
	"sort"
)

const MAXBUFF = 1024 * 1024 * 1024 * 2

func EncodeBwtStream(reader io.ByteReader, writer io.ByteWriter) {
	bain := make([]byte, 0, MAXBUFF)
	i := 0
	for in, err := reader.ReadByte(); err == nil; in, err = reader.ReadByte() {
		bain = append(bain, in)
		i++
	}
	index, baout := EncodeBwt(bain)
	writer.WriteByte(byte((index >> 24) & 0xff))
	writer.WriteByte(byte((index >> 16) & 0xff))
	writer.WriteByte(byte((index >> 8) & 0xff))
	writer.WriteByte(byte(index & 0xff))

	for i := 0; i < len(baout); i++ {
		err := writer.WriteByte(baout[i])
		if err != nil {
			break
		}
	}
}

func DecodeBwtStream(reader io.ByteReader, writer io.ByteWriter) {
	bain := make([]byte, 0, MAXBUFF)
	i := 0
	index0, err := reader.ReadByte()
	index1, err := reader.ReadByte()
	index2, err := reader.ReadByte()
	index3, err := reader.ReadByte()
	if err != nil {
		return
	}
	index := int(index0)<<8 | int(index1)<<8 | int(index2)<<8 | int(index3)
	for in, err := reader.ReadByte(); err == nil; in, err = reader.ReadByte() {
		bain = append(bain, in)
		i++
	}
	baout := DecodeBwt(bain, index)
	for i := 0; i < len(baout); i++ {
		err := writer.WriteByte(baout[i])
		if err != nil {
			break
		}
	}
}

// sort.Interface type for encoding
type bwtSort struct {
	n    int
	base []byte
	perm []int
}

func (s *bwtSort) Len() int {
	return s.n
}

func (s *bwtSort) Swap(i, j int) {
	s.perm[i], s.perm[j] = s.perm[j], s.perm[i]
}

func (s *bwtSort) Less(i, j int) bool {
	n := s.n
	for k, xi, xj := 0, s.perm[i], s.perm[j]; k < n; k, xi, xj = k+1, (xi+1)%n, (xj+1)%n {
		bi, bj := s.base[xi], s.base[xj]
		if bi != bj {
			return bi < bj
		}
	}
	return i < j
}

func createBwtSort(input []byte) *bwtSort {
	s := new(bwtSort)
	s.n = len(input)
	s.base = input
	s.perm = make([]int, s.n)
	for i := 0; i < len(input); i++ {
		s.perm[i] = i
	}
	return s
}

func (s *bwtSort) String() string {
	n := s.n
	out := ""
	for i := 0; i < n; i++ {
		pi := s.perm[i]
		out += fmt.Sprintf("%d %d %c %s%s\n", i, pi, s.base[(pi+n-1)%n], s.base[pi:n], s.base[0:pi])
	}
	return out
}

/*
LUA code of forward processing (wiki: Burrows-Wheeler-Transformation)
function BWT_vorwaerts(text)
  local len = string.len(text)

  -- Tabelle mit allen Rotationen des Textes erzeugen
  local matrix = {}
  for i = 1, len do
    matrix[i] = string.sub(text, i) .. string.sub(text, 1, i - 1)
  end

  -- Tabelle sortieren
  for i = 1, len do
    for j = i + 1, len do
      if matrix[i] > matrix[j] then
        matrix[i], matrix[j] = matrix[j], matrix[i]
      end
    end
  end

  -- Aus jeder Zeile das letzte Zeichen nehmen
  local codiert = ""
  local index = -1
  for i = 1, len do
    codiert = codiert .. string.sub(matrix[i], -1)
    if matrix[i] == text then
      index = i
    end
  end

  return codiert, index
end
*/
func EncodeBwt(input []byte) (index int, coded []byte) {
	s := createBwtSort(input)
	sort.Sort(s)
	n := s.n
	coded = make([]byte, 0, n)
	for i := 0; i < n; i++ {
		idx := s.perm[i]
		if idx == 0 {
			index = i
		}
		ix := (idx + n - 1) % n
		coded = append(coded, s.base[ix])
	}
	//D fmt.Print(s)
	return
}

// sort.Interface type for decoding
type bwtSortD struct {
	n    int
	base []byte
	perm []int
}

func (s *bwtSortD) Len() int {
	return s.n
}

func (s *bwtSortD) Swap(i, j int) {
	s.perm[i], s.perm[j] = s.perm[j], s.perm[i]
	s.base[i], s.base[j] = s.base[j], s.base[i]
}

//
func (s *bwtSortD) Less(i, j int) bool {
	bi, bj := s.base[i], s.base[j]
	if bi != bj {
		return bi < bj
	}
	return s.perm[i] < s.perm[j]
}

func createBwtSortD(input []byte) *bwtSortD {
	s := new(bwtSortD)
	s.n = len(input)
	s.base = input
	s.perm = make([]int, s.n)
	for i := 0; i < len(input); i++ {
		s.perm[i] = i
	}
	return s
}

func (s *bwtSortD) String() string {
	n := s.n
	out := ""
	for i := 0; i < n; i++ {
		pi := s.perm[i]
		out += fmt.Sprintf("%d %d %c\n", i, pi, s.base[i])
	}
	return out
}

/*
LUA code of backward processing
function BWT_rueckwaerts(text, index)
  local len = string.len(text)

  -- Zeichen mit zugehörigen Positionen in einer Tabelle speichern
  local tabelle = {}
  for i = 1, len do
    tabelle[i] = { position = i, zeichen = string.sub(text, i, i) }
  end

  -- Diese Tabelle nach den Zeichen sortieren. Wichtig ist hier,
  -- ein ''stabiles'' Sortierverfahren einzusetzen.
  for i = 1, len - 1 do
    for j = 1, len - 1 do
      if tabelle[j].zeichen > tabelle[j + 1].zeichen then
        tabelle[j], tabelle[j + 1] = tabelle[j + 1], tabelle[j]
      end
    end
  end

  -- Beim Index beginnend einmal durch die Tabelle
  -- wandern und dabei alle Zeichen aufsammeln.
  local decodiert = ""
  local idx = index
  for i = 1, len do
    decodiert = decodiert .. tabelle[idx].zeichen
    idx = tabelle[idx].position
  end

  return decodiert
end
*/
func DecodeBwt(input []byte, index int) (decoded []byte) {
	s := createBwtSortD(input)
	sort.Sort(s)
	n := s.n
	decoded = make([]byte, 0, n)
	for i, idx := 0, index; i < n; i, idx = i+1, s.perm[idx] {
		decoded = append(decoded, s.base[idx])
	}
	//D fmt.Print(s)
	return
}

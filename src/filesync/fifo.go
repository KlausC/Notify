package filesync

import ()

type Fifo struct {
	read  uint
	write uint
	data  map[uint]interface{}
}

func (f *Fifo) Insert(d interface{}) {
	if f.data == nil {
		f.data = make(map[uint]interface{})
	}
	f.data[f.write] = d
	f.write += 1
}

func (f *Fifo) Retrieve() (res interface{}) {
	if f.read < f.write {
		res = f.data[f.read]
		delete(f.data, f.read)
		f.read += 1
		if f.read >= f.write {
			f.read, f.write = 0, 0
		}
	}
	return
}

func (f Fifo) First() (res interface{}) {
	if f.read < f.write {
		res = f.data[f.read]
	}
	return
}

func (f Fifo) Last() (res interface{}) {
	if f.read < f.write {
		res = f.data[f.write-1]
	}
	return
}

func (f Fifo) Len() uint {
	return f.write - f.read
}

func (f Fifo) Walk(cb func(interface{})) {
	for i := f.read; i < f.write; i++ {
		cb(f.data[i])
	}
}

package wrapper

import (
	"fmt"
	"math"

	pb "github.com/evilsocket/sum/proto"
	"github.com/evilsocket/sum/storage"
)

type Record struct {
	Id     uint64
	record *pb.Record
	store  *storage.Records
}

func ForRecord(store *storage.Records, record *pb.Record) Record {
	id := uint64(0)
	if record != nil {
		id = record.Id
	}
	return Record{
		Id:     id,
		record: record,
		store:  store,
	}
}

func (w Record) IsNull() bool {
	return w.record == nil
}

func (w Record) Is(b Record) bool {
	if w.record == nil || b.record == nil {
		return false
	}
	return w.record.Id == b.record.Id
}

func (w Record) Get(index int) float32 {
	return w.record.Data[index]
}

func (w Record) flush() bool {
	if w.store != nil {
		if err := w.store.Update(w.record); err != nil {
			fmt.Printf("error while fushing record %d after an update: %s\n", w.record.Id, err)
			return false
		}
	}
	return true
}

func (w Record) Set(index int, value float32) {
	old := w.record.Data[index]
	w.record.Data[index] = value
	if w.flush() == false {
		w.record.Data[index] = old
	}
}

func (w Record) Meta(name string) string {
	if v, found := w.record.Meta[name]; found == true {
		return v
	} else {
		return ""
	}
}

func (w Record) SetMeta(name, value string) {
	old, found := w.record.Meta[name]
	w.record.Meta[name] = value
	if w.flush() == false {
		if found {
			w.record.Meta[name] = old
		}
	}
}

func (w Record) Dot(b Record) float64 {
	dot := float64(0.0)
	for i, va := range w.record.Data {
		vb := b.record.Data[i]
		dot += float64(va) * float64(vb)
	}
	return dot
}

func (w Record) Magnitude() float64 {
	return math.Sqrt(w.Dot(w))
}

func (w Record) Cosine(b Record) float64 {
	cos := 0.0
	if den := w.Magnitude() * b.Magnitude(); den != 0.0 {
		cos = w.Dot(b) / den
	}
	return cos
}

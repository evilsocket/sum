package wrapper

import (
	"log"
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
	return w.record.Id == b.record.Id
}

func (w Record) Get(index int) float32 {
	return w.record.Data[index]
}

func (w Record) Set(index int, value float32) {
	old := w.record.Data[index]
	w.record.Data[index] = value
	if err := w.store.Update(w.record); err != nil {
		w.record.Data[index] = old
		log.Printf("error while updating record %d data at index %d: %s", w.record.Id, index, err)
	}
}

func (w Record) Meta(name string) string {
	if v, found := w.record.Meta[name]; found == true {
		return v
	} else {
		return ""
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

func (w Record) Jaccard(b Record) float64 {
	dot := float64(0.0)
	accu := float64(0.0)

	for i, va := range w.record.Data {
		vb := float64(b.record.Data[i])
		dot += float64(va) * vb

		if sum := float64(va) + vb; sum == 1.0 {
			accu += 1
		}
	}

	den := accu + dot
	if den == 0 {
		return 0.0
	}
	return dot / den
}

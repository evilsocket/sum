package wrapper

import (
	"log"
	"math"

	pb "github.com/evilsocket/sum/proto"
	"github.com/evilsocket/sum/storage"
)

// Record is the wrapper for a single *pb.Record object used
// to allow access to specific records to oracles during
// execution.
type Record struct {
	// Id can be used to read the record identifier.
	Id     uint64
	record *pb.Record
	store  *storage.Records
}

// ForRecord creates a Record wrapper around a raw *pb.Record object.
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

// FIXME: these flush ops should not be executed on every single
// record updated, they should instead queue and being finalized
// by the context holder just once, like a sql transaction kind
// of thing.
func (w Record) flush() bool {
	if w.store != nil {
		if err := w.store.Update(w.record); err != nil {
			log.Printf("error while fushing record %d after an update: %s", w.record.Id, err)
			return false
		}
	}
	return true
}

// IsNull returns true if the record wrapped by this object is nil.
func (w Record) IsNull() bool {
	return w.record == nil
}

// Is returns true if this wrapped record and another wrapped
// record have the same identifier, in other words if they
// are just two wrappers around the same *pb.Record object.
func (w Record) Is(b Record) bool {
	if w.record == nil || b.record == nil {
		return false
	}
	return w.record.Id == b.record.Id
}

// Get returns the index-th elements of the *pb.Record contained
// by this wrapper.
func (w Record) Get(index int) float32 {
	return w.record.Data[index]
}

// Set sets the index-th elements of the *pb.Record contained
// by this wrapper to a new value.
func (w Record) Set(index int, value float32) {
	old := w.record.Data[index]
	w.record.Data[index] = value
	if !w.flush() {
		w.record.Data[index] = old
	}
}

// Meta returns the value of a record meta data given its name
// or an empty string if not found.
func (w Record) Meta(name string) string {
	return w.record.Meta[name]
}

// SetMeta changes or creates the value of a record meta data
// given its name.
func (w Record) SetMeta(name, value string) {
	old, found := w.record.Meta[name]
	w.record.Meta[name] = value
	if !w.flush() {
		if found {
			w.record.Meta[name] = old
		}
	}
}

// Dot performs the dot product between a vector and another.
func (w Record) Dot(b Record) float64 {
	dot := float64(0.0)
	for i, va := range w.record.Data {
		vb := b.record.Data[i]
		dot += float64(va) * float64(vb)
	}
	return dot
}

// Magnitude returns the magnitude of the vector.
func (w Record) Magnitude() float64 {
	return math.Sqrt(w.Dot(w))
}

// Cosine returns the cosine similarity between a vector and another.
func (w Record) Cosine(b Record) float64 {
	cos := 0.0
	if den := w.Magnitude() * b.Magnitude(); den != 0.0 {
		cos = w.Dot(b) / den
	}
	return cos
}

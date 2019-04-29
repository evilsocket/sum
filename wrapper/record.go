package wrapper

import (
	"math"

	pb "github.com/evilsocket/sum/proto"
)

// Record is the wrapper for a single *pb.Record object used
// to allow access to specific records to oracles during
// execution. Every oracle will have this read-only view of
// the dataset while being evaulated.
type Record struct {
	// ID can be used to read the record identifier.
	ID     uint64
	record *pb.Record
}

// WrapRecord creates a Record wrapper around a raw *pb.Record object.
func WrapRecord(record *pb.Record) Record {
	w := Record{record: record}
	if record != nil {
		w.ID = record.Id
	}
	return w
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
	return w.ID == b.ID
}

// Get returns the index-th elements of the *pb.Record contained
// by this wrapper.
func (w Record) Get(index int) float64 {
	return w.record.Data[index]
}

// Meta returns the value of a record meta data given its name
// or an empty string if not found.
func (w Record) Meta(name string) string {
	return w.record.Meta[name]
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

// DotSub performs the dot product between a vector and another using up until the specificed number of elements.
func (w Record) DotSub(b Record, elems uint) float64 {
	dot := float64(0.0)
	for i := uint(0); i < elems; i++ {
		va := w.record.Data[i]
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

// Jaccard returns the Jaccard distance between a vector and another.
func (w Record) Jaccard(b Record) float64 {
	m11 := 0.0
	m10 := 0.0

	for i, va := range w.record.Data {
		vb := b.record.Data[i]
		m11 += va * vb
		if (va + vb) == 1 {
			m10++
		}
	}

	if (m10 + m11) == 0 {
		return 0
	}

	return m11 / (m11 + m10)
}

// CosineSub returns the cosine similarity between a vector and another using up until the specificed number of elements.
func (w Record) CosineSub(b Record, elems uint) float64 {
	cos := 0.0
	aMag := math.Sqrt(w.DotSub(w, elems))
	bMag := math.Sqrt(b.DotSub(b, elems))

	if den := aMag * bMag; den != 0.0 {
		cos = w.DotSub(b, elems) / den
	}
	return cos
}

// DotRange performs the dot product between a vector and another using a range of elements.
func (w Record) DotRange(b Record, start uint, end uint) float64 {
	dot := float64(0.0)
	for i := start; i < end; i++ {
		va := w.record.Data[i]
		vb := b.record.Data[i]
		dot += float64(va) * float64(vb)
	}
	return dot
}

// CosineRange returns the cosine similarity between a vector and another within a range of elements.
func (w Record) CosineRange(b Record, start uint, end uint) float64 {
	cos := 0.0
	aMag := math.Sqrt(w.DotRange(w, start, end))
	bMag := math.Sqrt(b.DotRange(b, start, end))

	if den := aMag * bMag; den != 0.0 {
		cos = w.DotRange(b, start, end) / den
	}
	return cos
}

// JaccardRange returns the Jaccard distance between a vector and another within a range of elements.
func (w Record) JaccardRange(b Record, start uint, end uint) float64 {
	m11 := 0.0
	m10 := 0.0

	for i := start; i < end; i++ {
		va := w.record.Data[i]
		vb := b.record.Data[i]
		m11 += va * vb
		if (va + vb) == 1 {
			m10++
		}
	}

	if (m10 + m11) == 0 {
		return 0
	}

	return m11 / (m11 + m10)
}

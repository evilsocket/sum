package wrapper

import (
	"math"

	pb "github.com/evilsocket/sum/proto"
)

type Record struct {
	Id     uint64
	record *pb.Record
}

func ForRecord(record *pb.Record) Record {
	return Record{
		Id:     record.Id,
		record: record,
	}
}

func (w Record) IsNull() bool {
	return w.record == nil
}

func (w Record) Is(b Record) bool {
	return w.record.Id == b.record.Id
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
	m10 := float64(0.0)

	for i, va := range w.record.Data {
		vb := float64(b.record.Data[i])
		dot += float64(va) * vb

		if sum := float64(va) + vb; sum == 1.0 {
			m10 += 1
		}
	}

	if m10+dot == 0 {
		return 0.0
	}

	return dot / (dot + m10)
}

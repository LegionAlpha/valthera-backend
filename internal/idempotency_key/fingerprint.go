package idempotency_key

import (
	"crypto/sha256"
	"encoding/binary"
	"hash"

	"github.com/google/uuid"
)

type Fingerprinter struct {
	h hash.Hash
}

func NewFingerprinter() *Fingerprinter {
	return &Fingerprinter{h: sha256.New()}
}

func (f *Fingerprinter) Int64(value int64) *Fingerprinter {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], uint64(value))
	f.h.Write(buf[:])

	return f
}

func (f *Fingerprinter) String(value string) *Fingerprinter {
	f.Int64(int64(len(value)))
	f.h.Write([]byte(value))

	return f
}

func (f *Fingerprinter) UUID(value uuid.UUID) *Fingerprinter {
	f.h.Write(value[:])

	return f
}

func (f *Fingerprinter) OptionalUUID(value *uuid.UUID) *Fingerprinter {
	if value == nil {
		f.h.Write([]byte{0})

		return f
	}

	f.h.Write([]byte{1})

	return f.UUID(*value)
}

func (f *Fingerprinter) Sum() []byte {
	return f.h.Sum(nil)
}

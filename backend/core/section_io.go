package core

import (
	"encoding/binary"
	"fmt"
	"math"
)

// SectionWriter is a small append-only writer used by section serializers
// when rebuilding a slot. It mirrors the primitive helpers on Reader and
// keeps an explicit cursor so callers can track the resulting offset.
//
// Unlike io.Writer it cannot fail — writes always succeed against an
// internal []byte buffer that grows on demand.
type SectionWriter struct {
	buf []byte
}

// NewSectionWriter returns a writer pre-allocated to the given hint capacity.
// hint may be 0; the buffer will grow as needed.
func NewSectionWriter(hint int) *SectionWriter {
	return &SectionWriter{buf: make([]byte, 0, hint)}
}

// Bytes returns the underlying buffer (no copy).
func (w *SectionWriter) Bytes() []byte { return w.buf }

// Len returns the current write cursor (== len(Bytes())).
func (w *SectionWriter) Len() int { return len(w.buf) }

func (w *SectionWriter) WriteU8(v uint8) {
	w.buf = append(w.buf, v)
}

func (w *SectionWriter) WriteU16(v uint16) {
	var tmp [2]byte
	binary.LittleEndian.PutUint16(tmp[:], v)
	w.buf = append(w.buf, tmp[:]...)
}

func (w *SectionWriter) WriteU32(v uint32) {
	var tmp [4]byte
	binary.LittleEndian.PutUint32(tmp[:], v)
	w.buf = append(w.buf, tmp[:]...)
}

func (w *SectionWriter) WriteI32(v int32) {
	w.WriteU32(uint32(v))
}

func (w *SectionWriter) WriteU64(v uint64) {
	var tmp [8]byte
	binary.LittleEndian.PutUint64(tmp[:], v)
	w.buf = append(w.buf, tmp[:]...)
}

func (w *SectionWriter) WriteF32(v float32) {
	w.WriteU32(math.Float32bits(v))
}

// WriteBytes appends raw bytes verbatim.
func (w *SectionWriter) WriteBytes(b []byte) {
	w.buf = append(w.buf, b...)
}

// WriteSizedBytes serializes a size-prefixed blob: i32 size followed by data.
// Used for `field_area`, `world_area`, `world_geom_man`, `world_geom_man2`,
// `rend_man` (see tmp/repos/er-save-manager/parser/slot_rebuild.py).
func (w *SectionWriter) WriteSizedBytes(data []byte) {
	w.WriteI32(int32(len(data)))
	w.WriteBytes(data)
}

// PadZeros appends n zero bytes. n must be >= 0.
func (w *SectionWriter) PadZeros(n int) {
	if n < 0 {
		panic(fmt.Sprintf("PadZeros: negative count %d", n))
	}
	if n == 0 {
		return
	}
	w.buf = append(w.buf, make([]byte, n)...)
}

// ReadSizedBytes mirrors WriteSizedBytes: reads an i32 size, then `size`
// raw bytes. Returns the data slice (referencing the underlying buffer
// — copy if you need to retain it past the next read).
//
// Sanity-checks `size` against the provided maximum (matches er-save-manager
// behaviour: write size as 0 if unreasonable; here we surface as an error).
func (r *Reader) ReadSizedBytes(maxSize int, fieldName string) ([]byte, error) {
	rawSize, err := r.ReadI32()
	if err != nil {
		return nil, fmt.Errorf("%s: read size: %w", fieldName, err)
	}
	if rawSize < 0 || int(rawSize) > maxSize {
		return nil, fmt.Errorf("%s: size %d out of range (max %d)", fieldName, rawSize, maxSize)
	}
	return r.ReadBytes(int(rawSize))
}

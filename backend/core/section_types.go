package core

import "fmt"

// Shared primitive types used by section serializers. These mirror the
// dataclasses in tmp/repos/er-save-manager/parser/er_types.py.

// FloatVector3 — 12 bytes (3×f32, little-endian).
type FloatVector3 struct {
	X, Y, Z float32
}

func (v *FloatVector3) Read(r *Reader) error {
	x, err := r.ReadF32()
	if err != nil {
		return fmt.Errorf("FloatVector3.X: %w", err)
	}
	y, err := r.ReadF32()
	if err != nil {
		return fmt.Errorf("FloatVector3.Y: %w", err)
	}
	z, err := r.ReadF32()
	if err != nil {
		return fmt.Errorf("FloatVector3.Z: %w", err)
	}
	v.X, v.Y, v.Z = x, y, z
	return nil
}

func (v *FloatVector3) Write(w *SectionWriter) {
	w.WriteF32(v.X)
	w.WriteF32(v.Y)
	w.WriteF32(v.Z)
}

// FloatVector4 — 16 bytes (4×f32). Used for quaternion-style angles in
// the save format despite the name; we keep the 4 components as-is.
type FloatVector4 struct {
	X, Y, Z, W float32
}

func (v *FloatVector4) Read(r *Reader) error {
	x, err := r.ReadF32()
	if err != nil {
		return fmt.Errorf("FloatVector4.X: %w", err)
	}
	y, err := r.ReadF32()
	if err != nil {
		return fmt.Errorf("FloatVector4.Y: %w", err)
	}
	z, err := r.ReadF32()
	if err != nil {
		return fmt.Errorf("FloatVector4.Z: %w", err)
	}
	wv, err := r.ReadF32()
	if err != nil {
		return fmt.Errorf("FloatVector4.W: %w", err)
	}
	v.X, v.Y, v.Z, v.W = x, y, z, wv
	return nil
}

func (v *FloatVector4) Write(w *SectionWriter) {
	w.WriteF32(v.X)
	w.WriteF32(v.Y)
	w.WriteF32(v.Z)
	w.WriteF32(v.W)
}

// MapID — 4 raw bytes encoding (area, block, x, y) in some order. The
// game treats them as opaque tuples; we store the bytes verbatim.
type MapID [4]byte

func (m *MapID) Read(r *Reader) error {
	b, err := r.ReadBytes(4)
	if err != nil {
		return fmt.Errorf("MapID: %w", err)
	}
	copy(m[:], b)
	return nil
}

func (m *MapID) Write(w *SectionWriter) {
	w.WriteBytes(m[:])
}

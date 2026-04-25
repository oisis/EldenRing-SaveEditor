package core

import "fmt"

// Sanity ceilings per section, matching er-save-manager validation.
const (
	fieldAreaMaxSize    = 0x10000  // 64 KB
	worldAreaMaxSize    = 0x10000  // 64 KB
	worldGeomManMaxSize = 0x100000 // 1 MB (per er-save-manager)
	rendManMaxSize      = 0x100000 // 1 MB
)

// SizePrefixedBlob represents a `(size: i32, data: bytes[size])` section.
// The raw `Size` header is preserved verbatim even when out of range
// (matching er-save-manager behaviour); in that case `Data` is empty
// and the section serializes as just the 4-byte header.
type SizePrefixedBlob struct {
	Size int32  // raw header value
	Data []byte // length matches Size when Size is in valid range
}

func (b *SizePrefixedBlob) Read(r *Reader, maxSize int32, fieldName string) error {
	sz, err := r.ReadI32()
	if err != nil {
		return fmt.Errorf("%s: size: %w", fieldName, err)
	}
	b.Size = sz
	if sz > 0 && sz < maxSize {
		raw, err := r.ReadBytes(int(sz))
		if err != nil {
			return fmt.Errorf("%s: data (size=%d): %w", fieldName, sz, err)
		}
		b.Data = append([]byte(nil), raw...)
	} else {
		b.Data = nil
	}
	return nil
}

func (b *SizePrefixedBlob) Write(w *SectionWriter) {
	w.WriteI32(b.Size)
	if len(b.Data) > 0 {
		w.WriteBytes(b.Data)
	}
}

// ByteSize returns the serialized length: 4-byte header + payload bytes.
func (b *SizePrefixedBlob) ByteSize() int { return 4 + len(b.Data) }

// WorldGeomBlock — combined post-event_flags world block:
// field_area + world_area + world_geom_man + world_geom_man2 + rend_man.
// Each is a SizePrefixedBlob with its own sanity ceiling.
// Reference: tmp/repos/er-save-manager/parser/world.py + slot_rebuild.py
type WorldGeomBlock struct {
	FieldArea     SizePrefixedBlob
	WorldArea     SizePrefixedBlob
	WorldGeomMan  SizePrefixedBlob
	WorldGeomMan2 SizePrefixedBlob
	RendMan       SizePrefixedBlob
}

func (b *WorldGeomBlock) Read(r *Reader) error {
	if err := b.FieldArea.Read(r, fieldAreaMaxSize, "field_area"); err != nil {
		return err
	}
	if err := b.WorldArea.Read(r, worldAreaMaxSize, "world_area"); err != nil {
		return err
	}
	if err := b.WorldGeomMan.Read(r, worldGeomManMaxSize, "world_geom_man"); err != nil {
		return err
	}
	if err := b.WorldGeomMan2.Read(r, worldGeomManMaxSize, "world_geom_man2"); err != nil {
		return err
	}
	if err := b.RendMan.Read(r, rendManMaxSize, "rend_man"); err != nil {
		return err
	}
	return nil
}

func (b *WorldGeomBlock) Write(w *SectionWriter) {
	b.FieldArea.Write(w)
	b.WorldArea.Write(w)
	b.WorldGeomMan.Write(w)
	b.WorldGeomMan2.Write(w)
	b.RendMan.Write(w)
}

// ByteSize returns the total serialized size of the 5 size-prefixed blobs.
func (b *WorldGeomBlock) ByteSize() int {
	return b.FieldArea.ByteSize() +
		b.WorldArea.ByteSize() +
		b.WorldGeomMan.ByteSize() +
		b.WorldGeomMan2.ByteSize() +
		b.RendMan.ByteSize()
}

package core

import (
	"bytes"
	"testing"
)

func TestSectionWriterPrimitives(t *testing.T) {
	w := NewSectionWriter(0)
	w.WriteU8(0xAB)
	w.WriteU16(0x1234)
	w.WriteU32(0xDEADBEEF)
	w.WriteI32(-1)
	w.WriteU64(0x0102030405060708)
	w.WriteF32(1.5)
	w.WriteBytes([]byte{0xCA, 0xFE})

	want := []byte{
		0xAB,                                           // u8
		0x34, 0x12,                                     // u16
		0xEF, 0xBE, 0xAD, 0xDE,                         // u32 LE
		0xFF, 0xFF, 0xFF, 0xFF,                         // i32 -1
		0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, // u64 LE
		0x00, 0x00, 0xC0, 0x3F, // f32 1.5
		0xCA, 0xFE,
	}
	got := w.Bytes()
	if !bytes.Equal(got, want) {
		t.Fatalf("bytes mismatch\ngot  %x\nwant %x", got, want)
	}
	if w.Len() != len(want) {
		t.Errorf("Len()=%d, want %d", w.Len(), len(want))
	}
}

func TestSectionWriterPadZeros(t *testing.T) {
	w := NewSectionWriter(0)
	w.WriteU8(0xFF)
	w.PadZeros(3)
	w.WriteU8(0xAA)
	want := []byte{0xFF, 0x00, 0x00, 0x00, 0xAA}
	if !bytes.Equal(w.Bytes(), want) {
		t.Fatalf("got %x, want %x", w.Bytes(), want)
	}
}

func TestSizedBytesRoundTrip(t *testing.T) {
	original := []byte("hello, sized blob")

	w := NewSectionWriter(0)
	w.WriteU8(0x42) // some leading byte
	w.WriteSizedBytes(original)
	w.WriteU8(0x99) // trailing byte to ensure read advances cursor

	r := NewReader(w.Bytes())
	leading, err := r.ReadU8()
	if err != nil || leading != 0x42 {
		t.Fatalf("leading: got 0x%X err=%v", leading, err)
	}
	got, err := r.ReadSizedBytes(1024, "test")
	if err != nil {
		t.Fatalf("ReadSizedBytes: %v", err)
	}
	if !bytes.Equal(got, original) {
		t.Errorf("payload mismatch: got %q, want %q", got, original)
	}
	trailing, err := r.ReadU8()
	if err != nil || trailing != 0x99 {
		t.Errorf("trailing: got 0x%X err=%v", trailing, err)
	}
}

func TestSizedBytesRejectsOverflow(t *testing.T) {
	w := NewSectionWriter(0)
	w.WriteI32(2_000_000_000) // absurdly large size
	w.PadZeros(4)             // garbage
	r := NewReader(w.Bytes())
	if _, err := r.ReadSizedBytes(1024, "field"); err == nil {
		t.Error("expected error for oversized field, got nil")
	}
}

func TestSizedBytesRejectsNegative(t *testing.T) {
	w := NewSectionWriter(0)
	w.WriteI32(-5)
	r := NewReader(w.Bytes())
	if _, err := r.ReadSizedBytes(1024, "field"); err == nil {
		t.Error("expected error for negative size, got nil")
	}
}

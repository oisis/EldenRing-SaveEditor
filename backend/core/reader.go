package core

import (
	"encoding/binary"
	"io"
)

type Reader struct {
	data []byte
	pos  int
}

func NewReader(data []byte) *Reader {
	return &Reader{data: data, pos: 0}
}

func (r *Reader) ReadU8() (uint8, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	val := r.data[r.pos]
	r.pos++
	return val, nil
}

func (r *Reader) ReadU16() (uint16, error) {
	if r.pos+2 > len(r.data) {
		return 0, io.EOF
	}
	val := binary.LittleEndian.Uint16(r.data[r.pos:])
	r.pos += 2
	return val, nil
}

func (r *Reader) ReadU32() (uint32, error) {
	if r.pos+4 > len(r.data) {
		return 0, io.EOF
	}
	val := binary.LittleEndian.Uint32(r.data[r.pos:])
	r.pos += 4
	return val, nil
}

func (r *Reader) ReadI32() (int32, error) {
	val, err := r.ReadU32()
	return int32(val), err
}

func (r *Reader) ReadU64() (uint64, error) {
	if r.pos+8 > len(r.data) {
		return 0, io.EOF
	}
	val := binary.LittleEndian.Uint64(r.data[r.pos:])
	r.pos += 8
	return val, nil
}

func (r *Reader) ReadF32() (float32, error) {
	// Not used for stats but good for parity
	return 0, nil
}

func (r *Reader) ReadBytes(n int) ([]byte, error) {
	if r.pos+n > len(r.data) {
		return nil, io.EOF
	}
	val := r.data[r.pos : r.pos+n]
	r.pos += n
	return val, nil
}

func (r *Reader) Seek(offset int64, whence int) (int64, error) {
	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = int64(r.pos) + offset
	case io.SeekEnd:
		newPos = int64(len(r.data)) + offset
	}
	r.pos = int(newPos)
	return newPos, nil
}

func (r *Reader) Pos() int {
	return r.pos
}

// UTF16ToString converts a slice of uint16 (UTF-16) to a Go string, stopping at the first null terminator.
func UTF16ToString(u []uint16) string {
	for i, v := range u {
		if v == 0 {
			u = u[:i]
			break
		}
	}
	return string(decodeUTF16(u))
}

func decodeUTF16(u []uint16) []rune {
	return []rune(string(runeSliceFromUint16(u)))
}

func runeSliceFromUint16(u []uint16) []rune {
	r := make([]rune, len(u))
	for i, v := range u {
		r[i] = rune(v)
	}
	return r
}

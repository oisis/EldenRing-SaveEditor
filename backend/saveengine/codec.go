package saveengine

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// codec is the only component of SaveForge 2.0 with direct access to raw save
// bytes. It reads a file once into a private in-memory snapshot and exposes
// nothing but private, bounded accessors: the reads used to check magics, length
// and data ranges, and one equally bounded write into the snapshot. It never
// touches the file it was created from, which is closed before openCodec
// returns.
type codec struct {
	data []byte
}

// openCodec reads path into a private snapshot. The file is opened for reading
// only and is closed before this function returns, so the engine never holds a
// handle to the user's file. A directory or any other non-regular file is
// rejected before anything is read.
func openCodec(path string) (*codec, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open save source: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("cannot inspect save source: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("save source is not a regular file")
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("cannot read save source: %w", err)
	}
	return &codec{data: data}, nil
}

// length reports the size of the snapshot in bytes.
func (source *codec) length() int64 {
	return int64(len(source.data))
}

// covers reports whether the range [offset, offset+length) lies completely
// inside the snapshot.
func (source *codec) covers(offset, length int64) bool {
	size := source.length()
	if offset < 0 || length < 0 || length > size {
		return false
	}
	return offset <= size-length
}

// readAt returns a copy of the requested range, so no caller can reach the
// snapshot itself. A range reaching past the end of the snapshot is rejected
// before any read happens.
func (source *codec) readAt(offset int64, length int) ([]byte, error) {
	if !source.covers(offset, int64(length)) {
		return nil, fmt.Errorf("range [0x%X, 0x%X) is outside the file (0x%X bytes)",
			offset, offset+int64(length), source.length())
	}
	buffer := make([]byte, length)
	copy(buffer, source.data[offset:])
	return buffer, nil
}

// writeAt replaces the range [offset, offset+len(data)) of the snapshot with
// data. The whole range is checked against the snapshot before the first byte is
// changed, so a range reaching past the end is rejected without a partial write,
// and an accepted write touches exactly that range and nothing else.
//
// It writes into the private in-memory snapshot only. No file is opened and
// nothing is persisted; the user's save is untouched until a later WriteSave
// exists.
func (source *codec) writeAt(offset int64, data []byte) error {
	if !source.covers(offset, int64(len(data))) {
		return fmt.Errorf("range [0x%X, 0x%X) is outside the file (0x%X bytes)",
			offset, offset+int64(len(data)), source.length())
	}
	copy(source.data[offset:], data)
	return nil
}

// sameAt reports whether the snapshot range starting at offset already equals
// data. A range reaching past the end of the snapshot is never equal. It exists
// so a large range can be compared without copying it out of the codec.
func (source *codec) sameAt(offset int64, data []byte) bool {
	if !source.covers(offset, int64(len(data))) {
		return false
	}
	return bytes.Equal(source.data[offset:offset+int64(len(data))], data)
}

// indexIn reports the absolute offset of the first occurrence of pattern inside
// the window [offset, offset+length), or -1 when the window holds none. The
// window is checked against the snapshot before anything is read, and the caller
// receives a position only: no snapshot byte and no copy of the window leaves
// the codec.
func (source *codec) indexIn(offset, length int64, pattern []byte) (int64, error) {
	if !source.covers(offset, length) {
		return -1, fmt.Errorf("range [0x%X, 0x%X) is outside the file (0x%X bytes)",
			offset, offset+length, source.length())
	}
	found := bytes.Index(source.data[offset:offset+length], pattern)
	if found < 0 {
		return -1, nil
	}
	return offset + int64(found), nil
}

// uint32At reads one little-endian uint32 from the given offset.
func (source *codec) uint32At(offset int64) (uint32, error) {
	raw, err := source.readAt(offset, 4)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(raw), nil
}

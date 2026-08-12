package saveengine

import (
	"bytes"
	"testing"
)

// writeAt changes exactly the requested range and nothing around it.
func TestCodecWriteAtChangesOnlyTheRequestedRange(t *testing.T) {
	source := &codec{data: []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66}}

	if err := source.writeAt(2, []byte{0xAA, 0xBB}); err != nil {
		t.Fatalf("writeAt: %v", err)
	}

	want := []byte{0x11, 0x22, 0xAA, 0xBB, 0x55, 0x66}
	if !bytes.Equal(source.data, want) {
		t.Fatalf("snapshot = % X, want % X", source.data, want)
	}
	read, err := source.readAt(2, 2)
	if err != nil {
		t.Fatalf("readAt: %v", err)
	}
	if !bytes.Equal(read, []byte{0xAA, 0xBB}) {
		t.Errorf("readAt after writeAt = % X, want AA BB", read)
	}
}

// A range that is not completely covered by the snapshot is rejected before the
// first byte is touched, so no partial write can happen.
func TestCodecWriteAtRejectsARangeOutsideTheSnapshot(t *testing.T) {
	original := []byte{0x11, 0x22, 0x33, 0x44}

	cases := map[string]struct {
		offset int64
		data   []byte
	}{
		"past the end":     {3, []byte{0xAA, 0xBB}},
		"starting past it": {4, []byte{0xAA}},
		"negative offset":  {-1, []byte{0xAA}},
		"longer than all":  {0, []byte{1, 2, 3, 4, 5}},
	}
	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			source := &codec{data: bytes.Clone(original)}
			if err := source.writeAt(testCase.offset, testCase.data); err == nil {
				t.Fatal("writeAt accepted a range outside the snapshot")
			}
			if !bytes.Equal(source.data, original) {
				t.Errorf("snapshot = % X, want the unchanged % X", source.data, original)
			}
		})
	}
}

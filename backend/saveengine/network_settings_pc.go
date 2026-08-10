package saveengine

import (
	"bytes"
	"errors"
	"fmt"
)

// PC position of UserData11. It is the last block of the PC container, so it
// starts behind UserData10 and runs to the end of the file; it is the only block
// whose length is not fixed. Inside it the PC container keeps a 0x10-byte MD5
// prefix, which is never parsed or verified, then the 0x10-byte regulation
// header, and only then the encrypted regulation blob.
const (
	pcUserData11Offset = pcUserData10Offset + pcUserData10BlockSize

	pcUserData11MD5Size    = 0x10
	pcUserData11HeaderSize = 0x10

	pcRegulationHeaderOffset = pcUserData11Offset + pcUserData11MD5Size
	pcRegulationBlobOffset   = pcRegulationHeaderOffset + pcUserData11HeaderSize
)

// pcRegulationBlob returns a copy of the encrypted regulation blob of a PC
// container. A container that ends behind UserData10 carries no UserData11 at
// all, and a block that does not start with the confirmed regulation header is
// not treated as one: both are errors instead of a read at a guessed position.
func pcRegulationBlob(source *codec) ([]byte, error) {
	if !source.covers(pcRegulationBlobOffset, 1) {
		return nil, errors.New("the PC container carries no UserData11")
	}
	header, err := source.readAt(pcRegulationHeaderOffset, len(regulationHeaderMarker))
	if err != nil {
		return nil, fmt.Errorf("cannot read the PC UserData11 header: %w", err)
	}
	if !bytes.Equal(header, regulationHeaderMarker) {
		return nil, fmt.Errorf("PC UserData11 starts with 0x%X instead of the regulation header", header)
	}
	return source.readAt(pcRegulationBlobOffset, int(source.length()-pcRegulationBlobOffset))
}

package saveengine

import (
	"bytes"
	"errors"
	"fmt"
)

// PS4 position of UserData11. Like on PC it is the last, variable-length block
// of the container and starts behind UserData10, but the PS4 container stores it
// without the MD5 prefix the PC container uses, so the 0x10-byte regulation
// header sits at the very start of the block.
const (
	ps4UserData11Offset = ps4UserData10Offset + ps4UserData10Size

	ps4UserData11HeaderSize = 0x10

	ps4RegulationHeaderOffset = ps4UserData11Offset
	ps4RegulationBlobOffset   = ps4RegulationHeaderOffset + ps4UserData11HeaderSize
)

// ps4RegulationBlob returns a copy of the encrypted regulation blob of a PS4
// container. A container that ends behind UserData10 carries no UserData11 at
// all, and a block that does not start with the confirmed regulation header is
// not treated as one: both are errors instead of a read at a guessed position.
func ps4RegulationBlob(source *codec) ([]byte, error) {
	if !source.covers(ps4RegulationBlobOffset, 1) {
		return nil, errors.New("the PS4 container carries no UserData11")
	}
	header, err := source.readAt(ps4RegulationHeaderOffset, len(regulationHeaderMarker))
	if err != nil {
		return nil, fmt.Errorf("cannot read the PS4 UserData11 header: %w", err)
	}
	if !bytes.Equal(header, regulationHeaderMarker) {
		return nil, fmt.Errorf("PS4 UserData11 starts with 0x%X instead of the regulation header", header)
	}
	return source.readAt(ps4RegulationBlobOffset, int(source.length()-ps4RegulationBlobOffset))
}

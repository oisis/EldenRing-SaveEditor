package core

import "fmt"

// NetMan — network manager block, 131,076 bytes total (0x20004).
// Layout: unk0x0 u32 | data [0x20000]byte
// Reference: tmp/repos/er-save-manager/parser/world.py:NetMan
type NetMan struct {
	Unk0x0 uint32
	Data   [0x20000]byte // 128 KB opaque payload
}

const NetManSize = 4 + 0x20000

func (n *NetMan) Read(r *Reader) error {
	v, err := r.ReadU32()
	if err != nil {
		return fmt.Errorf("net_man.unk0x0: %w", err)
	}
	n.Unk0x0 = v
	d, err := r.ReadBytes(0x20000)
	if err != nil {
		return fmt.Errorf("net_man.data: %w", err)
	}
	copy(n.Data[:], d)
	return nil
}

func (n *NetMan) Write(w *SectionWriter) {
	w.WriteU32(n.Unk0x0)
	w.WriteBytes(n.Data[:])
}

package core

import "fmt"

// Sanity bounds matching er-save-manager validation.
const (
	menuSaveLoadMaxData = 0x10000 // 64 KB ceiling on MenuSaveLoad.Data
	tutorialDataMaxData = 0x10000 // 64 KB ceiling on TutorialData.Data
	gaitemEntryCount    = 7000
	gaitemEntrySize     = 16
)

// MenuSaveLoad — variable-size menu profile data block.
// Layout: unk0x0 u16 | unk0x2 u16 | size u32 | data[size]
// Reference: tmp/repos/er-save-manager/parser/world.py:MenuSaveLoad
type MenuSaveLoad struct {
	Unk0x0 uint16
	Unk0x2 uint16
	Data   []byte // serialized length is len(Data); Size header is rewritten on Write
}

func (m *MenuSaveLoad) Read(r *Reader) error {
	u, err := r.ReadU16()
	if err != nil {
		return fmt.Errorf("menu.unk0x0: %w", err)
	}
	m.Unk0x0 = u
	u, err = r.ReadU16()
	if err != nil {
		return fmt.Errorf("menu.unk0x2: %w", err)
	}
	m.Unk0x2 = u
	size, err := r.ReadU32()
	if err != nil {
		return fmt.Errorf("menu.size: %w", err)
	}
	if size > menuSaveLoadMaxData {
		return fmt.Errorf("menu.size %d exceeds max %d", size, menuSaveLoadMaxData)
	}
	data, err := r.ReadBytes(int(size))
	if err != nil {
		return fmt.Errorf("menu.data: %w", err)
	}
	m.Data = append([]byte(nil), data...) // copy — Reader returns slice into source
	return nil
}

func (m *MenuSaveLoad) Write(w *SectionWriter) {
	w.WriteU16(m.Unk0x0)
	w.WriteU16(m.Unk0x2)
	w.WriteU32(uint32(len(m.Data)))
	w.WriteBytes(m.Data)
}

// ByteSize returns the serialized length: 8-byte header + payload.
func (m *MenuSaveLoad) ByteSize() int { return 8 + len(m.Data) }

// TrophyEquipData — fixed 52-byte block of opaque equipment metadata.
// Layout: unk0x0 u32 | unk0x4 [16]byte | unk0x14 [16]byte | unk0x24 [16]byte
// Reference: tmp/repos/er-save-manager/parser/equipment.py:TrophyEquipData
type TrophyEquipData struct {
	Unk0x0  uint32
	Unk0x4  [16]byte
	Unk0x14 [16]byte
	Unk0x24 [16]byte
}

const TrophyEquipDataSize = 52

func (t *TrophyEquipData) Read(r *Reader) error {
	v, err := r.ReadU32()
	if err != nil {
		return fmt.Errorf("trophy.unk0x0: %w", err)
	}
	t.Unk0x0 = v
	for i, dst := range []*[16]byte{&t.Unk0x4, &t.Unk0x14, &t.Unk0x24} {
		b, err := r.ReadBytes(16)
		if err != nil {
			return fmt.Errorf("trophy.block[%d]: %w", i, err)
		}
		copy(dst[:], b)
	}
	return nil
}

func (t *TrophyEquipData) Write(w *SectionWriter) {
	w.WriteU32(t.Unk0x0)
	w.WriteBytes(t.Unk0x4[:])
	w.WriteBytes(t.Unk0x14[:])
	w.WriteBytes(t.Unk0x24[:])
}

// GaitemGameDataEntry — 16 bytes per entry.
// Layout: id u32 | unk0x4 u8 | pad0x5 [3]byte | next_item_id u32 | unk0xc u8 | pad0x0d [3]byte
type GaitemGameDataEntry struct {
	ID         uint32
	Unk0x4     uint8
	Pad0x5     [3]byte
	NextItemID uint32
	Unk0xc     uint8
	Pad0x0d    [3]byte
}

func (e *GaitemGameDataEntry) Read(r *Reader) error {
	id, err := r.ReadU32()
	if err != nil {
		return err
	}
	e.ID = id
	u, err := r.ReadU8()
	if err != nil {
		return err
	}
	e.Unk0x4 = u
	p, err := r.ReadBytes(3)
	if err != nil {
		return err
	}
	copy(e.Pad0x5[:], p)
	id, err = r.ReadU32()
	if err != nil {
		return err
	}
	e.NextItemID = id
	u, err = r.ReadU8()
	if err != nil {
		return err
	}
	e.Unk0xc = u
	p, err = r.ReadBytes(3)
	if err != nil {
		return err
	}
	copy(e.Pad0x0d[:], p)
	return nil
}

func (e *GaitemGameDataEntry) Write(w *SectionWriter) {
	w.WriteU32(e.ID)
	w.WriteU8(e.Unk0x4)
	w.WriteBytes(e.Pad0x5[:])
	w.WriteU32(e.NextItemID)
	w.WriteU8(e.Unk0xc)
	w.WriteBytes(e.Pad0x0d[:])
}

// GaitemGameData — header (i64 count) + 7000 entries × 16 bytes = 0x1B458 (112,008 bytes).
type GaitemGameData struct {
	Count   int64
	Entries [gaitemEntryCount]GaitemGameDataEntry
}

const GaitemGameDataSize = 8 + gaitemEntryCount*gaitemEntrySize // 0x1B458

func (g *GaitemGameData) Read(r *Reader) error {
	count, err := r.ReadU64()
	if err != nil {
		return fmt.Errorf("gaitem.count: %w", err)
	}
	g.Count = int64(count)
	for i := 0; i < gaitemEntryCount; i++ {
		if err := g.Entries[i].Read(r); err != nil {
			return fmt.Errorf("gaitem.entry[%d]: %w", i, err)
		}
	}
	return nil
}

func (g *GaitemGameData) Write(w *SectionWriter) {
	w.WriteU64(uint64(g.Count))
	for i := 0; i < gaitemEntryCount; i++ {
		g.Entries[i].Write(w)
	}
}

// TutorialData — variable-size tutorial progress block.
// Layout: unk0x0 u16 | unk0x2 u16 | size u32 | data[size]
//
// Inside `data`: u32 count followed by remaining bytes treated as u32 IDs by
// er-save-manager. We keep the inner block as raw bytes — the count is
// encoded as the first 4 bytes of Data and we don't need to interpret it.
// Reference: tmp/repos/er-save-manager/parser/world.py:TutorialData
type TutorialData struct {
	Unk0x0 uint16
	Unk0x2 uint16
	Data   []byte // serialized length is len(Data); Size header is rewritten on Write
}

func (t *TutorialData) Read(r *Reader) error {
	u, err := r.ReadU16()
	if err != nil {
		return fmt.Errorf("tutorial.unk0x0: %w", err)
	}
	t.Unk0x0 = u
	u, err = r.ReadU16()
	if err != nil {
		return fmt.Errorf("tutorial.unk0x2: %w", err)
	}
	t.Unk0x2 = u
	size, err := r.ReadU32()
	if err != nil {
		return fmt.Errorf("tutorial.size: %w", err)
	}
	if size > tutorialDataMaxData {
		return fmt.Errorf("tutorial.size %d exceeds max %d", size, tutorialDataMaxData)
	}
	data, err := r.ReadBytes(int(size))
	if err != nil {
		return fmt.Errorf("tutorial.data: %w", err)
	}
	t.Data = append([]byte(nil), data...)
	return nil
}

func (t *TutorialData) Write(w *SectionWriter) {
	w.WriteU16(t.Unk0x0)
	w.WriteU16(t.Unk0x2)
	w.WriteU32(uint32(len(t.Data)))
	w.WriteBytes(t.Data)
}

// ByteSize returns the serialized length: 8-byte header + payload.
func (t *TutorialData) ByteSize() int { return 8 + len(t.Data) }

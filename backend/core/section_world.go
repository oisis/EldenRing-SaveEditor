package core

import "fmt"

// RideGameData — Torrent / horse state. 0x28 = 40 bytes.
// Layout:    coordinates(12) | map_id(4) | angle(16) | hp(i32) | state(u32)
// Reference: tmp/repos/er-save-manager/parser/world.py:RideGameData
type RideGameData struct {
	Coordinates FloatVector3
	MapID       MapID
	Angle       FloatVector4
	HP          int32
	State       uint32 // HorseState enum (3=DEAD, 13=ACTIVE)
}

const RideGameDataSize = 40

func (h *RideGameData) Read(r *Reader) error {
	if err := h.Coordinates.Read(r); err != nil {
		return fmt.Errorf("horse.coords: %w", err)
	}
	if err := h.MapID.Read(r); err != nil {
		return fmt.Errorf("horse.map_id: %w", err)
	}
	if err := h.Angle.Read(r); err != nil {
		return fmt.Errorf("horse.angle: %w", err)
	}
	hp, err := r.ReadI32()
	if err != nil {
		return fmt.Errorf("horse.hp: %w", err)
	}
	state, err := r.ReadU32()
	if err != nil {
		return fmt.Errorf("horse.state: %w", err)
	}
	h.HP, h.State = hp, state
	return nil
}

func (h *RideGameData) Write(w *SectionWriter) {
	h.Coordinates.Write(w)
	h.MapID.Write(w)
	h.Angle.Write(w)
	w.WriteI32(h.HP)
	w.WriteU32(h.State)
}

// BloodStain — death drop. 0x44 = 68 bytes.
// Layout: coordinates(12) | angle(16) | 5×u32 | i32×2 | map_id(4) | u32×2
// Reference: tmp/repos/er-save-manager/parser/world.py:BloodStain
type BloodStain struct {
	Coordinates FloatVector3
	Angle       FloatVector4
	Unk1c       uint32
	Unk20       uint32
	Unk24       uint32
	Unk28       uint32
	Unk2c       uint32
	Unk30       int32
	Runes       int32
	MapID       MapID
	Unk3c       uint32
	Unk38       uint32
}

const BloodStainSize = 68

func (b *BloodStain) Read(r *Reader) error {
	if err := b.Coordinates.Read(r); err != nil {
		return fmt.Errorf("bloodstain.coords: %w", err)
	}
	if err := b.Angle.Read(r); err != nil {
		return fmt.Errorf("bloodstain.angle: %w", err)
	}
	for i, dst := range []*uint32{&b.Unk1c, &b.Unk20, &b.Unk24, &b.Unk28, &b.Unk2c} {
		v, err := r.ReadU32()
		if err != nil {
			return fmt.Errorf("bloodstain.unk[%d]: %w", i, err)
		}
		*dst = v
	}
	v, err := r.ReadI32()
	if err != nil {
		return fmt.Errorf("bloodstain.unk30: %w", err)
	}
	b.Unk30 = v
	v, err = r.ReadI32()
	if err != nil {
		return fmt.Errorf("bloodstain.runes: %w", err)
	}
	b.Runes = v
	if err := b.MapID.Read(r); err != nil {
		return fmt.Errorf("bloodstain.map_id: %w", err)
	}
	uv, err := r.ReadU32()
	if err != nil {
		return fmt.Errorf("bloodstain.unk3c: %w", err)
	}
	b.Unk3c = uv
	uv, err = r.ReadU32()
	if err != nil {
		return fmt.Errorf("bloodstain.unk38: %w", err)
	}
	b.Unk38 = uv
	return nil
}

func (b *BloodStain) Write(w *SectionWriter) {
	b.Coordinates.Write(w)
	b.Angle.Write(w)
	w.WriteU32(b.Unk1c)
	w.WriteU32(b.Unk20)
	w.WriteU32(b.Unk24)
	w.WriteU32(b.Unk28)
	w.WriteU32(b.Unk2c)
	w.WriteI32(b.Unk30)
	w.WriteI32(b.Runes)
	b.MapID.Write(w)
	w.WriteU32(b.Unk3c)
	w.WriteU32(b.Unk38)
}

// WorldHead — first block after `unlocked_regions`:
// horse(40) + control_byte(1) + blood_stain(68) + unk_gdm_120(4) + unk_gdm_88(4)
// Total 117 bytes. Field naming follows er-save-manager.
type WorldHead struct {
	Horse           RideGameData
	ControlByteMaybe uint8
	BloodStain      BloodStain
	UnkGameDataMan120 uint32 // gamedataman_0x120 or _0x130 depending on version
	UnkGameDataMan88  uint32 // gamedataman_0x88
}

const WorldHeadSize = RideGameDataSize + 1 + BloodStainSize + 4 + 4

func (h *WorldHead) Read(r *Reader) error {
	if err := h.Horse.Read(r); err != nil {
		return err
	}
	cb, err := r.ReadU8()
	if err != nil {
		return fmt.Errorf("worldhead.control_byte: %w", err)
	}
	h.ControlByteMaybe = cb
	if err := h.BloodStain.Read(r); err != nil {
		return err
	}
	v, err := r.ReadU32()
	if err != nil {
		return fmt.Errorf("worldhead.unk_gdm_120: %w", err)
	}
	h.UnkGameDataMan120 = v
	v, err = r.ReadU32()
	if err != nil {
		return fmt.Errorf("worldhead.unk_gdm_88: %w", err)
	}
	h.UnkGameDataMan88 = v
	return nil
}

func (h *WorldHead) Write(w *SectionWriter) {
	h.Horse.Write(w)
	w.WriteU8(h.ControlByteMaybe)
	h.BloodStain.Write(w)
	w.WriteU32(h.UnkGameDataMan120)
	w.WriteU32(h.UnkGameDataMan88)
}

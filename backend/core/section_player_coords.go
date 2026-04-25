package core

import "fmt"

// PlayerCoordinates — current player position block, 61 bytes.
//
// Layout: coords (12) | map_id (4) | angle (16) | game_man_0xbf0 (1) |
//         unk_coords (12) | unk_angle (16)
//
// (er-save-manager labels this 57 bytes in a comment — the actual struct
//  is 61 bytes; the comment is stale.)
//
// Reference: tmp/repos/er-save-manager/parser/world.py:PlayerCoordinates
type PlayerCoordinates struct {
	Coordinates  FloatVector3
	MapID        MapID
	Angle        FloatVector4
	GameMan0xbf0 uint8
	UnkCoords    FloatVector3
	UnkAngle     FloatVector4
}

const PlayerCoordinatesSize = 12 + 4 + 16 + 1 + 12 + 16

func (p *PlayerCoordinates) Read(r *Reader) error {
	if err := p.Coordinates.Read(r); err != nil {
		return fmt.Errorf("player_coords.coords: %w", err)
	}
	if err := p.MapID.Read(r); err != nil {
		return fmt.Errorf("player_coords.map_id: %w", err)
	}
	if err := p.Angle.Read(r); err != nil {
		return fmt.Errorf("player_coords.angle: %w", err)
	}
	v, err := r.ReadU8()
	if err != nil {
		return fmt.Errorf("player_coords.game_man_0xbf0: %w", err)
	}
	p.GameMan0xbf0 = v
	if err := p.UnkCoords.Read(r); err != nil {
		return fmt.Errorf("player_coords.unk_coords: %w", err)
	}
	if err := p.UnkAngle.Read(r); err != nil {
		return fmt.Errorf("player_coords.unk_angle: %w", err)
	}
	return nil
}

func (p *PlayerCoordinates) Write(w *SectionWriter) {
	p.Coordinates.Write(w)
	p.MapID.Write(w)
	p.Angle.Write(w)
	w.WriteU8(p.GameMan0xbf0)
	p.UnkCoords.Write(w)
	p.UnkAngle.Write(w)
}

// SpawnPointBlock — fields after PlayerCoordinates and its 2-byte padding.
//
// Layout:
//   pad_after_coords           [2]byte (always zero, preserved verbatim)
//   spawn_point_entity_id      u32
//   game_man_0xb64             u32
//   temp_spawn_point_entity_id u32   (only when slot Version >= 65)
//   game_man_0xcb3             u8    (only when slot Version >= 65 — see note)
//
// Note: er-save-manager gates the two trailing fields on version >= 65 and
// >= 66 separately. Our saves are always version >= 230, so both fire; we
// pass the slot version explicitly so the rebuild matches whatever the
// slot was originally read with.
//
// Reference: tmp/repos/er-save-manager/parser/user_data_x.py:386-395
type SpawnPointBlock struct {
	PadAfterCoords          [2]byte
	SpawnPointEntityID      uint32
	GameMan0xb64            uint32
	HasTempSpawnPoint       bool   // version >= 65
	TempSpawnPointEntityID  uint32 // populated only if HasTempSpawnPoint
	HasGameMan0xcb3         bool   // version >= 66
	GameMan0xcb3            uint8  // populated only if HasGameMan0xcb3
}

func (s *SpawnPointBlock) Read(r *Reader, version uint32) error {
	pad, err := r.ReadBytes(2)
	if err != nil {
		return fmt.Errorf("spawn.pad: %w", err)
	}
	copy(s.PadAfterCoords[:], pad)
	v, err := r.ReadU32()
	if err != nil {
		return fmt.Errorf("spawn.spawn_point_entity_id: %w", err)
	}
	s.SpawnPointEntityID = v
	v, err = r.ReadU32()
	if err != nil {
		return fmt.Errorf("spawn.game_man_0xb64: %w", err)
	}
	s.GameMan0xb64 = v
	if version >= 65 {
		s.HasTempSpawnPoint = true
		v, err := r.ReadU32()
		if err != nil {
			return fmt.Errorf("spawn.temp_spawn_point: %w", err)
		}
		s.TempSpawnPointEntityID = v
	}
	if version >= 66 {
		s.HasGameMan0xcb3 = true
		u, err := r.ReadU8()
		if err != nil {
			return fmt.Errorf("spawn.game_man_0xcb3: %w", err)
		}
		s.GameMan0xcb3 = u
	}
	return nil
}

func (s *SpawnPointBlock) Write(w *SectionWriter) {
	w.WriteBytes(s.PadAfterCoords[:])
	w.WriteU32(s.SpawnPointEntityID)
	w.WriteU32(s.GameMan0xb64)
	if s.HasTempSpawnPoint {
		w.WriteU32(s.TempSpawnPointEntityID)
	}
	if s.HasGameMan0xcb3 {
		w.WriteU8(s.GameMan0xcb3)
	}
}

// ByteSize reflects the version-gated layout chosen at Read time.
func (s *SpawnPointBlock) ByteSize() int {
	n := 2 + 4 + 4
	if s.HasTempSpawnPoint {
		n += 4
	}
	if s.HasGameMan0xcb3 {
		n++
	}
	return n
}

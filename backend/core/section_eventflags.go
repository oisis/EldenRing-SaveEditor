package core

import "fmt"

// EventFlagsByteCount is the fixed length of the event_flags bitfield in
// every save slot.
// Reference: tmp/repos/er-save-manager/parser/user_data_x.py:374
const EventFlagsByteCount = 0x1BF99F

// PreEventFlagsScalars — block of scalar fields between TutorialData and
// the event_flags bitfield.
// Layout (all little-endian):
//   gameman_0x8c             u8
//   gameman_0x8d             u8
//   gameman_0x8e             u8
//   total_deaths_count       u32
//   character_type           i32
//   in_online_session_flag   u8
//   character_type_online    u32
//   last_rested_grace        u32
//   not_alone_flag           u8
//   in_game_countdown_timer  u32
//   unk_gamedataman_0x124    u32
//
// Total: 3 + 4 + 4 + 1 + 4 + 4 + 1 + 4 + 4 = 29 bytes.
// Reference: tmp/repos/er-save-manager/parser/user_data_x.py:358-371
type PreEventFlagsScalars struct {
	GameMan0x8c           uint8
	GameMan0x8d           uint8
	GameMan0x8e           uint8
	TotalDeathsCount      uint32
	CharacterType         int32
	InOnlineSessionFlag   uint8
	CharacterTypeOnline   uint32
	LastRestedGrace       uint32
	NotAloneFlag          uint8
	InGameCountdownTimer  uint32
	UnkGameDataMan0x124   uint32
}

const PreEventFlagsScalarsSize = 3 + 4 + 4 + 1 + 4 + 4 + 1 + 4 + 4 // 29

func (s *PreEventFlagsScalars) Read(r *Reader) error {
	for i, dst := range []*uint8{&s.GameMan0x8c, &s.GameMan0x8d, &s.GameMan0x8e} {
		v, err := r.ReadU8()
		if err != nil {
			return fmt.Errorf("scalars.gameman[%d]: %w", i, err)
		}
		*dst = v
	}
	v, err := r.ReadU32()
	if err != nil {
		return fmt.Errorf("scalars.total_deaths: %w", err)
	}
	s.TotalDeathsCount = v
	iv, err := r.ReadI32()
	if err != nil {
		return fmt.Errorf("scalars.character_type: %w", err)
	}
	s.CharacterType = iv
	u, err := r.ReadU8()
	if err != nil {
		return fmt.Errorf("scalars.in_online_session: %w", err)
	}
	s.InOnlineSessionFlag = u
	v, err = r.ReadU32()
	if err != nil {
		return fmt.Errorf("scalars.character_type_online: %w", err)
	}
	s.CharacterTypeOnline = v
	v, err = r.ReadU32()
	if err != nil {
		return fmt.Errorf("scalars.last_rested_grace: %w", err)
	}
	s.LastRestedGrace = v
	u, err = r.ReadU8()
	if err != nil {
		return fmt.Errorf("scalars.not_alone: %w", err)
	}
	s.NotAloneFlag = u
	v, err = r.ReadU32()
	if err != nil {
		return fmt.Errorf("scalars.countdown: %w", err)
	}
	s.InGameCountdownTimer = v
	v, err = r.ReadU32()
	if err != nil {
		return fmt.Errorf("scalars.unk_0x124: %w", err)
	}
	s.UnkGameDataMan0x124 = v
	return nil
}

func (s *PreEventFlagsScalars) Write(w *SectionWriter) {
	w.WriteU8(s.GameMan0x8c)
	w.WriteU8(s.GameMan0x8d)
	w.WriteU8(s.GameMan0x8e)
	w.WriteU32(s.TotalDeathsCount)
	w.WriteI32(s.CharacterType)
	w.WriteU8(s.InOnlineSessionFlag)
	w.WriteU32(s.CharacterTypeOnline)
	w.WriteU32(s.LastRestedGrace)
	w.WriteU8(s.NotAloneFlag)
	w.WriteU32(s.InGameCountdownTimer)
	w.WriteU32(s.UnkGameDataMan0x124)
}

// EventFlagsBlock — fixed-size 0x1BF99F bitfield + 1-byte terminator.
type EventFlagsBlock struct {
	Flags      []byte // length = EventFlagsByteCount
	Terminator uint8
}

const EventFlagsBlockSize = EventFlagsByteCount + 1

func (e *EventFlagsBlock) Read(r *Reader) error {
	data, err := r.ReadBytes(EventFlagsByteCount)
	if err != nil {
		return fmt.Errorf("event_flags: %w", err)
	}
	e.Flags = append([]byte(nil), data...)
	term, err := r.ReadU8()
	if err != nil {
		return fmt.Errorf("event_flags.terminator: %w", err)
	}
	e.Terminator = term
	return nil
}

func (e *EventFlagsBlock) Write(w *SectionWriter) {
	if len(e.Flags) != EventFlagsByteCount {
		// Defensive: guard against an accidental short slice. We pad/truncate
		// rather than panic so the rebuild never produces a malformed slot.
		buf := make([]byte, EventFlagsByteCount)
		copy(buf, e.Flags)
		w.WriteBytes(buf)
	} else {
		w.WriteBytes(e.Flags)
	}
	w.WriteU8(e.Terminator)
}

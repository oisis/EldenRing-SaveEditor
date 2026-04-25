package core

import "fmt"

// WorldAreaWeather — 12 bytes.
// Layout: area_id u16 | weather_type u16 | timer u32 | padding u32
type WorldAreaWeather struct {
	AreaID      uint16
	WeatherType uint16
	Timer       uint32
	Padding     uint32
}

const WorldAreaWeatherSize = 12

func (w *WorldAreaWeather) Read(r *Reader) error {
	v, err := r.ReadU16()
	if err != nil {
		return fmt.Errorf("weather.area_id: %w", err)
	}
	w.AreaID = v
	v, err = r.ReadU16()
	if err != nil {
		return fmt.Errorf("weather.weather_type: %w", err)
	}
	w.WeatherType = v
	u, err := r.ReadU32()
	if err != nil {
		return fmt.Errorf("weather.timer: %w", err)
	}
	w.Timer = u
	u, err = r.ReadU32()
	if err != nil {
		return fmt.Errorf("weather.padding: %w", err)
	}
	w.Padding = u
	return nil
}

func (w *WorldAreaWeather) Write(sw *SectionWriter) {
	sw.WriteU16(w.AreaID)
	sw.WriteU16(w.WeatherType)
	sw.WriteU32(w.Timer)
	sw.WriteU32(w.Padding)
}

// WorldAreaTime — 12 bytes (hour, minute, second as u32 each).
type WorldAreaTime struct {
	Hour   uint32
	Minute uint32
	Second uint32
}

const WorldAreaTimeSize = 12

func (t *WorldAreaTime) Read(r *Reader) error {
	v, err := r.ReadU32()
	if err != nil {
		return fmt.Errorf("time.hour: %w", err)
	}
	t.Hour = v
	v, err = r.ReadU32()
	if err != nil {
		return fmt.Errorf("time.minute: %w", err)
	}
	t.Minute = v
	v, err = r.ReadU32()
	if err != nil {
		return fmt.Errorf("time.second: %w", err)
	}
	t.Second = v
	return nil
}

func (t *WorldAreaTime) Write(sw *SectionWriter) {
	sw.WriteU32(t.Hour)
	sw.WriteU32(t.Minute)
	sw.WriteU32(t.Second)
}

// BaseVersion — 16 bytes (4×u32).
type BaseVersion struct {
	BaseVersionCopy uint32
	BaseVersion     uint32
	IsLatestVersion uint32
	Unk0xc          uint32
}

const BaseVersionSize = 16

func (b *BaseVersion) Read(r *Reader) error {
	v, err := r.ReadU32()
	if err != nil {
		return fmt.Errorf("base_version.copy: %w", err)
	}
	b.BaseVersionCopy = v
	v, err = r.ReadU32()
	if err != nil {
		return fmt.Errorf("base_version.value: %w", err)
	}
	b.BaseVersion = v
	v, err = r.ReadU32()
	if err != nil {
		return fmt.Errorf("base_version.is_latest: %w", err)
	}
	b.IsLatestVersion = v
	v, err = r.ReadU32()
	if err != nil {
		return fmt.Errorf("base_version.unk0xc: %w", err)
	}
	b.Unk0xc = v
	return nil
}

func (b *BaseVersion) Write(sw *SectionWriter) {
	sw.WriteU32(b.BaseVersionCopy)
	sw.WriteU32(b.BaseVersion)
	sw.WriteU32(b.IsLatestVersion)
	sw.WriteU32(b.Unk0xc)
}

// PS5Activity — 32 opaque bytes.
type PS5Activity struct {
	Data [0x20]byte
}

const PS5ActivitySize = 0x20

func (p *PS5Activity) Read(r *Reader) error {
	d, err := r.ReadBytes(0x20)
	if err != nil {
		return fmt.Errorf("ps5_activity: %w", err)
	}
	copy(p.Data[:], d)
	return nil
}

func (p *PS5Activity) Write(sw *SectionWriter) {
	sw.WriteBytes(p.Data[:])
}

// DLCSection — 50 bytes.
// Layout: pre_order_the_ring u8 | shadow_of_erdtree u8 |
//         pre_order_ring_of_miquella u8 | unused [47]byte
type DLCSection struct {
	PreorderTheRing         uint8
	ShadowOfErdtreeFlag     uint8 // SotE entry flag (non-zero = entered DLC)
	PreorderRingOfMiquella  uint8
	Unused                  [47]byte
}

const DLCSectionSerializedSize = 50

func (d *DLCSection) Read(r *Reader) error {
	v, err := r.ReadU8()
	if err != nil {
		return fmt.Errorf("dlc.preorder_ring: %w", err)
	}
	d.PreorderTheRing = v
	v, err = r.ReadU8()
	if err != nil {
		return fmt.Errorf("dlc.shadow_of_erdtree: %w", err)
	}
	d.ShadowOfErdtreeFlag = v
	v, err = r.ReadU8()
	if err != nil {
		return fmt.Errorf("dlc.preorder_miquella: %w", err)
	}
	d.PreorderRingOfMiquella = v
	rest, err := r.ReadBytes(47)
	if err != nil {
		return fmt.Errorf("dlc.unused: %w", err)
	}
	copy(d.Unused[:], rest)
	return nil
}

func (d *DLCSection) Write(sw *SectionWriter) {
	sw.WriteU8(d.PreorderTheRing)
	sw.WriteU8(d.ShadowOfErdtreeFlag)
	sw.WriteU8(d.PreorderRingOfMiquella)
	sw.WriteBytes(d.Unused[:])
}

// TrailingFixedBlock — Weather + Time + BaseVersion + SteamID + PS5Activity + DLC.
// Total 12 + 12 + 16 + 8 + 32 + 50 = 130 bytes.
type TrailingFixedBlock struct {
	Weather     WorldAreaWeather
	Time        WorldAreaTime
	BaseVersion BaseVersion
	SteamID     uint64
	PS5Activity PS5Activity
	DLC         DLCSection
}

const TrailingFixedBlockSize = WorldAreaWeatherSize + WorldAreaTimeSize +
	BaseVersionSize + 8 + PS5ActivitySize + DLCSectionSerializedSize

func (b *TrailingFixedBlock) Read(r *Reader) error {
	if err := b.Weather.Read(r); err != nil {
		return err
	}
	if err := b.Time.Read(r); err != nil {
		return err
	}
	if err := b.BaseVersion.Read(r); err != nil {
		return err
	}
	id, err := r.ReadU64()
	if err != nil {
		return fmt.Errorf("steam_id: %w", err)
	}
	b.SteamID = id
	if err := b.PS5Activity.Read(r); err != nil {
		return err
	}
	if err := b.DLC.Read(r); err != nil {
		return err
	}
	return nil
}

func (b *TrailingFixedBlock) Write(sw *SectionWriter) {
	b.Weather.Write(sw)
	b.Time.Write(sw)
	b.BaseVersion.Write(sw)
	sw.WriteU64(b.SteamID)
	b.PS5Activity.Write(sw)
	b.DLC.Write(sw)
}

package saveengine

import (
	"bytes"
	"compress/flate"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/klauspost/compress/zstd"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
)

// wantNetworkSettings is the parameter set every happy-path fixture stores. Each
// of the 22 values is distinct, so a swapped offset, a swapped field or a
// float/int confusion cannot pass unnoticed.
var wantNetworkSettings = gamecatalog.NetworkParamValues{
	MaxBreakInTargetListCount:     11,
	BreakInRequestIntervalTimeSec: 12.5,
	BreakInRequestTimeOutSec:      13.25,
	BreakInRequestAreaCount:       14,

	SummonTimeoutTime: 15.75,

	ReloadSignIntervalTime2: 16.5,
	ReloadSignTotalCount:    17,
	ReloadSignCellCount:     18,
	UpdateSignIntervalTime:  19.25,
	SingGetMax:              20,
	SignDownloadSpan:        21.5,
	SignUpdateSpan:          22.75,

	ReloadVisitListCoolTime:   23.5,
	MaxCoopBlueSummonCount:    24,
	MaxVisitListCount:         25,
	ReloadSearchCoopBlueMin:   26.25,
	ReloadSearchCoopBlueMax:   27.75,
	AllAreaSearchRateCoopBlue: 28,
	AllAreaSearchRateVsBlue:   29,

	VisitorListMax:      30,
	VisitorTimeOutTime:  31.5,
	VisitorDownloadSpan: 32.25,
}

func TestGetNetworkSettingsReadsAPCSave(t *testing.T) {
	engine := New()
	session, err := engine.LoadSave(writeNetworkPCFixture(t, networkUserData11(t, "pc", "DFLT")), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	values, err := engine.GetNetworkSettings(session.SaveSessionID)
	if err != nil {
		t.Fatalf("GetNetworkSettings: %v", err)
	}
	if values != wantNetworkSettings {
		t.Fatalf("values = %+v, want %+v", values, wantNetworkSettings)
	}
}

func TestGetNetworkSettingsReadsAPS4Save(t *testing.T) {
	engine := New()
	session, err := engine.LoadSave(writeNetworkPS4Fixture(t, networkUserData11(t, "ps4", "DFLT")), "ps4")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	values, err := engine.GetNetworkSettings(session.SaveSessionID)
	if err != nil {
		t.Fatalf("GetNetworkSettings: %v", err)
	}
	if values != wantNetworkSettings {
		t.Fatalf("values = %+v, want %+v", values, wantNetworkSettings)
	}
}

// A ZSTD regulation is the second confirmed compression format, so it has to
// decode into exactly the same values as the DFLT one.
func TestGetNetworkSettingsReadsAZSTDRegulation(t *testing.T) {
	engine := New()
	session, err := engine.LoadSave(writeNetworkPCFixture(t, networkUserData11(t, "pc", "ZSTD")), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	values, err := engine.GetNetworkSettings(session.SaveSessionID)
	if err != nil {
		t.Fatalf("GetNetworkSettings: %v", err)
	}
	if values != wantNetworkSettings {
		t.Fatalf("values = %+v, want %+v", values, wantNetworkSettings)
	}
}

func TestGetNetworkSettingsRejectsMissingUnknownAndClosedSessions(t *testing.T) {
	engine := New()
	session, err := engine.LoadSave(writeNetworkPCFixture(t, networkUserData11(t, "pc", "DFLT")), "")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	if err := engine.CloseSession(session.SaveSessionID); err != nil {
		t.Fatalf("CloseSession: %v", err)
	}

	for _, id := range []string{"", "unknown-session", session.SaveSessionID} {
		if _, err := engine.GetNetworkSettings(id); err == nil {
			t.Fatalf("GetNetworkSettings(%q) succeeded, want an error", id)
		}
	}
}

func TestGetNetworkSettingsRejectsAContainerWithoutUserData11(t *testing.T) {
	engine := New()
	pc, err := engine.LoadSave(writeNetworkPCFixture(t, nil), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	ps4, err := engine.LoadSave(writeNetworkPS4Fixture(t, nil), "ps4")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	for _, session := range []string{pc.SaveSessionID, ps4.SaveSessionID} {
		if _, err := engine.GetNetworkSettings(session); err == nil {
			t.Fatal("a container without UserData11 was decoded, want an error")
		}
	}
}

func TestGetNetworkSettingsRejectsTruncatedUserData11(t *testing.T) {
	full := networkUserData11(t, "pc", "DFLT")

	for name, userData11 := range map[string][]byte{
		"header only":          full[:0x18],
		"blob start only":      full[:0x24],
		"unaligned ciphertext": full[:len(full)-8],
	} {
		engine := New()
		session, err := engine.LoadSave(writeNetworkPCFixture(t, userData11), "pc")
		if err != nil {
			t.Fatalf("%s: LoadSave: %v", name, err)
		}
		if _, err := engine.GetNetworkSettings(session.SaveSessionID); err == nil {
			t.Fatalf("%s: truncated UserData11 was decoded, want an error", name)
		}
	}
}

func TestGetNetworkSettingsRejectsMalformedContainerData(t *testing.T) {
	noHeader := networkUserData11(t, "pc", "DFLT")
	noHeader[0x10] = 0x00

	plain := networkUserData11(t, "pc", "DFLT")
	// Overwrite the ciphertext so it decrypts into something that is not a DCX
	// archive, which is what an encrypted-only or foreign blob looks like.
	for index := 0x30; index < len(plain); index++ {
		plain[index] = 0xAB
	}

	foreign := networkUserData11PC(t, networkRegulation(t, "DFLT", networkBND4(t, "ForeignParam.param")))

	for name, userData11 := range map[string][]byte{
		"missing regulation header": noHeader,
		"undecodable blob":          plain,
		"archive without the param": foreign,
	} {
		engine := New()
		session, err := engine.LoadSave(writeNetworkPCFixture(t, userData11), "pc")
		if err != nil {
			t.Fatalf("%s: LoadSave: %v", name, err)
		}
		if _, err := engine.GetNetworkSettings(session.SaveSessionID); err == nil {
			t.Fatalf("%s: malformed UserData11 was decoded, want an error", name)
		}
	}
}

func TestGetNetworkSettingsRejectsAnUnsupportedCompressionFormat(t *testing.T) {
	engine := New()
	session, err := engine.LoadSave(writeNetworkPCFixture(t, networkUserData11(t, "pc", "EDGE")), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	if _, err := engine.GetNetworkSettings(session.SaveSessionID); err == nil {
		t.Fatal("an unsupported DCX format was decoded, want an error")
	}
}

// The getter is read-only: the source file must be byte-identical afterwards and
// a second call must return the same values from the untouched snapshot.
func TestGetNetworkSettingsMutatesNothing(t *testing.T) {
	path := writeNetworkPCFixture(t, networkUserData11(t, "pc", "DFLT"))
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	engine := New()
	session, err := engine.LoadSave(path, "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	first, err := engine.GetNetworkSettings(session.SaveSessionID)
	if err != nil {
		t.Fatalf("GetNetworkSettings: %v", err)
	}
	second, err := engine.GetNetworkSettings(session.SaveSessionID)
	if err != nil {
		t.Fatalf("second GetNetworkSettings: %v", err)
	}
	if first != second {
		t.Fatalf("second read = %+v, want the first read %+v", second, first)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("re-read fixture: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("the source fixture changed while it was read")
	}
	if info, err := engine.GetSessionInfo(session.SaveSessionID); err != nil || info.UnsavedChanges {
		t.Fatalf("session info = %+v, err = %v, want an unchanged session", info, err)
	}
}

// --- synthetic fixtures -------------------------------------------------
//
// Every fixture is built here from the format rules the getter implements. No
// real save, no captured blob and no file outside t.TempDir() is involved.

// networkRegulationKey is the AES-256-CBC key of the regulation blob, stated
// here so the fixtures encrypt what the getter decrypts.
var networkRegulationKey = []byte{
	0x99, 0xBF, 0xFC, 0x36, 0x6A, 0x6B, 0xC8, 0xC6,
	0xF5, 0x82, 0x7D, 0x09, 0x36, 0x02, 0xD6, 0x76,
	0xC4, 0x28, 0x92, 0xA0, 0x1C, 0x20, 0x7F, 0xB0,
	0x24, 0xD3, 0xAF, 0x4E, 0x49, 0x3F, 0xEF, 0x99,
}

// networkUserData11 builds a complete UserData11 block for one platform.
func networkUserData11(t *testing.T, platform string, format string) []byte {
	t.Helper()

	blob := networkRegulation(t, format, networkBND4(t, "NetworkParam.param"))
	if platform == "ps4" {
		return networkUserData11PS4(blob)
	}
	return networkUserData11PC(t, blob)
}

// networkUserData11PC prefixes the blob the way a PC container stores it: an MD5
// prefix that is never parsed, then the regulation header.
func networkUserData11PC(t *testing.T, blob []byte) []byte {
	t.Helper()

	block := make([]byte, 0x20, 0x20+len(blob))
	copy(block[0x10:], []byte{0x20, 0x47, 0x45, 0x52})
	return append(block, blob...)
}

// networkUserData11PS4 prefixes the blob the way a PS4 container stores it: the
// regulation header only, without the PC MD5 prefix.
func networkUserData11PS4(blob []byte) []byte {
	block := make([]byte, 0x10, 0x10+len(blob))
	copy(block, []byte{0x20, 0x47, 0x45, 0x52})
	return append(block, blob...)
}

// networkRegulation compresses the archive into a DCX of the given format and
// encrypts it with its own initialisation vector.
func networkRegulation(t *testing.T, format string, bnd4 []byte) []byte {
	t.Helper()

	var payload []byte
	switch format {
	case "ZSTD":
		payload = networkZSTD(t, bnd4)
	default:
		payload = networkDeflate(t, bnd4)
	}

	archive := make([]byte, 76, 76+len(payload))
	copy(archive, []byte("DCX\x00"))
	binary.BigEndian.PutUint32(archive[28:], uint32(len(bnd4)))
	binary.BigEndian.PutUint32(archive[32:], uint32(len(payload)))
	copy(archive[40:], []byte(format))
	archive = append(archive, payload...)

	plaintext := make([]byte, (len(archive)+aes.BlockSize-1)/aes.BlockSize*aes.BlockSize)
	copy(plaintext, archive)

	iv := bytes.Repeat([]byte{0x5A}, aes.BlockSize)
	block, err := aes.NewCipher(networkRegulationKey)
	if err != nil {
		t.Fatalf("aes.NewCipher: %v", err)
	}
	ciphertext := make([]byte, len(plaintext))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, plaintext)

	return append(append([]byte{}, iv...), ciphertext...)
}

// networkDeflate produces the zlib-framed deflate stream a DFLT payload carries.
func networkDeflate(t *testing.T, data []byte) []byte {
	t.Helper()

	var buffer bytes.Buffer
	buffer.Write([]byte{0x78, 0x01})
	writer, err := flate.NewWriter(&buffer, flate.BestSpeed)
	if err != nil {
		t.Fatalf("flate.NewWriter: %v", err)
	}
	if _, err := writer.Write(data); err != nil {
		t.Fatalf("deflate: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close deflate: %v", err)
	}
	return buffer.Bytes()
}

// networkZSTD produces the ZSTD stream a ZSTD payload carries.
func networkZSTD(t *testing.T, data []byte) []byte {
	t.Helper()

	var buffer bytes.Buffer
	writer, err := zstd.NewWriter(&buffer)
	if err != nil {
		t.Fatalf("zstd.NewWriter: %v", err)
	}
	if _, err := writer.Write(data); err != nil {
		t.Fatalf("zstd write: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zstd: %v", err)
	}
	return buffer.Bytes()
}

// networkBND4 builds a one-entry BND4 archive holding the parameter file under
// the given name, stored behind a directory path the way the game does.
func networkBND4(t *testing.T, name string) []byte {
	t.Helper()

	param := networkParamFile(t)
	stored := encodeUTF16LE(`N:\GR\data\Param\param\GameParam\` + name)

	const nameAt = 0x64
	dataAt := (nameAt + len(stored) + 2 + 0x0F) &^ 0x0F

	archive := make([]byte, dataAt+len(param))
	copy(archive, []byte("BND4"))
	binary.LittleEndian.PutUint32(archive[0x0C:], 1)
	binary.LittleEndian.PutUint64(archive[0x40+8:], uint64(len(param)))
	binary.LittleEndian.PutUint32(archive[0x40+24:], uint32(dataAt))
	binary.LittleEndian.PutUint32(archive[0x40+32:], uint32(nameAt))
	copy(archive[nameAt:], stored)
	copy(archive[dataAt:], param)
	return archive
}

// networkParamFile builds a parameter file in the long-data-offset layout with
// one row whose data holds the 22 values.
func networkParamFile(t *testing.T) []byte {
	t.Helper()

	const rowAt = 0x60
	param := make([]byte, rowAt+networkParamRowSize)
	param[0x2D] = 0x04
	binary.LittleEndian.PutUint64(param[0x48:], rowAt)
	copy(param[rowAt:], networkParamRowBytes())
	return param
}

// networkParamRowBytes writes wantNetworkSettings at the offsets the getter reads.
func networkParamRowBytes() []byte {
	row := make([]byte, networkParamRowSize)
	putInt := func(offset int, value int32) {
		binary.LittleEndian.PutUint32(row[offset:], uint32(value))
	}
	putFloat := func(offset int, value float32) {
		binary.LittleEndian.PutUint32(row[offset:], math.Float32bits(value))
	}

	putInt(networkMaxBreakInTargetListCount, wantNetworkSettings.MaxBreakInTargetListCount)
	putFloat(networkBreakInRequestIntervalTimeSec, wantNetworkSettings.BreakInRequestIntervalTimeSec)
	putFloat(networkBreakInRequestTimeOutSec, wantNetworkSettings.BreakInRequestTimeOutSec)
	putInt(networkBreakInRequestAreaCount, wantNetworkSettings.BreakInRequestAreaCount)

	putFloat(networkSummonTimeoutTime, wantNetworkSettings.SummonTimeoutTime)

	putFloat(networkReloadSignIntervalTime2, wantNetworkSettings.ReloadSignIntervalTime2)
	putInt(networkReloadSignTotalCount, wantNetworkSettings.ReloadSignTotalCount)
	putInt(networkReloadSignCellCount, wantNetworkSettings.ReloadSignCellCount)
	putFloat(networkUpdateSignIntervalTime, wantNetworkSettings.UpdateSignIntervalTime)
	putInt(networkSingGetMax, wantNetworkSettings.SingGetMax)
	putFloat(networkSignDownloadSpan, wantNetworkSettings.SignDownloadSpan)
	putFloat(networkSignUpdateSpan, wantNetworkSettings.SignUpdateSpan)

	putFloat(networkReloadVisitListCoolTime, wantNetworkSettings.ReloadVisitListCoolTime)
	putInt(networkMaxCoopBlueSummonCount, wantNetworkSettings.MaxCoopBlueSummonCount)
	putInt(networkMaxVisitListCount, wantNetworkSettings.MaxVisitListCount)
	putFloat(networkReloadSearchCoopBlueMin, wantNetworkSettings.ReloadSearchCoopBlueMin)
	putFloat(networkReloadSearchCoopBlueMax, wantNetworkSettings.ReloadSearchCoopBlueMax)
	row[networkAllAreaSearchRateCoopBlue] = byte(wantNetworkSettings.AllAreaSearchRateCoopBlue)
	row[networkAllAreaSearchRateVsBlue] = byte(wantNetworkSettings.AllAreaSearchRateVsBlue)

	putInt(networkVisitorListMax, wantNetworkSettings.VisitorListMax)
	putFloat(networkVisitorTimeOutTime, wantNetworkSettings.VisitorTimeOutTime)
	putFloat(networkVisitorDownloadSpan, wantNetworkSettings.VisitorDownloadSpan)
	return row
}

// writeNetworkPCFixture writes a synthetic PC container with the given
// UserData11 into t.TempDir(). A nil UserData11 produces a container that ends
// behind UserData10.
func writeNetworkPCFixture(t *testing.T, userData11 []byte) string {
	t.Helper()

	data := make([]byte, pcUserData11Offset)
	copy(data, pcMagic)
	binary.LittleEndian.PutUint32(data[pcEntryCountOffset:], pcEntryCount)
	return writeNetworkFixture(t, "pc.sl2", append(data, userData11...))
}

// writeNetworkPS4Fixture writes a synthetic PS4 container with the given
// UserData11 into t.TempDir(). A nil UserData11 produces a container that ends
// behind UserData10.
func writeNetworkPS4Fixture(t *testing.T, userData11 []byte) string {
	t.Helper()

	data := make([]byte, ps4UserData11Offset)
	copy(data, ps4Magic)
	for entry := 0; entry < ps4EntryCount; entry++ {
		at := ps4EntryTableOffset + entry*ps4EntryStride
		binary.LittleEndian.PutUint32(data[at:], uint32(ps4FirstEntryIndex+entry))
		binary.LittleEndian.PutUint32(data[at+4:], ps4EntryMarker)
	}
	return writeNetworkFixture(t, "ps4.sl2", append(data, userData11...))
}

func writeNetworkFixture(t *testing.T, name string, data []byte) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

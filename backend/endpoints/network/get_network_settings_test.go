package network

import (
	"bytes"
	"compress/flate"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf16"

	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
	"github.com/oisis/EldenRing-SaveForge/backend/saveengine"
)

// wantSettings is the parameter set the synthetic fixture stores. All 22 values
// are distinct, so the endpoint cannot pass the test with a shifted or a
// defaulted parameter set.
var wantSettings = gamecatalog.NetworkParamValues{
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

func TestGetNetworkSettingsReturnsTheStoredParameters(t *testing.T) {
	engine := saveengine.New()
	session, err := engine.LoadSave(writeSettingsFixture(t), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := GetNetworkSettings(engine, session.SaveSessionID)
	if err != nil {
		t.Fatalf("GetNetworkSettings: %v", err)
	}
	if result.SaveSessionID != session.SaveSessionID {
		t.Fatalf("saveSessionID = %q, want %q", result.SaveSessionID, session.SaveSessionID)
	}
	if result.SaveRevision != "0" {
		t.Fatalf("saveRevision = %q, want 0", result.SaveRevision)
	}
	if result.Parameters != wantSettings {
		t.Fatalf("parameters = %+v, want %+v", result.Parameters, wantSettings)
	}
}

// The result is what the transport serialises, so the exact JSON shape and every
// decoded value are part of the contract.
func TestGetNetworkSettingsSerialisesTheExactValues(t *testing.T) {
	engine := saveengine.New()
	session, err := engine.LoadSave(writeSettingsFixture(t), "pc", "local")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}

	result, err := GetNetworkSettings(engine, session.SaveSessionID)
	if err != nil {
		t.Fatalf("GetNetworkSettings: %v", err)
	}
	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}

	var decoded struct {
		SaveSessionID string                 `json:"saveSessionID"`
		SaveRevision  string                 `json:"saveRevision"`
		Parameters    map[string]json.Number `json:"parameters"`
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if decoded.SaveSessionID != session.SaveSessionID {
		t.Fatalf("saveSessionID = %q, want %q", decoded.SaveSessionID, session.SaveSessionID)
	}
	if decoded.SaveRevision != "0" {
		t.Fatalf("saveRevision = %q, want 0", decoded.SaveRevision)
	}

	want := map[string]string{
		"maxBreakInTargetListCount":     "11",
		"breakInRequestIntervalTimeSec": "12.5",
		"breakInRequestTimeOutSec":      "13.25",
		"breakInRequestAreaCount":       "14",
		"summonTimeoutTime":             "15.75",
		"reloadSignIntervalTime2":       "16.5",
		"reloadSignTotalCount":          "17",
		"reloadSignCellCount":           "18",
		"updateSignIntervalTime":        "19.25",
		"singGetMax":                    "20",
		"signDownloadSpan":              "21.5",
		"signUpdateSpan":                "22.75",
		"reloadVisitListCoolTime":       "23.5",
		"maxCoopBlueSummonCount":        "24",
		"maxVisitListCount":             "25",
		"reloadSearchCoopBlueMin":       "26.25",
		"reloadSearchCoopBlueMax":       "27.75",
		"allAreaSearchRateCoopBlue":     "28",
		"allAreaSearchRateVsBlue":       "29",
		"visitorListMax":                "30",
		"visitorTimeOutTime":            "31.5",
		"visitorDownloadSpan":           "32.25",
	}
	if len(decoded.Parameters) != len(want) {
		t.Fatalf("parameters hold %d fields, want %d", len(decoded.Parameters), len(want))
	}
	for field, value := range want {
		got, exists := decoded.Parameters[field]
		if !exists {
			t.Fatalf("the parameters are missing %q", field)
		}
		if got.String() != value {
			t.Fatalf("%s = %s, want %s", field, got, value)
		}
	}
}

func TestGetNetworkSettingsRejectsAMissingEngine(t *testing.T) {
	if _, err := GetNetworkSettings(nil, "any-session"); err == nil {
		t.Fatal("a missing engine was accepted, want an error")
	}
}

// Session validation belongs to SaveEngine, so the endpoint must pass an empty
// and an unknown identifier through instead of resolving or defaulting it.
func TestGetNetworkSettingsDelegatesSessionValidation(t *testing.T) {
	engine := saveengine.New()

	for _, id := range []string{"", "unknown-session"} {
		if _, err := GetNetworkSettings(engine, id); err == nil {
			t.Fatalf("GetNetworkSettings(%q) succeeded, want an error", id)
		}
	}
}

// --- synthetic fixture --------------------------------------------------
//
// The fixture is a synthetic PC container written into t.TempDir(). It is built
// from the documented format rules; no real save and no captured blob is used.

var settingsRegulationKey = []byte{
	0x99, 0xBF, 0xFC, 0x36, 0x6A, 0x6B, 0xC8, 0xC6,
	0xF5, 0x82, 0x7D, 0x09, 0x36, 0x02, 0xD6, 0x76,
	0xC4, 0x28, 0x92, 0xA0, 0x1C, 0x20, 0x7F, 0xB0,
	0x24, 0xD3, 0xAF, 0x4E, 0x49, 0x3F, 0xEF, 0x99,
}

const (
	settingsUserData11Offset = 0x300 + 10*0x280010 + 0x60010
	settingsRowSize          = 0x24C
)

func writeSettingsFixture(t *testing.T) string {
	t.Helper()

	data := make([]byte, settingsUserData11Offset)
	copy(data, []byte("BND4"))
	binary.LittleEndian.PutUint32(data[0x0C:], 12)

	userData11 := make([]byte, 0x20)
	copy(userData11[0x10:], []byte{0x20, 0x47, 0x45, 0x52})
	data = append(data, append(userData11, settingsRegulation(t)...)...)

	path := filepath.Join(t.TempDir(), "network-settings.sl2")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

// settingsRegulation builds the encrypted DFLT regulation blob holding one
// NetworkParam.param with the values above.
func settingsRegulation(t *testing.T) []byte {
	t.Helper()

	bnd4 := settingsBND4(t)

	var payload bytes.Buffer
	payload.Write([]byte{0x78, 0x01})
	writer, err := flate.NewWriter(&payload, flate.BestSpeed)
	if err != nil {
		t.Fatalf("flate.NewWriter: %v", err)
	}
	if _, err := writer.Write(bnd4); err != nil {
		t.Fatalf("deflate: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close deflate: %v", err)
	}

	archive := make([]byte, 76, 76+payload.Len())
	copy(archive, []byte("DCX\x00"))
	binary.BigEndian.PutUint32(archive[28:], uint32(len(bnd4)))
	binary.BigEndian.PutUint32(archive[32:], uint32(payload.Len()))
	copy(archive[40:], []byte("DFLT"))
	archive = append(archive, payload.Bytes()...)

	plaintext := make([]byte, (len(archive)+aes.BlockSize-1)/aes.BlockSize*aes.BlockSize)
	copy(plaintext, archive)

	iv := bytes.Repeat([]byte{0x5A}, aes.BlockSize)
	block, err := aes.NewCipher(settingsRegulationKey)
	if err != nil {
		t.Fatalf("aes.NewCipher: %v", err)
	}
	ciphertext := make([]byte, len(plaintext))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, plaintext)
	return append(append([]byte{}, iv...), ciphertext...)
}

func settingsBND4(t *testing.T) []byte {
	t.Helper()

	param := settingsParamFile()

	units := utf16.Encode([]rune(`N:\GR\data\Param\param\GameParam\NetworkParam.param`))
	name := make([]byte, len(units)*2)
	for index, unit := range units {
		binary.LittleEndian.PutUint16(name[index*2:], unit)
	}

	const nameAt = 0x64
	dataAt := (nameAt + len(name) + 2 + 0x0F) &^ 0x0F

	archive := make([]byte, dataAt+len(param))
	copy(archive, []byte("BND4"))
	binary.LittleEndian.PutUint32(archive[0x0C:], 1)
	binary.LittleEndian.PutUint64(archive[0x40+8:], uint64(len(param)))
	binary.LittleEndian.PutUint32(archive[0x40+24:], uint32(dataAt))
	binary.LittleEndian.PutUint32(archive[0x40+32:], uint32(nameAt))
	copy(archive[nameAt:], name)
	copy(archive[dataAt:], param)
	return archive
}

func settingsParamFile() []byte {
	const rowAt = 0x60
	param := make([]byte, rowAt+settingsRowSize)
	param[0x2D] = 0x04
	binary.LittleEndian.PutUint64(param[0x48:], rowAt)

	row := param[rowAt:]
	putInt := func(offset int, value int32) {
		binary.LittleEndian.PutUint32(row[offset:], uint32(value))
	}
	putFloat := func(offset int, value float32) {
		binary.LittleEndian.PutUint32(row[offset:], math.Float32bits(value))
	}

	putFloat(0x08, wantSettings.SummonTimeoutTime)
	putFloat(0x1C, wantSettings.ReloadSignIntervalTime2)
	putInt(0x20, wantSettings.ReloadSignTotalCount)
	putInt(0x24, wantSettings.ReloadSignCellCount)
	putFloat(0x28, wantSettings.UpdateSignIntervalTime)
	putInt(0x60, wantSettings.SingGetMax)
	putFloat(0x64, wantSettings.SignDownloadSpan)
	putFloat(0x68, wantSettings.SignUpdateSpan)
	putInt(0x70, wantSettings.MaxBreakInTargetListCount)
	putFloat(0x74, wantSettings.BreakInRequestIntervalTimeSec)
	putFloat(0x78, wantSettings.BreakInRequestTimeOutSec)
	putInt(0x7C, wantSettings.BreakInRequestAreaCount)
	putFloat(0x180, wantSettings.ReloadVisitListCoolTime)
	putInt(0x184, wantSettings.MaxCoopBlueSummonCount)
	putInt(0x18C, wantSettings.MaxVisitListCount)
	putFloat(0x190, wantSettings.ReloadSearchCoopBlueMin)
	putFloat(0x194, wantSettings.ReloadSearchCoopBlueMax)
	row[0x1D8] = byte(wantSettings.AllAreaSearchRateCoopBlue)
	row[0x1D9] = byte(wantSettings.AllAreaSearchRateVsBlue)
	putInt(0x240, wantSettings.VisitorListMax)
	putFloat(0x244, wantSettings.VisitorTimeOutTime)
	putFloat(0x248, wantSettings.VisitorDownloadSpan)
	return param
}

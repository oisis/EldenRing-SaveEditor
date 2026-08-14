package saveengine

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
)

var setNetworkSettingsTarget = gamecatalog.NetworkParamValues{
	MaxBreakInTargetListCount:     9,
	BreakInRequestIntervalTimeSec: 10.5,
	BreakInRequestTimeOutSec:      11.5,
	BreakInRequestAreaCount:       12,
	SummonTimeoutTime:             13.5,
	ReloadSignIntervalTime2:       14.5,
	ReloadSignTotalCount:          24,
	ReloadSignCellCount:           16,
	UpdateSignIntervalTime:        17.5,
	SingGetMax:                    32,
	SignDownloadSpan:              18.5,
	SignUpdateSpan:                19.5,
	ReloadVisitListCoolTime:       20.5,
	MaxCoopBlueSummonCount:        3,
	MaxVisitListCount:             22,
	ReloadSearchCoopBlueMin:       23.5,
	ReloadSearchCoopBlueMax:       24.5,
	AllAreaSearchRateCoopBlue:     25,
	AllAreaSearchRateVsBlue:       26,
	VisitorListMax:                27,
	VisitorTimeOutTime:            28.5,
	VisitorDownloadSpan:           29.5,
}

func TestSetNetworkSettingsRoundTripsSupportedLayouts(t *testing.T) {
	for _, test := range []struct {
		name     string
		platform string
		format   string
	}{
		{"PC DFLT", "pc", "DFLT"},
		{"PC ZSTD", "pc", "ZSTD"},
		{"PS4 DFLT", "ps4", "DFLT"},
		{"PS4 ZSTD", "ps4", "ZSTD"},
	} {
		t.Run(test.name, func(t *testing.T) {
			userData11 := networkUserData11ForSet(t, test.platform, test.format)
			var originalPS4Archive []byte
			if test.platform == "ps4" && test.format == "ZSTD" {
				archive, err := decryptRegulation(userData11[ps4UserData11HeaderSize:])
				if err != nil {
					t.Fatalf("decrypt original PS4 regulation: %v", err)
				}
				originalPS4Archive = archive
			}
			var source string
			if test.platform == "ps4" {
				source = writeNetworkPS4Fixture(t, userData11)
			} else {
				source = writeNetworkPCFixture(t, userData11)
			}
			original, err := os.ReadFile(source)
			if err != nil {
				t.Fatalf("read source: %v", err)
			}

			engine := New()
			loaded, err := engine.LoadSave(source, test.platform)
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			result, err := engine.SetNetworkSettings(
				loaded.SaveSessionID, setNetworkSettingsTarget, "0")
			if err != nil {
				t.Fatalf("SetNetworkSettings: %v", err)
			}
			if result.SaveRevision != "1" || result.NetworkSettings != setNetworkSettingsTarget {
				t.Fatalf("result = %+v", result)
			}
			got, err := engine.GetNetworkSettings(loaded.SaveSessionID)
			if err != nil {
				t.Fatalf("GetNetworkSettings: %v", err)
			}
			if got != setNetworkSettingsTarget {
				t.Fatalf("settings = %+v, want %+v", got, setNetworkSettingsTarget)
			}
			snapshot := engine.sessions[loaded.SaveSessionID].snapshot
			if test.platform == "pc" {
				userData, err := snapshot.readAt(
					pcUserData11Offset, int(snapshot.length()-pcUserData11Offset))
				if err != nil {
					t.Fatalf("read PC UserData11: %v", err)
				}
				sum := md5.Sum(userData[pcUserData11MD5Size:])
				if !bytes.Equal(userData[:pcUserData11MD5Size], sum[:]) {
					t.Fatal("PC UserData11 MD5 was not updated")
				}
			}
			if test.platform == "ps4" && test.format == "ZSTD" {
				blob, err := ps4RegulationBlob(snapshot)
				if err != nil {
					t.Fatalf("read patched PS4 regulation: %v", err)
				}
				patchedArchive, err := decryptRegulation(blob)
				if err != nil {
					t.Fatalf("decrypt patched PS4 regulation: %v", err)
				}
				originalBlocks, err := walkNetworkZSTDBlocks(
					originalPS4Archive[dcxHeaderSize:], 1)
				if err != nil || len(originalBlocks) != 1 {
					t.Fatalf("walk original ZSTD blocks: blocks=%d err=%v", len(originalBlocks), err)
				}
				patchedBlocks, err := walkNetworkZSTDBlocks(patchedArchive[dcxHeaderSize:], 1)
				if err != nil || len(patchedBlocks) != 1 {
					t.Fatalf("walk patched ZSTD blocks: blocks=%d err=%v", len(patchedBlocks), err)
				}
				frameHeaderEnd := originalBlocks[0].start
				if !bytes.Equal(
					originalPS4Archive[dcxHeaderSize:dcxHeaderSize+frameHeaderEnd],
					patchedArchive[dcxHeaderSize:dcxHeaderSize+frameHeaderEnd],
				) || patchedBlocks[0].typeID != 0 {
					t.Fatal("PS4 ZSTD frame header was changed or its target block was not replaced with RAW")
				}
			}

			target := filepath.Join(t.TempDir(), "written.sl2")
			if _, err := engine.WriteSave(loaded.SaveSessionID, "1", target); err != nil {
				t.Fatalf("WriteSave: %v", err)
			}
			reloaded := New()
			again, err := reloaded.LoadSave(target, test.platform)
			if err != nil {
				t.Fatalf("reload: %v", err)
			}
			got, err = reloaded.GetNetworkSettings(again.SaveSessionID)
			if err != nil || got != setNetworkSettingsTarget {
				t.Fatalf("reloaded settings = %+v, err = %v", got, err)
			}

			after, err := os.ReadFile(source)
			if err != nil {
				t.Fatalf("re-read source: %v", err)
			}
			if !bytes.Equal(after, original) {
				t.Fatal("the source save was modified")
			}
		})
	}
}

func TestSetNetworkSettingsRejectsInvalidInputWithoutMutation(t *testing.T) {
	for name, alter := range map[string]func(*gamecatalog.NetworkParamValues){
		"field range": func(values *gamecatalog.NetworkParamValues) {
			values.MaxBreakInTargetListCount = 21
		},
		"non-finite float": func(values *gamecatalog.NetworkParamValues) {
			values.SummonTimeoutTime = float32(math.NaN())
		},
		"sign count relation": func(values *gamecatalog.NetworkParamValues) {
			values.ReloadSignCellCount = values.ReloadSignTotalCount + 1
		},
		"search range relation": func(values *gamecatalog.NetworkParamValues) {
			values.ReloadSearchCoopBlueMin = values.ReloadSearchCoopBlueMax + 1
		},
	} {
		t.Run(name, func(t *testing.T) {
			engine := New()
			loaded, err := engine.LoadSave(
				writeNetworkPCFixture(t, networkUserData11ForSet(t, "pc", "DFLT")), "pc")
			if err != nil {
				t.Fatalf("LoadSave: %v", err)
			}
			before := append([]byte(nil), engine.sessions[loaded.SaveSessionID].snapshot.data...)
			values := setNetworkSettingsTarget
			alter(&values)

			if result, err := engine.SetNetworkSettings(loaded.SaveSessionID, values, "0"); err == nil {
				t.Fatalf("SetNetworkSettings succeeded: %+v", result)
			}
			session := engine.sessions[loaded.SaveSessionID]
			if !bytes.Equal(session.snapshot.data, before) || session.session.revisionString() != "0" || session.session.dirty {
				t.Fatal("rejected input changed the snapshot, revision or dirty flag")
			}
		})
	}
}

func TestSetNetworkSettingsRejectsARevisionConflictBeforeMutation(t *testing.T) {
	engine := New()
	loaded, err := engine.LoadSave(
		writeNetworkPCFixture(t, networkUserData11ForSet(t, "pc", "DFLT")), "pc")
	if err != nil {
		t.Fatalf("LoadSave: %v", err)
	}
	before := append([]byte(nil), engine.sessions[loaded.SaveSessionID].snapshot.data...)

	if _, err := engine.SetNetworkSettings(loaded.SaveSessionID, setNetworkSettingsTarget, "1"); err == nil {
		t.Fatal("a stale expectedRevision was accepted")
	}
	session := engine.sessions[loaded.SaveSessionID]
	if !bytes.Equal(session.snapshot.data, before) || session.session.revisionString() != "0" || session.session.dirty {
		t.Fatal("a revision conflict changed the session")
	}
}

// networkUserData11ForSet gives the encrypted blob spare fixed capacity. Native
// UserData11 has that capacity; the getter's compact fixture intentionally does
// not. ZSTD is emitted without a checksum, matching the confirmed PS4 frame.
func networkUserData11ForSet(t *testing.T, platform, format string) []byte {
	t.Helper()
	bnd4 := networkBND4(t, "NetworkParam.param")
	var payload []byte
	if format == "ZSTD" {
		var buffer bytes.Buffer
		writer, err := zstd.NewWriter(&buffer, zstd.WithEncoderCRC(false))
		if err != nil {
			t.Fatalf("zstd.NewWriter: %v", err)
		}
		if _, err := writer.Write(bnd4); err != nil {
			t.Fatalf("zstd write: %v", err)
		}
		if err := writer.Close(); err != nil {
			t.Fatalf("zstd close: %v", err)
		}
		payload = buffer.Bytes()
	} else {
		payload = networkDeflate(t, bnd4)
	}

	archive := make([]byte, dcxHeaderSize, dcxHeaderSize+len(payload))
	copy(archive, []byte("DCX\x00"))
	binary.BigEndian.PutUint32(archive[dcxDecompressedSizeOffset:], uint32(len(bnd4)))
	binary.BigEndian.PutUint32(archive[dcxCompressedSizeOffset:], uint32(len(payload)))
	copy(archive[dcxFormatOffset:], format)
	archive = append(archive, payload...)

	plaintext := make([]byte, (len(archive)+4096+aes.BlockSize-1)/aes.BlockSize*aes.BlockSize)
	copy(plaintext, archive)
	iv := bytes.Repeat([]byte{0x5A}, aes.BlockSize)
	block, err := aes.NewCipher(networkRegulationKey)
	if err != nil {
		t.Fatalf("aes.NewCipher: %v", err)
	}
	ciphertext := make([]byte, len(plaintext))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, plaintext)
	blob := append(append([]byte(nil), iv...), ciphertext...)
	if platform == "ps4" {
		return networkUserData11PS4(blob)
	}
	return networkUserData11PC(t, blob)
}

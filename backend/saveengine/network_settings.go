package saveengine

import (
	"bytes"
	"compress/flate"
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"unicode/utf16"

	"github.com/klauspost/compress/zstd"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"

	"github.com/oisis/EldenRing-SaveForge/backend/apperror"
)

// Layout of the regulation archive stored in UserData11, shared by PC and PS4.
// The container part — where UserData11 begins and how many bytes precede the
// regulation blob — differs per platform and lives in network_settings_pc.go and
// network_settings_ps4.go; everything from the encrypted blob inwards is
// identical on both platforms and is decoded here.
const (
	// The regulation blob is AES-256-CBC encrypted and carries its own
	// initialisation vector in front of the ciphertext.
	regulationIVSize = 16

	// The plaintext is a DCX archive: the magic, the two big-endian sizes and
	// the four-character compression format all sit in its fixed 0x4C header,
	// and the compressed payload follows it.
	dcxMagic                  = "DCX\x00"
	dcxDecompressedSizeOffset = 28
	dcxCompressedSizeOffset   = 32
	dcxFormatOffset           = 40
	dcxFormatSize             = 4
	dcxHeaderSize             = 76
	dcxFormatDFLT             = "DFLT"
	dcxFormatZSTD             = "ZSTD"

	// The DFLT payload is a zlib stream, so its two-byte header precedes the
	// raw deflate data this decoder consumes.
	dcxDeflateHeaderSize = 2

	// maxRegulationSize bounds the decompressed archive. The declared
	// decompressed size comes from the file itself, so it is never trusted as an
	// allocation size; a stream claiming more than this is rejected instead of
	// being buffered.
	maxRegulationSize = 64 << 20

	// The decompressed archive is a BND4 with 0x24-byte entry descriptors: the
	// file count sits in the header, and every descriptor carries the size, the
	// data offset and the offset of its UTF-16LE name.
	bnd4Magic                = "BND4"
	bnd4FileCountOffset      = 0x0C
	bnd4EntryTableOffset     = 0x40
	bnd4EntrySize            = 0x24
	bnd4EntrySizeOffset      = 8
	bnd4EntryDataOffset      = 24
	bnd4EntryNameOffset      = 32
	networkParamName         = "NetworkParam.param"
	networkParamHeaderSize   = 0x58
	networkParamFormatOffset = 0x2D
	networkParamLongDataFlag = 0x04
	networkParamRow0Offset   = 0x48
)

// Byte offsets of the 22 reported values inside the NETWORK_PARAM_ST row 0 data,
// and the number of bytes that row must therefore provide. Every value is
// little-endian; the two search rates are single bytes and the remaining twenty
// are four-byte scalars, read as int32 or float32 exactly as stored.
const (
	networkSummonTimeoutTime = 0x08

	networkReloadSignIntervalTime2 = 0x1C
	networkReloadSignTotalCount    = 0x20
	networkReloadSignCellCount     = 0x24
	networkUpdateSignIntervalTime  = 0x28
	networkSingGetMax              = 0x60
	networkSignDownloadSpan        = 0x64
	networkSignUpdateSpan          = 0x68

	networkMaxBreakInTargetListCount     = 0x70
	networkBreakInRequestIntervalTimeSec = 0x74
	networkBreakInRequestTimeOutSec      = 0x78
	// 0x7C is declared as padding in the PARAMDEF but holds the effective
	// break-in area count, which is why it is read like any other field.
	networkBreakInRequestAreaCount = 0x7C

	networkReloadVisitListCoolTime = 0x180
	networkMaxCoopBlueSummonCount  = 0x184
	networkMaxVisitListCount       = 0x18C
	networkReloadSearchCoopBlueMin = 0x190
	networkReloadSearchCoopBlueMax = 0x194

	networkAllAreaSearchRateCoopBlue = 0x1D8
	networkAllAreaSearchRateVsBlue   = 0x1D9

	networkVisitorListMax      = 0x240
	networkVisitorTimeOutTime  = 0x244
	networkVisitorDownloadSpan = 0x248

	networkParamRowSize = 0x24C
)

// regulationKey is the AES-256-CBC key of the regulation blob. Both platforms
// use the same key for reading and writing the private session snapshot.
var regulationKey = []byte{
	0x99, 0xBF, 0xFC, 0x36, 0x6A, 0x6B, 0xC8, 0xC6,
	0xF5, 0x82, 0x7D, 0x09, 0x36, 0x02, 0xD6, 0x76,
	0xC4, 0x28, 0x92, 0xA0, 0x1C, 0x20, 0x7F, 0xB0,
	0x24, 0xD3, 0xAF, 0x4E, 0x49, 0x3F, 0xEF, 0x99,
}

// NetworkSettingsSnapshot is the network parameter set and the exact session
// revision whose private snapshot supplied it.
type NetworkSettingsSnapshot struct {
	SaveSessionID string                         `json:"saveSessionID"`
	SaveRevision  string                         `json:"saveRevision"`
	Parameters    gamecatalog.NetworkParamValues `json:"parameters"`
}

// regulationHeaderMarker is the confirmed start of the 0x10-byte header the
// regulation blob sits behind. Only these two bytes are confirmed native
// evidence, so only they are matched: the platform files use the marker to prove
// they are looking at UserData11 and not at arbitrary trailing bytes.
var regulationHeaderMarker = []byte{0x20, 0x47}

// GetNetworkSettings returns the 22 network parameters stored in the UserData11
// regulation of an existing session. Like every other SaveEngine getter it reads
// the session's private snapshot through the codec only: it opens no file,
// writes nothing, changes no session and returns no snapshot byte. It reads no
// GameCatalog either — gamecatalog.NetworkParamValues is used as the shared
// typed model and nothing else.
//
// saveSessionID is matched exactly. It is never trimmed, normalised or guessed,
// so an empty, unknown or already closed identifier is rejected instead of
// resolving to a session.
//
// The values are reported exactly as the save stores them. They are never
// compared against a preset, clamped into a documented range, defaulted or
// repaired, because this getter reports the state of the loaded save rather than
// the state the game would accept. Decoding, in contrast, is strict: a missing,
// truncated, malformed or unsupported UserData11 — an absent regulation header,
// an unaligned or undecryptable blob, a foreign DCX variant, an archive without
// NetworkParam.param or a row too short for the 22 values — is a hard error, and
// no partial or guessed parameter set is ever returned.
func (engine *Engine) GetNetworkSettings(saveSessionID string) (NetworkSettingsSnapshot, error) {
	if saveSessionID == "" {
		return NetworkSettingsSnapshot{}, apperror.MissingField("saveSessionID")
	}

	engine.mutex.Lock()
	defer engine.mutex.Unlock()
	loaded, exists := engine.sessions[saveSessionID]
	if !exists {
		return NetworkSettingsSnapshot{}, apperror.UnknownSaveSession(saveSessionID)
	}

	var blob []byte
	var err error
	switch loaded.session.platform {
	case PlatformPS4:
		blob, err = ps4RegulationBlob(loaded.snapshot)
	default:
		blob, err = pcRegulationBlob(loaded.snapshot)
	}
	if err != nil {
		return NetworkSettingsSnapshot{}, err
	}

	row, err := networkParamRow(blob)
	if err != nil {
		return NetworkSettingsSnapshot{}, err
	}
	return NetworkSettingsSnapshot{
		SaveSessionID: saveSessionID,
		SaveRevision:  loaded.session.revisionString(),
		Parameters:    decodeNetworkParams(row),
	}, nil
}

// networkParamRow turns the encrypted regulation blob into the row 0 data of
// NetworkParam.param. Every step is fail-closed: nothing is retried with a
// second key, a second format or a second position.
func networkParamRow(blob []byte) ([]byte, error) {
	archive, err := decryptRegulation(blob)
	if err != nil {
		return nil, err
	}
	bnd4, err := decompressRegulation(archive)
	if err != nil {
		return nil, err
	}
	return findNetworkParamRow(bnd4)
}

// decryptRegulation decrypts the blob with its own initialisation vector and
// proves the result is a DCX archive of a supported compression format. A blob
// that is too short, not block aligned or does not decrypt into a DCX header is
// rejected rather than inspected further.
func decryptRegulation(blob []byte) ([]byte, error) {
	if len(blob) <= regulationIVSize {
		return nil, fmt.Errorf("UserData11 holds no regulation blob (%d bytes)", len(blob))
	}
	iv, ciphertext := blob[:regulationIVSize], blob[regulationIVSize:]
	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("regulation ciphertext of %d bytes is not block aligned", len(ciphertext))
	}

	block, err := aes.NewCipher(regulationKey)
	if err != nil {
		return nil, fmt.Errorf("cannot prepare the regulation cipher: %w", err)
	}
	archive := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(archive, ciphertext)

	if len(archive) < dcxHeaderSize || string(archive[:len(dcxMagic)]) != dcxMagic {
		return nil, errors.New("the decrypted regulation is not a DCX archive")
	}
	return archive, nil
}

// decompressRegulation decompresses the DCX payload. Only the two confirmed
// compression formats are accepted, the declared decompressed size is never used
// as an allocation size, and a stream longer than the accepted maximum is
// rejected instead of being buffered.
func decompressRegulation(archive []byte) ([]byte, error) {
	format := string(archive[dcxFormatOffset : dcxFormatOffset+dcxFormatSize])
	declared := int64(binary.BigEndian.Uint32(archive[dcxCompressedSizeOffset:]))
	payload := archive[dcxHeaderSize:]
	if declared >= 0 && declared < int64(len(payload)) {
		payload = payload[:declared]
	}

	var stream io.Reader
	switch format {
	case dcxFormatZSTD:
		decoder, err := zstd.NewReader(bytes.NewReader(payload))
		if err != nil {
			return nil, fmt.Errorf("cannot read the ZSTD regulation: %w", err)
		}
		defer decoder.Close()
		stream = decoder
	case dcxFormatDFLT:
		if len(payload) <= dcxDeflateHeaderSize {
			return nil, errors.New("the DFLT regulation payload is too short")
		}
		reader := flate.NewReader(bytes.NewReader(payload[dcxDeflateHeaderSize:]))
		defer reader.Close()
		stream = reader
	default:
		return nil, fmt.Errorf("unsupported regulation compression format %q", format)
	}

	bnd4, err := io.ReadAll(io.LimitReader(stream, maxRegulationSize+1))
	if err != nil {
		return nil, fmt.Errorf("cannot decompress the regulation: %w", err)
	}
	if len(bnd4) > maxRegulationSize {
		return nil, fmt.Errorf("the decompressed regulation exceeds %d bytes", maxRegulationSize)
	}
	return bnd4, nil
}

// findNetworkParamRow locates NetworkParam.param inside the BND4 archive and
// returns the row 0 data of the parameter file. The entry is found by its name
// alone; no index, order or size is assumed, and an archive without that entry
// is an error rather than a fallback to another parameter file.
func findNetworkParamRow(bnd4 []byte) ([]byte, error) {
	row, _, err := locateNetworkParamRow(bnd4)
	return row, err
}

// locateNetworkParamRow also reports the row's absolute BND4 offset. The
// setter needs that offset to replace only the affected native ZSTD blocks on
// PS4; the getter deliberately discards it.
func locateNetworkParamRow(bnd4 []byte) ([]byte, int, error) {
	if len(bnd4) < bnd4EntryTableOffset || string(bnd4[:len(bnd4Magic)]) != bnd4Magic {
		return nil, 0, errors.New("the decompressed regulation is not a BND4 archive")
	}
	files := int(binary.LittleEndian.Uint32(bnd4[bnd4FileCountOffset:]))
	name := encodeUTF16LE(networkParamName)

	for index := 0; index < files; index++ {
		entry := bnd4EntryTableOffset + index*bnd4EntrySize
		if entry < 0 || entry+bnd4EntrySize > len(bnd4) {
			break
		}
		nameAt := int64(binary.LittleEndian.Uint32(bnd4[entry+bnd4EntryNameOffset:]))
		if nameAt <= 0 || nameAt >= int64(len(bnd4)) || !hasUTF16Suffix(bnd4, int(nameAt), name) {
			continue
		}

		size := int64(binary.LittleEndian.Uint64(bnd4[entry+bnd4EntrySizeOffset:]))
		dataAt := int64(binary.LittleEndian.Uint32(bnd4[entry+bnd4EntryDataOffset:]))
		if size < 0 || dataAt < 0 || dataAt+size > int64(len(bnd4)) {
			return nil, 0, errors.New("NetworkParam.param reaches past the regulation archive")
		}
		row, rowAt, err := parseNetworkParamRow(bnd4[dataAt : dataAt+size])
		if err != nil {
			return nil, 0, err
		}
		return row, int(dataAt) + rowAt, nil
	}
	return nil, 0, fmt.Errorf("the regulation archive holds no %s (%d entries)", networkParamName, files)
}

// parseNetworkParamRow reads the parameter header far enough to reach the data
// of row 0. Only the confirmed long-data-offset layout is accepted, and a row
// that does not provide the full 0x24C bytes the 22 values live in is rejected
// instead of being read partially.
func parseNetworkParamRow(param []byte) ([]byte, int, error) {
	if len(param) < networkParamHeaderSize {
		return nil, 0, fmt.Errorf("%s is too small (%d bytes)", networkParamName, len(param))
	}
	if param[networkParamFormatOffset]&networkParamLongDataFlag == 0 {
		return nil, 0, fmt.Errorf("%s uses the unsupported format flags 0x%02X",
			networkParamName, param[networkParamFormatOffset])
	}
	rowAt := int64(binary.LittleEndian.Uint64(param[networkParamRow0Offset:]))
	if rowAt <= 0 || rowAt+networkParamRowSize > int64(len(param)) {
		return nil, 0, fmt.Errorf("%s declares the unusable row offset 0x%X", networkParamName, rowAt)
	}
	return param[rowAt : rowAt+networkParamRowSize], int(rowAt), nil
}

// decodeNetworkParams reads the 22 values from the row data. The row is already
// known to be long enough, every value is taken exactly as stored, and no value
// is validated, clamped or replaced.
func decodeNetworkParams(row []byte) gamecatalog.NetworkParamValues {
	return gamecatalog.NetworkParamValues{
		MaxBreakInTargetListCount:     networkInt32(row, networkMaxBreakInTargetListCount),
		BreakInRequestIntervalTimeSec: networkFloat32(row, networkBreakInRequestIntervalTimeSec),
		BreakInRequestTimeOutSec:      networkFloat32(row, networkBreakInRequestTimeOutSec),
		BreakInRequestAreaCount:       networkInt32(row, networkBreakInRequestAreaCount),

		SummonTimeoutTime: networkFloat32(row, networkSummonTimeoutTime),

		ReloadSignIntervalTime2: networkFloat32(row, networkReloadSignIntervalTime2),
		ReloadSignTotalCount:    networkInt32(row, networkReloadSignTotalCount),
		ReloadSignCellCount:     networkInt32(row, networkReloadSignCellCount),
		UpdateSignIntervalTime:  networkFloat32(row, networkUpdateSignIntervalTime),
		SingGetMax:              networkInt32(row, networkSingGetMax),
		SignDownloadSpan:        networkFloat32(row, networkSignDownloadSpan),
		SignUpdateSpan:          networkFloat32(row, networkSignUpdateSpan),

		ReloadVisitListCoolTime:   networkFloat32(row, networkReloadVisitListCoolTime),
		MaxCoopBlueSummonCount:    networkInt32(row, networkMaxCoopBlueSummonCount),
		MaxVisitListCount:         networkInt32(row, networkMaxVisitListCount),
		ReloadSearchCoopBlueMin:   networkFloat32(row, networkReloadSearchCoopBlueMin),
		ReloadSearchCoopBlueMax:   networkFloat32(row, networkReloadSearchCoopBlueMax),
		AllAreaSearchRateCoopBlue: int32(row[networkAllAreaSearchRateCoopBlue]),
		AllAreaSearchRateVsBlue:   int32(row[networkAllAreaSearchRateVsBlue]),

		VisitorListMax:      networkInt32(row, networkVisitorListMax),
		VisitorTimeOutTime:  networkFloat32(row, networkVisitorTimeOutTime),
		VisitorDownloadSpan: networkFloat32(row, networkVisitorDownloadSpan),
	}
}

func networkInt32(row []byte, offset int) int32 {
	return int32(binary.LittleEndian.Uint32(row[offset:]))
}

func networkFloat32(row []byte, offset int) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(row[offset:]))
}

// encodeUTF16LE renders an archive entry name the way BND4 stores it.
func encodeUTF16LE(value string) []byte {
	units := utf16.Encode([]rune(value))
	encoded := make([]byte, len(units)*2)
	for index, unit := range units {
		binary.LittleEndian.PutUint16(encoded[index*2:], unit)
	}
	return encoded
}

// hasUTF16Suffix reports whether the NUL-terminated UTF-16LE name at offset ends
// with suffix. Archive names carry a full path, so the file name is matched as a
// suffix instead of comparing the whole stored string.
func hasUTF16Suffix(data []byte, offset int, suffix []byte) bool {
	end := offset
	for end+1 < len(data) && (data[end] != 0 || data[end+1] != 0) {
		end += 2
	}
	name := data[offset:end]
	if len(name) < len(suffix) {
		return false
	}
	return bytes.Equal(name[len(name)-len(suffix):], suffix)
}

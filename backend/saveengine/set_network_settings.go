package saveengine

import (
	"bytes"
	"compress/flate"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/klauspost/compress/zstd"
	"github.com/oisis/EldenRing-SaveForge/backend/gamecatalog"
)

const networkZSTDBlockSize = 64 * 1024

// SetNetworkSettingsResult reports one committed complete network-parameter
// assignment. The embedded receipt is the one the shared writer produced, so it
// names the public entry point that was called; the settings are the values
// that entry point committed.
type SetNetworkSettingsResult struct {
	MutationReceipt
	NetworkSettings gamecatalog.NetworkParamValues `json:"networkSettings"`
}

// SetNetworkSettings replaces all 22 supported NetworkParam values in the
// loaded session. The complete replacement is built and decoded successfully
// before the private snapshot is touched.
func (engine *Engine) SetNetworkSettings(
	saveSessionID string,
	networkSettings gamecatalog.NetworkParamValues,
	expectedRevision string,
) (SetNetworkSettingsResult, error) {
	return engine.setNetworkSettings(
		saveSessionID, networkSettings, expectedRevision, kindSetNetworkSettings)
}

// ApplyNetworkPreset is SetNetworkSettings for a preset the caller resolved. It
// exists so the receipt reports the preset operation instead of the plain
// settings one; it shares every rule and byte of the writer below.
func (engine *Engine) ApplyNetworkPreset(
	saveSessionID string,
	networkSettings gamecatalog.NetworkParamValues,
	expectedRevision string,
) (SetNetworkSettingsResult, error) {
	return engine.setNetworkSettings(
		saveSessionID, networkSettings, expectedRevision, kindApplyNetworkPreset)
}

// setNetworkSettings is the one writer behind both public entry points.
// operationKind is chosen by those entry points and never by a caller outside
// this package.
func (engine *Engine) setNetworkSettings(
	saveSessionID string,
	networkSettings gamecatalog.NetworkParamValues,
	expectedRevision string,
	operationKind string,
) (SetNetworkSettingsResult, error) {
	if !isCanonicalRevision(expectedRevision) {
		return SetNetworkSettingsResult{}, fmt.Errorf(
			"expectedRevision must be a canonical decimal saveRevision; got %q", expectedRevision)
	}
	if err := validateNetworkSettings(networkSettings); err != nil {
		return SetNetworkSettingsResult{}, err
	}

	committed, err := engine.commitRevision(saveSessionID, operationKind, func(loaded *loadedSave) error {
		current := loaded.session.revisionString()
		if expectedRevision != current {
			return fmt.Errorf(
				"expectedRevision %q does not match the current saveRevision %q",
				expectedRevision, current)
		}

		userDataAt, blobAt, err := networkUserData11Layout(loaded)
		if err != nil {
			return err
		}
		before, err := loaded.snapshot.readAt(userDataAt, int(loaded.snapshot.length()-userDataAt))
		if err != nil {
			return fmt.Errorf("cannot read UserData11: %w", err)
		}
		patched, err := patchNetworkUserData11(before, blobAt, loaded.session.platform, networkSettings)
		if err != nil {
			return err
		}
		if err := loaded.snapshot.writeAt(userDataAt, patched); err != nil {
			return fmt.Errorf("cannot write UserData11: %w", err)
		}

		written, err := loaded.snapshot.readAt(userDataAt, len(patched))
		if err == nil {
			row, decodeErr := networkParamRow(written[blobAt:])
			if decodeErr == nil && decodeNetworkParams(row) == networkSettings {
				return nil
			}
		}
		if rollback := loaded.snapshot.writeAt(userDataAt, before); rollback != nil {
			return fmt.Errorf("network settings could not be verified and UserData11 could not be restored: %w", rollback)
		}
		return errors.New("network settings mutation could not be verified; the save is unchanged")
	})
	if err != nil {
		return SetNetworkSettingsResult{}, err
	}

	return SetNetworkSettingsResult{
		MutationReceipt: committed,
		NetworkSettings: networkSettings,
	}, nil
}

func networkUserData11Layout(loaded *loadedSave) (int64, int, error) {
	switch loaded.session.platform {
	case PlatformPC:
		if _, err := pcRegulationBlob(loaded.snapshot); err != nil {
			return 0, 0, err
		}
		return pcUserData11Offset, pcUserData11MD5Size + pcUserData11HeaderSize, nil
	case PlatformPS4:
		if _, err := ps4RegulationBlob(loaded.snapshot); err != nil {
			return 0, 0, err
		}
		return ps4UserData11Offset, ps4UserData11HeaderSize, nil
	default:
		return 0, 0, fmt.Errorf("unsupported save platform %q", loaded.session.platform)
	}
}

func patchNetworkUserData11(
	userData11 []byte,
	blobAt int,
	platform Platform,
	values gamecatalog.NetworkParamValues,
) ([]byte, error) {
	if blobAt < 0 || blobAt >= len(userData11) {
		return nil, errors.New("UserData11 carries no regulation blob")
	}
	blob := userData11[blobAt:]
	archive, err := decryptRegulation(blob)
	if err != nil {
		return nil, err
	}
	bnd4, err := decompressRegulation(archive)
	if err != nil {
		return nil, err
	}
	row, rowAt, err := locateNetworkParamRow(bnd4)
	if err != nil {
		return nil, err
	}
	encodeNetworkParams(row, values)

	var replacement []byte
	format := string(archive[dcxFormatOffset : dcxFormatOffset+dcxFormatSize])
	if platform == PlatformPS4 && format == dcxFormatZSTD {
		replacement, err = patchNetworkZSTD(
			archive,
			bnd4,
			rowAt+networkSummonTimeoutTime,
			rowAt+networkVisitorDownloadSpan+3,
		)
	} else {
		replacement, err = compressNetworkDCX(archive, bnd4, format)
	}
	if err != nil {
		return nil, err
	}

	ciphertextSize := len(blob) - regulationIVSize
	if len(replacement) > ciphertextSize {
		return nil, fmt.Errorf(
			"patched regulation blob of %d bytes exceeds its %d-byte capacity",
			len(replacement), ciphertextSize)
	}
	encrypted, err := encryptNetworkRegulation(replacement, blob[:regulationIVSize], ciphertextSize)
	if err != nil {
		return nil, err
	}

	result := append([]byte(nil), userData11...)
	copy(result[blobAt:], encrypted)
	if platform == PlatformPC {
		sum := md5.Sum(result[pcUserData11MD5Size:])
		copy(result[:pcUserData11MD5Size], sum[:])
	}
	return result, nil
}

func validateNetworkSettings(values gamecatalog.NetworkParamValues) error {
	integers := []struct {
		name    string
		value   int32
		minimum int32
		maximum int32
	}{
		{"maxBreakInTargetListCount", values.MaxBreakInTargetListCount, 1, 20},
		{"breakInRequestAreaCount", values.BreakInRequestAreaCount, 1, 50},
		{"reloadSignTotalCount", values.ReloadSignTotalCount, 1, 128},
		{"reloadSignCellCount", values.ReloadSignCellCount, 1, 99},
		{"singGetMax", values.SingGetMax, 1, 128},
		{"maxCoopBlueSummonCount", values.MaxCoopBlueSummonCount, 1, 10},
		{"maxVisitListCount", values.MaxVisitListCount, 1, 50},
		{"allAreaSearchRateCoopBlue", values.AllAreaSearchRateCoopBlue, 0, 100},
		{"allAreaSearchRateVsBlue", values.AllAreaSearchRateVsBlue, 0, 100},
		{"visitorListMax", values.VisitorListMax, 1, 100},
	}
	for _, field := range integers {
		if field.value < field.minimum || field.value > field.maximum {
			return fmt.Errorf("%s must be %d..%d; got %d",
				field.name, field.minimum, field.maximum, field.value)
		}
	}

	floats := []struct {
		name    string
		value   float32
		minimum float32
		maximum float32
	}{
		{"breakInRequestIntervalTimeSec", values.BreakInRequestIntervalTimeSec, 2, 30},
		{"breakInRequestTimeOutSec", values.BreakInRequestTimeOutSec, 3, 20},
		{"summonTimeoutTime", values.SummonTimeoutTime, 1, 999},
		{"reloadSignIntervalTime2", values.ReloadSignIntervalTime2, 1, 1000},
		{"updateSignIntervalTime", values.UpdateSignIntervalTime, 1, 1000},
		{"signDownloadSpan", values.SignDownloadSpan, 1, 1000},
		{"signUpdateSpan", values.SignUpdateSpan, 1, 1000},
		{"reloadVisitListCoolTime", values.ReloadVisitListCoolTime, 1, 1000},
		{"reloadSearchCoopBlueMin", values.ReloadSearchCoopBlueMin, 1, 999},
		{"reloadSearchCoopBlueMax", values.ReloadSearchCoopBlueMax, 1, 999},
		{"visitorTimeOutTime", values.VisitorTimeOutTime, 1, 600},
		{"visitorDownloadSpan", values.VisitorDownloadSpan, 1, 600},
	}
	for _, field := range floats {
		if math.IsNaN(float64(field.value)) || math.IsInf(float64(field.value), 0) ||
			field.value < field.minimum || field.value > field.maximum {
			return fmt.Errorf("%s must be finite and within %g..%g; got %g",
				field.name, field.minimum, field.maximum, field.value)
		}
	}

	if values.ReloadSignCellCount > values.ReloadSignTotalCount {
		return fmt.Errorf("reloadSignCellCount %d exceeds reloadSignTotalCount %d",
			values.ReloadSignCellCount, values.ReloadSignTotalCount)
	}
	if values.ReloadSignTotalCount > values.SingGetMax {
		return fmt.Errorf("reloadSignTotalCount %d exceeds singGetMax %d",
			values.ReloadSignTotalCount, values.SingGetMax)
	}
	if values.ReloadSearchCoopBlueMin > values.ReloadSearchCoopBlueMax {
		return fmt.Errorf("reloadSearchCoopBlueMin %g exceeds reloadSearchCoopBlueMax %g",
			values.ReloadSearchCoopBlueMin, values.ReloadSearchCoopBlueMax)
	}
	return nil
}

func encodeNetworkParams(row []byte, values gamecatalog.NetworkParamValues) {
	putInt := func(offset int, value int32) {
		binary.LittleEndian.PutUint32(row[offset:], uint32(value))
	}
	putFloat := func(offset int, value float32) {
		binary.LittleEndian.PutUint32(row[offset:], math.Float32bits(value))
	}

	putInt(networkMaxBreakInTargetListCount, values.MaxBreakInTargetListCount)
	putFloat(networkBreakInRequestIntervalTimeSec, values.BreakInRequestIntervalTimeSec)
	putFloat(networkBreakInRequestTimeOutSec, values.BreakInRequestTimeOutSec)
	putInt(networkBreakInRequestAreaCount, values.BreakInRequestAreaCount)
	putFloat(networkSummonTimeoutTime, values.SummonTimeoutTime)
	putFloat(networkReloadSignIntervalTime2, values.ReloadSignIntervalTime2)
	putInt(networkReloadSignTotalCount, values.ReloadSignTotalCount)
	putInt(networkReloadSignCellCount, values.ReloadSignCellCount)
	putFloat(networkUpdateSignIntervalTime, values.UpdateSignIntervalTime)
	putInt(networkSingGetMax, values.SingGetMax)
	putFloat(networkSignDownloadSpan, values.SignDownloadSpan)
	putFloat(networkSignUpdateSpan, values.SignUpdateSpan)
	putFloat(networkReloadVisitListCoolTime, values.ReloadVisitListCoolTime)
	putInt(networkMaxCoopBlueSummonCount, values.MaxCoopBlueSummonCount)
	putInt(networkMaxVisitListCount, values.MaxVisitListCount)
	putFloat(networkReloadSearchCoopBlueMin, values.ReloadSearchCoopBlueMin)
	putFloat(networkReloadSearchCoopBlueMax, values.ReloadSearchCoopBlueMax)
	row[networkAllAreaSearchRateCoopBlue] = byte(values.AllAreaSearchRateCoopBlue)
	row[networkAllAreaSearchRateVsBlue] = byte(values.AllAreaSearchRateVsBlue)
	putInt(networkVisitorListMax, values.VisitorListMax)
	putFloat(networkVisitorTimeOutTime, values.VisitorTimeOutTime)
	putFloat(networkVisitorDownloadSpan, values.VisitorDownloadSpan)
}

func compressNetworkDCX(original, bnd4 []byte, format string) ([]byte, error) {
	var payload []byte
	switch format {
	case dcxFormatDFLT:
		var buffer bytes.Buffer
		buffer.Write([]byte{0x78, 0x9C})
		writer, err := flate.NewWriter(&buffer, flate.DefaultCompression)
		if err != nil {
			return nil, fmt.Errorf("cannot prepare DFLT compression: %w", err)
		}
		if _, err := writer.Write(bnd4); err != nil {
			return nil, fmt.Errorf("cannot compress DFLT regulation: %w", err)
		}
		if err := writer.Close(); err != nil {
			return nil, fmt.Errorf("cannot finish DFLT regulation: %w", err)
		}
		payload = buffer.Bytes()
	case dcxFormatZSTD:
		encoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedDefault))
		if err != nil {
			return nil, fmt.Errorf("cannot prepare ZSTD compression: %w", err)
		}
		payload = encoder.EncodeAll(bnd4, nil)
		encoder.Close()
	default:
		return nil, fmt.Errorf("unsupported regulation compression format %q", format)
	}

	archive := make([]byte, dcxHeaderSize, dcxHeaderSize+len(payload))
	copy(archive, original[:dcxHeaderSize])
	binary.BigEndian.PutUint32(archive[dcxDecompressedSizeOffset:], uint32(len(bnd4)))
	binary.BigEndian.PutUint32(archive[dcxCompressedSizeOffset:], uint32(len(payload)))
	return append(archive, payload...), nil
}

func encryptNetworkRegulation(archive, iv []byte, ciphertextSize int) ([]byte, error) {
	if len(iv) != aes.BlockSize || ciphertextSize < 0 || ciphertextSize%aes.BlockSize != 0 {
		return nil, errors.New("the regulation encryption layout is invalid")
	}
	plaintext := make([]byte, ciphertextSize)
	copy(plaintext, archive)
	block, err := aes.NewCipher(regulationKey)
	if err != nil {
		return nil, fmt.Errorf("cannot prepare the regulation cipher: %w", err)
	}
	ciphertext := make([]byte, ciphertextSize)
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, plaintext)
	result := append([]byte(nil), iv...)
	return append(result, ciphertext...), nil
}

type networkZSTDBlock struct {
	start  int
	end    int
	typeID int
	last   bool
}

// patchNetworkZSTD preserves the native PS4 ZSTD frame and every block outside
// the field range. Recompressing the complete frame is known to produce saves
// rejected by the console.
func patchNetworkZSTD(archive, bnd4 []byte, first, last int) ([]byte, error) {
	compressedSize := int(binary.BigEndian.Uint32(archive[dcxCompressedSizeOffset:]))
	if compressedSize < 0 || dcxHeaderSize+compressedSize > len(archive) {
		return nil, errors.New("the ZSTD payload reaches past the DCX archive")
	}
	stream := archive[dcxHeaderSize : dcxHeaderSize+compressedSize]
	firstBlock, lastBlock := first/networkZSTDBlockSize, last/networkZSTDBlockSize
	blocks, err := walkNetworkZSTDBlocks(stream, lastBlock+2)
	if err != nil {
		return nil, err
	}
	if firstBlock < 0 || lastBlock >= len(blocks) {
		return nil, fmt.Errorf("the ZSTD stream has no block %d", lastBlock)
	}
	for index := firstBlock; index <= lastBlock; index++ {
		if blocks[index].typeID != 0 && blocks[index].typeID != 2 {
			return nil, fmt.Errorf("ZSTD block %d uses unsupported type %d", index, blocks[index].typeID)
		}
	}

	replaceThrough := lastBlock
	if next := lastBlock + 1; next < len(blocks) && blocks[next].typeID == 2 {
		payloadAt := blocks[next].start + 3
		if payloadAt < len(stream) && stream[payloadAt]&0x03 == 3 {
			replaceThrough = next
		}
	}

	var patched bytes.Buffer
	patched.Write(stream[:blocks[firstBlock].start])
	for index := firstBlock; index <= replaceThrough; index++ {
		start := index * networkZSTDBlockSize
		end := start + networkZSTDBlockSize
		if end > len(bnd4) {
			end = len(bnd4)
		}
		if start >= end {
			return nil, fmt.Errorf("ZSTD block %d exceeds the decompressed archive", index)
		}
		patched.Write(networkZSTDRawHeader(end-start, blocks[index].last))
		patched.Write(bnd4[start:end])
	}
	patched.Write(stream[blocks[replaceThrough].end:])

	result := make([]byte, dcxHeaderSize, dcxHeaderSize+patched.Len())
	copy(result, archive[:dcxHeaderSize])
	binary.BigEndian.PutUint32(result[dcxCompressedSizeOffset:], uint32(patched.Len()))
	return append(result, patched.Bytes()...), nil
}

func walkNetworkZSTDBlocks(stream []byte, limit int) ([]networkZSTDBlock, error) {
	if len(stream) < 6 || !bytes.Equal(stream[:4], []byte{0x28, 0xB5, 0x2F, 0xFD}) {
		return nil, errors.New("the regulation does not carry a valid ZSTD frame")
	}
	header := stream[4]
	if header&0x04 != 0 {
		return nil, errors.New("the PS4 ZSTD frame unexpectedly carries a checksum")
	}
	singleSegment := header&0x20 != 0
	dictionarySizes := [4]int{0, 1, 2, 4}
	contentSizes := [4]int{0, 2, 4, 8}
	position := 5
	if !singleSegment {
		position++
	}
	position += dictionarySizes[int(header&0x03)]
	contentFlag := int(header >> 6)
	if singleSegment && contentFlag == 0 {
		position++
	} else {
		position += contentSizes[contentFlag]
	}
	if position > len(stream) {
		return nil, errors.New("the ZSTD frame header is truncated")
	}

	blocks := make([]networkZSTDBlock, 0, limit)
	for position+3 <= len(stream) && len(blocks) < limit {
		raw := int(stream[position]) | int(stream[position+1])<<8 | int(stream[position+2])<<16
		block := networkZSTDBlock{
			start:  position,
			typeID: (raw >> 1) & 3,
			last:   raw&1 != 0,
		}
		position += 3
		size := raw >> 3
		switch block.typeID {
		case 0, 2:
			position += size
		case 1:
			position++
		default:
			return nil, fmt.Errorf("ZSTD block at 0x%X uses the reserved type", block.start)
		}
		if position > len(stream) {
			return nil, errors.New("a ZSTD block reaches past the frame")
		}
		block.end = position
		blocks = append(blocks, block)
		if block.last {
			break
		}
	}
	return blocks, nil
}

func networkZSTDRawHeader(size int, last bool) []byte {
	header := uint32(size) << 3
	if last {
		header++
	}
	return []byte{byte(header), byte(header >> 8), byte(header >> 16)}
}

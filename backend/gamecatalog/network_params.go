package gamecatalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
)

// NetworkParamsPath is the catalog-relative path of the network parameter file.
// It is the single source of truth for the backend network presets.
const NetworkParamsPath = "regulation/network_params.json"

// DefaultNetworkPresetID is the only identifier the default preset may carry.
const DefaultNetworkPresetID = "vanilla"

// NetworkParamValues is the complete network parameter set of one preset. It
// holds scalars only, so copying the struct copies the whole parameter set and
// no caller can reach stored catalog state through it.
type NetworkParamValues struct {
	MaxBreakInTargetListCount     int32   `json:"maxBreakInTargetListCount"`
	BreakInRequestIntervalTimeSec float32 `json:"breakInRequestIntervalTimeSec"`
	BreakInRequestTimeOutSec      float32 `json:"breakInRequestTimeOutSec"`
	BreakInRequestAreaCount       int32   `json:"breakInRequestAreaCount"`

	SummonTimeoutTime float32 `json:"summonTimeoutTime"`

	ReloadSignIntervalTime2 float32 `json:"reloadSignIntervalTime2"`
	ReloadSignTotalCount    int32   `json:"reloadSignTotalCount"`
	ReloadSignCellCount     int32   `json:"reloadSignCellCount"`
	UpdateSignIntervalTime  float32 `json:"updateSignIntervalTime"`
	SingGetMax              int32   `json:"singGetMax"`
	SignDownloadSpan        float32 `json:"signDownloadSpan"`
	SignUpdateSpan          float32 `json:"signUpdateSpan"`

	ReloadVisitListCoolTime   float32 `json:"reloadVisitListCoolTime"`
	MaxCoopBlueSummonCount    int32   `json:"maxCoopBlueSummonCount"`
	MaxVisitListCount         int32   `json:"maxVisitListCount"`
	ReloadSearchCoopBlueMin   float32 `json:"reloadSearchCoopBlueMin"`
	ReloadSearchCoopBlueMax   float32 `json:"reloadSearchCoopBlueMax"`
	AllAreaSearchRateCoopBlue int32   `json:"allAreaSearchRateCoopBlue"`
	AllAreaSearchRateVsBlue   int32   `json:"allAreaSearchRateVsBlue"`

	VisitorListMax      int32   `json:"visitorListMax"`
	VisitorTimeOutTime  float32 `json:"visitorTimeOutTime"`
	VisitorDownloadSpan float32 `json:"visitorDownloadSpan"`
}

// NetworkPreset is one stored preset: its stable identifier and the full
// parameter set it stands for.
type NetworkPreset struct {
	ID         string             `json:"id"`
	Parameters NetworkParamValues `json:"parameters"`
}

// networkParamsFile is the on-disk shape of NetworkParamsPath: the default
// preset and the remaining presets, in file order.
type networkParamsFile struct {
	Default NetworkPreset   `json:"default"`
	Presets []NetworkPreset `json:"presets"`
}

// LoadNetworkParams reads and validates NetworkParamsPath from the catalog data
// filesystem. The default preset comes first, the remaining presets keep their
// file order, and an absent, malformed, or inconsistent file is an error instead
// of a partially populated catalog.
func LoadNetworkParams(catalogFS fs.FS) ([]NetworkPreset, error) {
	content, err := fs.ReadFile(catalogFS, NetworkParamsPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", NetworkParamsPath, err)
	}

	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	var file networkParamsFile
	if err := decoder.Decode(&file); err != nil {
		return nil, fmt.Errorf("decode %s: %w", NetworkParamsPath, err)
	}
	// The file must hold exactly one document: a second value, or any trailing
	// data that is not whitespace, would silently be ignored otherwise.
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("decode %s: multiple JSON values are not allowed", NetworkParamsPath)
		}
		return nil, fmt.Errorf("decode %s: trailing JSON data: %w", NetworkParamsPath, err)
	}

	presets := append([]NetworkPreset{file.Default}, file.Presets...)
	if err := validateNetworkPresets(presets); err != nil {
		return nil, fmt.Errorf("%s: %w", NetworkParamsPath, err)
	}
	return presets, nil
}

// validateNetworkPresets is the single place that decides whether a preset list
// may enter the catalog, so the loader and the constructor reject the same data.
func validateNetworkPresets(presets []NetworkPreset) error {
	if len(presets) == 0 || presets[0].ID == "" {
		return fmt.Errorf("the default preset is required")
	}
	if presets[0].ID != DefaultNetworkPresetID {
		return fmt.Errorf(
			"the default preset ID must be %q; got %q",
			DefaultNetworkPresetID,
			presets[0].ID,
		)
	}
	seen := make(map[string]struct{}, len(presets))
	for index, preset := range presets {
		if preset.ID == "" {
			return fmt.Errorf("preset %d: ID must not be empty", index)
		}
		if _, duplicate := seen[preset.ID]; duplicate {
			return fmt.Errorf("preset %d: duplicate preset ID %q", index, preset.ID)
		}
		seen[preset.ID] = struct{}{}
	}
	return nil
}

// NetworkPresets returns the stored presets in catalog order as an independent
// copy, so a caller can never change the catalog content. A catalog built
// without network parameters reports that instead of returning an empty list.
func (catalog *Catalog) NetworkPresets() ([]NetworkPreset, error) {
	if len(catalog.networkPresets) == 0 {
		return nil, fmt.Errorf("network parameters are not loaded")
	}
	// ponytail: NetworkPreset holds scalars only, so a slice copy is a deep copy.
	return append([]NetworkPreset(nil), catalog.networkPresets...), nil
}

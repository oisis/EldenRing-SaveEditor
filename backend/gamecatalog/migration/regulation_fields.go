package migration

import (
	"fmt"
	"strconv"
)

func regulationString(row ParameterRow, field string) (string, error) {
	value, exists := row.Field(field)
	if !exists {
		return "", fmt.Errorf("row %d has no %s field", row.RowID, field)
	}
	return value, nil
}

func regulationInt32(row ParameterRow, field string) (int32, error) {
	raw, err := regulationString(row, field)
	if err != nil {
		return 0, err
	}
	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("row %d %s=%q is not int32", row.RowID, field, raw)
	}
	return int32(value), nil
}

func regulationUint32(row ParameterRow, field string) (uint32, error) {
	raw, err := regulationString(row, field)
	if err != nil {
		return 0, err
	}
	value, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("row %d %s=%q is not uint32", row.RowID, field, raw)
	}
	return uint32(value), nil
}

func regulationUint16(row ParameterRow, field string) (uint16, error) {
	value, err := regulationUint32(row, field)
	if err != nil || value > 0xFFFF {
		if err != nil {
			return 0, err
		}
		return 0, fmt.Errorf("row %d %s=%d exceeds uint16", row.RowID, field, value)
	}
	return uint16(value), nil
}

func regulationUint8(row ParameterRow, field string) (uint8, error) {
	value, err := regulationUint32(row, field)
	if err != nil || value > 0xFF {
		if err != nil {
			return 0, err
		}
		return 0, fmt.Errorf("row %d %s=%d exceeds uint8", row.RowID, field, value)
	}
	return uint8(value), nil
}

func regulationFloat64(row ParameterRow, field string) (float64, error) {
	raw, err := regulationString(row, field)
	if err != nil {
		return 0, err
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("row %d %s=%q is not float64", row.RowID, field, raw)
	}
	return value, nil
}

func regulationBool(row ParameterRow, field string) (bool, error) {
	value, err := regulationUint8(row, field)
	if err != nil {
		return false, err
	}
	switch value {
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, fmt.Errorf("row %d %s=%d is not boolean", row.RowID, field, value)
	}
}

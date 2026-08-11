package dbviewer

import (
	"fmt"
	"strconv"
	"strings"
)

func parseGameID(value string) (uint32, error) {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(strings.ToLower(value), "0x")
	parsed, err := strconv.ParseUint(value, 16, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid game ID")
	}
	return uint32(parsed), nil
}

func formatGameID(value uint32) string {
	return fmt.Sprintf("0x%08X", value)
}

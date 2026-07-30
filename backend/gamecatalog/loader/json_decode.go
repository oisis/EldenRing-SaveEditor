package loader

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

func decodeJSON[T any](content []byte, destination *T) error {
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values are not allowed")
		}
		return fmt.Errorf("trailing JSON data: %w", err)
	}
	return nil
}

package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

// JSONMap is a map[string]any persisted as Postgres jsonb. Used for outbox
// params and template params_schema.
type JSONMap map[string]any

// Value implements driver.Valuer. A nil/empty map serialises to "{}" so the
// jsonb column never holds NULL when we explicitly mean "empty".
func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(map[string]any(m))
}

// Scan implements sql.Scanner. Accepts []byte, string, or nil.
func (m *JSONMap) Scan(src any) error {
	if src == nil {
		*m = JSONMap{}
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("JSONMap.Scan: unsupported type %T", src)
	}
	if len(data) == 0 {
		*m = JSONMap{}
		return nil
	}
	tmp := map[string]any{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return errors.New("JSONMap.Scan: " + err.Error())
	}
	*m = tmp
	return nil
}

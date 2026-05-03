package models_test

import (
	"database/sql/driver"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/hoshina-dev/copium/internal/models"
)

func TestJSONMap_ValueEmpty(t *testing.T) {
	var m models.JSONMap
	v, err := m.Value()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if v == nil {
		t.Fatal("nil JSONMap should serialise to a non-nil JSON value")
	}
	b, _ := v.([]byte)
	if string(b) != "{}" {
		t.Errorf("Value()=%q want {}", string(b))
	}
}

func TestJSONMap_ValueRoundTrip(t *testing.T) {
	m := models.JSONMap{"name": "alice", "n": float64(7)}
	v, err := m.Value()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	var back models.JSONMap
	if err := back.Scan(v); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !reflect.DeepEqual(m, back) {
		t.Errorf("roundtrip mismatch: got %v want %v", back, m)
	}
}

func TestJSONMap_ScanFromString(t *testing.T) {
	var m models.JSONMap
	if err := m.Scan(`{"a":1}`); err != nil {
		t.Fatalf("scan string: %v", err)
	}
	if m["a"].(float64) != 1 {
		t.Errorf("got %v", m)
	}
}

func TestJSONMap_ScanFromNil(t *testing.T) {
	var m models.JSONMap
	if err := m.Scan(nil); err != nil {
		t.Fatalf("scan nil: %v", err)
	}
	if len(m) != 0 {
		t.Errorf("nil scan should leave empty map; got %v", m)
	}
}

func TestJSONMap_ScanInvalidType(t *testing.T) {
	var m models.JSONMap
	if err := m.Scan(123); err == nil {
		t.Fatal("expected error scanning int")
	}
}

func TestJSONMap_JSONMarshal(t *testing.T) {
	m := models.JSONMap{"k": "v"}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"k":"v"}` {
		t.Errorf("got %s", b)
	}
}

// Ensure JSONMap satisfies driver.Valuer at compile time.
var _ driver.Valuer = (models.JSONMap)(nil)

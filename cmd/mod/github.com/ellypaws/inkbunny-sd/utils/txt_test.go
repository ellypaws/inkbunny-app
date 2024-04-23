package utils

import (
	_ "embed"
	"encoding/json"
	"testing"
)

//go:embed samples/multi-chunk.txt
var file []byte

func TestAutoSnep(t *testing.T) {
	params, err := AutoSnep(WithBytes(file))
	if err != nil {
		t.Error(err)
	}
	t.Logf("%+v", params)
}

func TestParseParams(t *testing.T) {
	params, err := AutoSnep(WithBytes(file))
	if err != nil {
		t.Error(err)
	}

	requests := ParseParams(params)
	if requests == nil {
		t.Error("no requests")
	}

	marshal, err := json.MarshalIndent(requests, "", "  ")
	if err != nil {
		t.Error(err)
	}

	t.Logf("%s", marshal)
}

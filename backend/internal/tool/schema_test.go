package tool

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestValidateArgumentsAcceptsValidObject(t *testing.T) {
	schema := json.RawMessage(`{
		"type":"object",
		"required":["title","priority"],
		"properties":{
			"title":{"type":"string"},
			"priority":{"type":"string","enum":["low","normal","high"]},
			"limit":{"type":"integer","minimum":1,"maximum":20},
			"tags":{"type":"array","items":{"type":"string"}}
		}
	}`)

	validated, err := validateArguments(schema, json.RawMessage(`{"title":"写测试","priority":"high","limit":3,"tags":["tool","test"]}`))
	if err != nil {
		t.Fatalf("expected valid arguments, got %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(validated, &result); err != nil {
		t.Fatalf("validated arguments should be json: %v", err)
	}
	if result["title"] != "写测试" {
		t.Fatalf("unexpected title: %#v", result["title"])
	}
}

func TestValidateArgumentsRejectsMissingRequired(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","required":["query"],"properties":{"query":{"type":"string"}}}`)
	_, err := validateArguments(schema, json.RawMessage(`{}`))
	if !errors.Is(err, ErrInvalidArguments) {
		t.Fatalf("expected ErrInvalidArguments, got %v", err)
	}
}

func TestValidateArgumentsRejectsWrongTypeEnumAndRange(t *testing.T) {
	schema := json.RawMessage(`{
		"type":"object",
		"properties":{
			"status":{"type":"string","enum":["open","done"]},
			"limit":{"type":"integer","minimum":1,"maximum":10},
			"args":{"type":"array","items":{"type":"string"}}
		}
	}`)

	cases := []json.RawMessage{
		json.RawMessage(`{"status":"missing"}`),
		json.RawMessage(`{"limit":11}`),
		json.RawMessage(`{"limit":"5"}`),
		json.RawMessage(`{"args":["ok",1]}`),
	}
	for _, tc := range cases {
		if _, err := validateArguments(schema, tc); !errors.Is(err, ErrInvalidArguments) {
			t.Fatalf("expected ErrInvalidArguments for %s, got %v", tc, err)
		}
	}
}

func TestValidateArgumentsRejectsNonObjectArguments(t *testing.T) {
	schema := json.RawMessage(`{"type":"object"}`)
	_, err := validateArguments(schema, json.RawMessage(`[]`))
	if !errors.Is(err, ErrInvalidArguments) {
		t.Fatalf("expected ErrInvalidArguments, got %v", err)
	}
}

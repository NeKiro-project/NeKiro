package contracts

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestDecodeRouterInstanceDirectoryV1(t *testing.T) {
	valid := []byte(`{"schemaVersion":"1","revision":"stack-1","targets":[]}`)
	document, err := DecodeRouterInstanceDirectoryV1(valid)
	if err != nil {
		t.Fatalf("valid document rejected: %v", err)
	}
	if document.Revision != "stack-1" || document.Targets == nil {
		t.Fatalf("document = %#v", document)
	}
	for name, payload := range map[string][]byte{
		"unknown version":  []byte(`{"schemaVersion":"2","revision":"stack-1","targets":[]}`),
		"unknown member":   []byte(`{"schemaVersion":"1","revision":"stack-1","targets":[],"fallback":true}`),
		"duplicate member": []byte(`{"schemaVersion":"1","revision":"stack-1","revision":"stack-2","targets":[]}`),
		"missing targets":  []byte(`{"schemaVersion":"1","revision":"stack-1"}`),
		"invalid target":   []byte(`{"schemaVersion":"1","revision":"stack-1","targets":[{"agentId":"not safe"}]}`),
		"trailing value":   []byte(`{"schemaVersion":"1","revision":"stack-1","targets":[]} {}`),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeRouterInstanceDirectoryV1(payload); err == nil {
				t.Fatal("invalid document accepted")
			}
		})
	}
}

func TestValidateRouterInstancePortNameV1(t *testing.T) {
	for _, test := range []struct {
		value string
		want  bool
	}{
		{value: "a2a", want: true},
		{value: "a2a-v1", want: true},
		{value: "a2a port", want: false},
		{value: "", want: false},
	} {
		t.Run(test.value, func(t *testing.T) {
			got := ValidateRouterInstancePortNameV1(test.value) == nil
			if got != test.want {
				t.Fatalf("valid=%v want=%v", got, test.want)
			}
		})
	}
}

func TestRouterInstanceDirectoryV1SchemaCompilesAndMatchesDecoder(t *testing.T) {
	data, err := fs.ReadFile(ContractFiles(), "schemas/router-instance-directory.v1.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	document, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	const schemaID = "https://schemas.nekiro.dev/router-instance-directory/v1"
	if err := compiler.AddResource(schemaID, document); err != nil {
		t.Fatal(err)
	}
	schema, err := compiler.Compile(schemaID)
	if err != nil {
		t.Fatal(err)
	}
	digest := strings.Repeat("a", 64)
	payload := []byte(`{"schemaVersion":"1","revision":"stack-1","targets":[{"agentId":"runtime-b","agentCardVersion":"1.0.0","releaseId":"release-b","cardDigest":"` + digest + `","canonicalEndpoint":"http://runtime-b:8092/","audience":"http://runtime-b:8092","instances":[{"instanceId":"runtime-b-1","ready":true,"serving":true,"terminating":false,"endpoints":[{"addressType":"DNS","address":"runtime-b-1","portName":"a2a","port":8092,"protocol":"TCP"}]}]}]}`)
	var value any
	if err := json.Unmarshal(payload, &value); err != nil {
		t.Fatal(err)
	}
	if err := schema.Validate(value); err != nil {
		t.Fatalf("schema rejected valid document: %v", err)
	}
	if _, err := DecodeRouterInstanceDirectoryV1(payload); err != nil {
		t.Fatalf("decoder rejected schema-valid document: %v", err)
	}
}

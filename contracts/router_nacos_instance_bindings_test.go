package contracts

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestDecodeRouterNacosInstanceBindingsV1(t *testing.T) {
	digest := strings.Repeat("a", 64)
	valid := []byte(`{"schemaVersion":"1","revision":"stack-1","targets":[{"agentId":"runtime-b","agentCardVersion":"1.0.0","releaseId":"release-b","cardDigest":"` + digest + `","canonicalEndpoint":"http://runtime-b:8092/","audience":"http://runtime-b:8092","serviceName":"runtime-b","groupName":"NEKIRO","clusterName":"DEFAULT"}]}`)
	document, err := DecodeRouterNacosInstanceBindingsV1(valid)
	if err != nil || len(document.Targets) != 1 {
		t.Fatalf("decode valid bindings: value=%#v error=%v", document, err)
	}
	for name, payload := range map[string][]byte{
		"malformed":       []byte(`{`),
		"unknown":         []byte(`{"schemaVersion":"1","revision":"stack-1","targets":[],"fallback":true}`),
		"duplicate":       []byte(`{"schemaVersion":"1","revision":"stack-1","revision":"stack-2","targets":[]}`),
		"trailing value":  []byte(`{"schemaVersion":"1","revision":"stack-1","targets":[]} {}`),
		"trailing token":  []byte(`{"schemaVersion":"1","revision":"stack-1","targets":[]} !`),
		"missing targets": []byte(`{"schemaVersion":"1","revision":"stack-1"}`),
		"bad version":     []byte(`{"schemaVersion":"2","revision":"stack-1","targets":[]}`),
		"bad service":     []byte(`{"schemaVersion":"1","revision":"stack-1","targets":[{"agentId":"runtime-b","agentCardVersion":"1.0.0","releaseId":"release-b","cardDigest":"` + digest + `","canonicalEndpoint":"http://runtime-b:8092/","audience":"http://runtime-b:8092","serviceName":"bad service","groupName":"NEKIRO","clusterName":"DEFAULT"}]}`),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeRouterNacosInstanceBindingsV1(payload); err == nil {
				t.Fatal("invalid bindings accepted")
			}
		})
	}
}

func TestRouterNacosInstanceBindingsV1SchemaCompilesAndMatchesDecoder(t *testing.T) {
	data, err := fs.ReadFile(ContractFiles(), "schemas/router-nacos-instance-bindings.v1.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	document, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	const schemaID = "https://schemas.nekiro.dev/router-nacos-instance-bindings/v1"
	if err := compiler.AddResource(schemaID, document); err != nil {
		t.Fatal(err)
	}
	schema, err := compiler.Compile(schemaID)
	if err != nil {
		t.Fatal(err)
	}
	digest := strings.Repeat("a", 64)
	payload := []byte(`{"schemaVersion":"1","revision":"stack-1","targets":[{"agentId":"runtime-b","agentCardVersion":"1.0.0","releaseId":"release-b","cardDigest":"` + digest + `","canonicalEndpoint":"http://runtime-b:8092/","audience":"http://runtime-b:8092","serviceName":"runtime-b","groupName":"NEKIRO","clusterName":"DEFAULT"}]}`)
	var value any
	if err := json.Unmarshal(payload, &value); err != nil {
		t.Fatal(err)
	}
	if err := schema.Validate(value); err != nil {
		t.Fatalf("schema rejected valid document: %v", err)
	}
	if _, err := DecodeRouterNacosInstanceBindingsV1(payload); err != nil {
		t.Fatalf("decoder rejected schema-valid document: %v", err)
	}
}

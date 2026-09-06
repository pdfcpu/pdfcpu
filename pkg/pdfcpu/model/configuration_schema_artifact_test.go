//go:build !js
// +build !js

/*
Copyright 2026 The pdfcpu Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"slices"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"go.yaml.in/yaml/v3"
)

const (
	configurationSchemaV1Dialect = "https://json-schema.org/draft/2020-12/schema"
	configurationSchemaV1ID      = "https://pdfcpu.io/schemas/config-v1.schema.json"
	configurationSchemaV1Path    = "resources/config-v1.schema.json"
)

func loadConfigurationSchemaV1(t *testing.T) (*jsonschema.Schema, *jsonschema.Resolved) {
	t.Helper()

	bb, err := os.ReadFile(configurationSchemaV1Path)
	if err != nil {
		t.Fatal(err)
	}
	var schema jsonschema.Schema
	if err := json.Unmarshal(bb, &schema); err != nil {
		t.Fatal(err)
	}
	if schema.Schema != configurationSchemaV1Dialect {
		t.Fatalf("schema dialect: got %q, want %q", schema.Schema, configurationSchemaV1Dialect)
	}
	if schema.ID != configurationSchemaV1ID {
		t.Fatalf("schema ID: got %q, want %q", schema.ID, configurationSchemaV1ID)
	}
	resolved, err := schema.Resolve(&jsonschema.ResolveOptions{ValidateDefaults: true})
	if err != nil {
		t.Fatalf("resolve schema offline: %v", err)
	}
	return &schema, resolved
}

func schemaValidationModeName(mode int) string {
	if mode == ValidationStrict {
		return "ValidationStrict"
	}
	return "ValidationRelaxed"
}

func schemaEOLName(eol string) string {
	switch eol {
	case "\r":
		return "EolCR"
	case "\r\n":
		return "EolCRLF"
	default:
		return "EolLF"
	}
}

func schemaUnitName(unit types.DisplayUnit) string {
	switch unit {
	case types.INCHES:
		return "inches"
	case types.CENTIMETRES:
		return "cm"
	case types.MILLIMETRES:
		return "mm"
	default:
		return "points"
	}
}

func schemaRevocationCheckerName(checker int) string {
	if checker == OCSP {
		return "ocsp"
	}
	return "crl"
}

func schema1DefaultValues(conf *Configuration) map[string]any {
	return map[string]any{
		"allowedRevocationHosts":          []string{},
		"checkFileNameExt":                conf.CheckFileNameExt,
		"createBookmarks":                 conf.CreateBookmarks,
		"dateFormat":                      conf.DateFormat,
		"decodeAllStreams":                conf.DecodeAllStreams,
		"encryptKeyLength":                conf.EncryptKeyLength,
		"encryptUsingAES":                 conf.EncryptUsingAES,
		"eol":                             schemaEOLName(conf.Eol),
		"formFieldListMaxColWidth":        conf.FormFieldListMaxColWidth,
		"maxDecodeBytes":                  conf.Limits.MaxDecodeBytes,
		"maxImageBytes":                   conf.Limits.MaxImageBytes,
		"maxImagePixels":                  conf.Limits.MaxImagePixels,
		"maxStreamBytes":                  conf.Limits.MaxStreamBytes,
		"needAppearances":                 conf.NeedAppearances,
		"offline":                         conf.Offline,
		"optimize":                        conf.Optimize,
		"optimizeBeforeWriting":           conf.OptimizeBeforeWriting,
		"optimizeDuplicateContentStreams": conf.OptimizeDuplicateContentStreams,
		"optimizeResourceDicts":           conf.OptimizeResourceDicts,
		"permissions":                     int(conf.Permissions),
		"postProcessValidate":             conf.PostProcessValidate,
		"preferredCertRevocationChecker":  schemaRevocationCheckerName(conf.PreferredCertRevocationChecker),
		"reader15":                        conf.Reader15,
		"schemaVersion":                   conf.SchemaVersion,
		"timeout":                         conf.Timeout,
		"timeoutCRL":                      conf.TimeoutCRL,
		"timeoutOCSP":                     conf.TimeoutOCSP,
		"timestampFormat":                 conf.TimestampFormat,
		"unit":                            schemaUnitName(conf.Unit),
		"validationMode":                  schemaValidationModeName(conf.ValidationMode),
		"writeObjectStream":               conf.WriteObjectStream,
		"writeXRefStream":                 conf.WriteXRefStream,
	}
}

func compactJSON(t *testing.T, bb []byte) string {
	t.Helper()
	var buf bytes.Buffer
	if err := json.Compact(&buf, bb); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func yamlScalarJSONValue(node *yaml.Node) (any, error) {
	switch node.Tag {
	case "!!bool":
		var value bool
		return value, node.Decode(&value)
	case "!!int":
		var value int64
		return value, node.Decode(&value)
	case "!!float":
		var value float64
		return value, node.Decode(&value)
	case "!!null":
		return nil, nil
	default:
		return node.Value, nil
	}
}

func yamlJSONValue(node *yaml.Node) (any, error) {
	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) == 0 {
			return nil, nil
		}
		return yamlJSONValue(node.Content[0])
	case yaml.MappingNode:
		value := make(map[string]any, len(node.Content)/2)
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i]
			if key.Kind != yaml.ScalarNode {
				return nil, fmt.Errorf("configuration key is not a scalar: %s", key.Value)
			}
			item, err := yamlJSONValue(node.Content[i+1])
			if err != nil {
				return nil, err
			}
			value[key.Value] = item
		}
		return value, nil
	case yaml.SequenceNode:
		value := make([]any, 0, len(node.Content))
		for _, item := range node.Content {
			converted, err := yamlJSONValue(item)
			if err != nil {
				return nil, err
			}
			value = append(value, converted)
		}
		return value, nil
	case yaml.AliasNode:
		return yamlJSONValue(node.Alias)
	case yaml.ScalarNode:
		return yamlScalarJSONValue(node)
	default:
		return nil, fmt.Errorf("unsupported YAML node kind: %d", node.Kind)
	}
}

func TestConfigurationSchemaV1ResolvesOffline(t *testing.T) {
	loadConfigurationSchemaV1(t)
}

func TestConfigurationSchemaV1KeysMatchLoader(t *testing.T) {
	schema, _ := loadConfigurationSchemaV1(t)
	got := slices.Sorted(maps.Keys(schema.Properties))
	want := slices.Sorted(maps.Keys(schema1ConfigurationKeys))
	if !slices.Equal(got, want) {
		t.Fatalf("schema keys:\n got: %v\nwant: %v", got, want)
	}
}

func TestShippedConfigurationSatisfiesSchemaV1(t *testing.T) {
	_, resolved := loadConfigurationSchemaV1(t)
	var document yaml.Node
	if err := yaml.Unmarshal(configFileBytes, &document); err != nil {
		t.Fatal(err)
	}
	instance, err := yamlJSONValue(&document)
	if err != nil {
		t.Fatal(err)
	}
	if err := resolved.Validate(instance); err != nil {
		t.Fatalf("validate shipped configuration: %v", err)
	}
}

func TestConfigurationSchemaV1DefaultsMatchGoDefaults(t *testing.T) {
	schema, _ := loadConfigurationSchemaV1(t)
	defaults := schema1DefaultValues(NewStatelessConfiguration())
	if got, want := len(defaults), len(schema.Properties)-1; got != want {
		t.Fatalf("documented defaults: got %d, want %d", got, want)
	}
	for _, key := range slices.Sorted(maps.Keys(defaults)) {
		property := schema.Properties[key]
		if len(property.Default) == 0 {
			t.Fatalf("property %q has no documented default", key)
		}
		want, err := json.Marshal(defaults[key])
		if err != nil {
			t.Fatal(err)
		}
		if got, want := compactJSON(t, property.Default), compactJSON(t, want); got != want {
			t.Fatalf("property %q default: got %s, want %s", key, got, want)
		}
	}
	if len(schema.Properties["created"].Default) != 0 {
		t.Fatal("informational property created must not have a static default")
	}
}

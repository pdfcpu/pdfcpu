//go:build !js
// +build !js

/*
Copyright 2020 The pdfcpu Authors.

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
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"go.yaml.in/yaml/v3"
)

type configuration struct {
	CreationDate                    string           `yaml:"created"`
	Version                         string           `yaml:"version"`
	SchemaVersion                   int              `yaml:"-"`
	CheckFileNameExt                bool             `yaml:"checkFileNameExt"`
	Reader15                        bool             `yaml:"reader15"`
	DecodeAllStreams                bool             `yaml:"decodeAllStreams"`
	ValidationMode                  string           `yaml:"validationMode"`
	PostProcessValidate             bool             `yaml:"postProcessValidate"`
	Eol                             string           `yaml:"eol"`
	WriteObjectStream               bool             `yaml:"writeObjectStream"`
	WriteXRefStream                 bool             `yaml:"writeXRefStream"`
	EncryptUsingAES                 bool             `yaml:"encryptUsingAES"`
	EncryptKeyLength                int              `yaml:"encryptKeyLength"`
	Permissions                     int              `yaml:"permissions"`
	Unit                            string           `yaml:"unit"`
	TimestampFormat                 string           `yaml:"timestampFormat"`
	DateFormat                      string           `yaml:"dateFormat"`
	Optimize                        bool             `yaml:"optimize"`
	OptimizeBeforeWriting           bool             `yaml:"optimizeBeforeWriting"`
	OptimizeResourceDicts           bool             `yaml:"optimizeResourceDicts"`
	OptimizeDuplicateContentStreams bool             `yaml:"optimizeDuplicateContentStreams"`
	CreateBookmarks                 bool             `yaml:"createBookmarks"`
	NeedAppearances                 bool             `yaml:"needAppearances"`
	Offline                         bool             `yaml:"offline"`
	Timeout                         int              `yaml:"timeout"`
	TimeoutCRL                      int              `yaml:"timeoutCRL"`
	TimeoutOCSP                     int              `yaml:"timeoutOCSP"`
	AllowedRevocationHosts          []string         `yaml:"allowedRevocationHosts"`
	PreferredCertRevocationChecker  string           `yaml:"preferredCertRevocationChecker"`
	FormFieldListMaxColWidth        int              `yaml:"formFieldListMaxColWidth"`
	MaxInputBytes                   *inputLimitValue `yaml:"maxInputBytes"`
	MaxObjectBytes                  *int64Value      `yaml:"maxObjectBytes"`
	MaxStreamBytes                  *int64Value      `yaml:"maxStreamBytes"`
	MaxDecodeBytes                  *int64Value      `yaml:"maxDecodeBytes"`
	MaxImagePixels                  *int64Value      `yaml:"maxImagePixels"`
	MaxImageBytes                   *int64Value      `yaml:"maxImageBytes"`
}

func schemaVersionValueFromYAML(document *yaml.Node) (int, error) {
	if document.Kind != yaml.DocumentNode || len(document.Content) == 0 {
		return ConfigurationSchemaVersionLegacy, nil
	}
	mapping := document.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return ConfigurationSchemaVersionLegacy, nil
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value != "schemaVersion" {
			continue
		}
		value := mapping.Content[i+1]
		if value.Kind != yaml.ScalarNode || value.Tag != "!!int" {
			return 0, fmt.Errorf(
				"%w: schemaVersion must be a positive integer, got %q",
				ErrInvalidConfigurationSchema,
				value.Value,
			)
		}
		var version int
		if err := value.Decode(&version); err != nil {
			return 0, fmt.Errorf("%w: decode schemaVersion: %v", ErrInvalidConfigurationSchema, err)
		}
		if err := validateConfigurationSchemaVersionValue(version); err != nil {
			return 0, err
		}
		return version, nil
	}
	return ConfigurationSchemaVersionLegacy, nil
}

func schemaVersionFromYAML(document *yaml.Node) (int, error) {
	version, err := schemaVersionValueFromYAML(document)
	if err != nil || version == ConfigurationSchemaVersionLegacy {
		return version, err
	}
	if err := validateConfigurationSchemaVersion(version); err != nil {
		return 0, err
	}
	return version, nil
}

func readConfigurationSchemaVersion(r io.Reader) (int, error) {
	var document yaml.Node
	if err := yaml.NewDecoder(r).Decode(&document); err != nil {
		return 0, err
	}
	return schemaVersionValueFromYAML(&document)
}

func configurationKeyNamesFromYAML(document *yaml.Node) []string {
	if document.Kind != yaml.DocumentNode || len(document.Content) == 0 {
		return nil
	}
	mapping := document.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return nil
	}
	keys := make([]string, 0, len(mapping.Content)/2)
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		keys = append(keys, mapping.Content[i].Value)
	}
	return keys
}

type inputLimitValue int64

// UnmarshalYAML unmarshals an input byte limit, including zero for unlimited input.
func (i *inputLimitValue) UnmarshalYAML(value *yaml.Node) error {
	if (value.Tag == "!!int" || value.Tag == "!!str") && value.Value == "0" {
		*i = 0
		return nil
	}
	var n int64Value
	if err := n.UnmarshalYAML(value); err != nil {
		return err
	}
	*i = inputLimitValue(n)
	return nil
}

type int64Value int64

// UnmarshalYAML unmarshals a positive int64 value.
func (i *int64Value) UnmarshalYAML(value *yaml.Node) error {
	var n int64
	if err := value.Decode(&n); err == nil {
		if n <= 0 {
			return fmt.Errorf("numeric value must be > 0: %d", n)
		}
		*i = int64Value(n)
		return nil
	}

	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}

	n, err := parseReadableInt64(s)
	if err != nil {
		return err
	}
	*i = int64Value(n)
	return nil
}

func parseReadableInt64(s string) (int64, error) {
	ss := strings.Fields(strings.ToUpper(strings.TrimSpace(s)))
	if len(ss) == 0 || len(ss) > 2 {
		return 0, fmt.Errorf("invalid numeric value: %s", s)
	}

	n, err := strconv.ParseInt(ss[0], 10, 64)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("numeric value must be > 0: %s", s)
	}
	if len(ss) == 1 {
		return n, nil
	}

	m := int64(1)
	switch ss[1] {
	case "B", "BYTE", "BYTES":
	case "KB", "KIB":
		m = 1 << 10
	case "MB", "MIB":
		m = 1 << 20
	case "GB", "GIB":
		m = 1 << 30
	case "MP", "MPIXELS":
		m = 1000 * 1000
	default:
		return 0, fmt.Errorf("unsupported numeric unit: %s", ss[1])
	}

	if n > (1<<63-1)/m {
		return 0, fmt.Errorf("numeric value overflows int64: %s", s)
	}
	return n * m, nil
}

func loadValidationMode(c configuration, conf *Configuration) {
	switch c.ValidationMode {
	case "ValidationStrict":
		conf.ValidationMode = ValidationStrict
	case "ValidationRelaxed":
		conf.ValidationMode = ValidationRelaxed
	}
}

func loadedConfig(c configuration, configPath string) *Configuration {
	conf := Configuration{UnsupportedResourcePolicy: UnsupportedResourceSkip}
	conf.Path = configPath

	conf.CreationDate = c.CreationDate
	conf.Version = c.Version
	conf.SchemaVersion = c.SchemaVersion
	conf.CheckFileNameExt = c.CheckFileNameExt
	conf.Reader15 = c.Reader15
	conf.DecodeAllStreams = c.DecodeAllStreams
	conf.WriteObjectStream = c.WriteObjectStream
	conf.WriteXRefStream = c.WriteXRefStream
	conf.EncryptUsingAES = c.EncryptUsingAES
	conf.EncryptKeyLength = c.EncryptKeyLength
	conf.Permissions = PermissionFlags(c.Permissions)

	loadValidationMode(c, &conf)

	conf.PostProcessValidate = c.PostProcessValidate

	switch c.Eol {
	case "EolLF":
		conf.Eol = types.EolLF
	case "EolCR":
		conf.Eol = types.EolCR
	case "EolCRLF":
		conf.Eol = types.EolCRLF
	}

	switch c.Unit {
	case "points":
		conf.Unit = types.POINTS
	case "inches":
		conf.Unit = types.INCHES
	case "cm":
		conf.Unit = types.CENTIMETRES
	case "mm":
		conf.Unit = types.MILLIMETRES
	}

	conf.TimestampFormat = c.TimestampFormat
	conf.DateFormat = c.DateFormat
	conf.Optimize = c.Optimize
	conf.OptimizeBeforeWriting = true
	conf.OptimizeResourceDicts = c.OptimizeResourceDicts
	conf.OptimizeDuplicateContentStreams = c.OptimizeDuplicateContentStreams
	conf.CreateBookmarks = c.CreateBookmarks
	conf.MergeBookmarkMode = MergeBookmarkModeWrap
	conf.NeedAppearances = c.NeedAppearances
	conf.Offline = c.Offline
	conf.Timeout = c.Timeout
	conf.TimeoutCRL = c.TimeoutCRL
	conf.TimeoutOCSP = c.TimeoutOCSP
	conf.AllowedRevocationHosts = append([]string(nil), c.AllowedRevocationHosts...)
	conf.FormFieldListMaxColWidth = c.FormFieldListMaxColWidth
	conf.Limits = DefaultResourceLimits()

	applyConfigurationLimits(c, &conf)

	switch strings.ToLower(c.PreferredCertRevocationChecker) {
	case "crl":
		conf.PreferredCertRevocationChecker = CRL
	case "ocsp":
		conf.PreferredCertRevocationChecker = OCSP
	}

	return &conf
}

func overlaySchema1Metadata(c configuration, keys map[string]bool, conf *Configuration) {
	if keys["created"] {
		conf.CreationDate = c.CreationDate
	}
	if keys["checkFileNameExt"] {
		conf.CheckFileNameExt = c.CheckFileNameExt
	}
	if keys["reader15"] {
		conf.Reader15 = c.Reader15
	}
	if keys["decodeAllStreams"] {
		conf.DecodeAllStreams = c.DecodeAllStreams
	}
	if keys["validationMode"] {
		loadValidationMode(c, conf)
	}
	if keys["postProcessValidate"] {
		conf.PostProcessValidate = c.PostProcessValidate
	}
}

func overlaySchema1Output(c configuration, keys map[string]bool, conf *Configuration) {
	if keys["eol"] {
		switch c.Eol {
		case "EolLF":
			conf.Eol = types.EolLF
		case "EolCR":
			conf.Eol = types.EolCR
		case "EolCRLF":
			conf.Eol = types.EolCRLF
		}
	}
	if keys["writeObjectStream"] {
		conf.WriteObjectStream = c.WriteObjectStream
	}
	if keys["writeXRefStream"] {
		conf.WriteXRefStream = c.WriteXRefStream
	}
	if keys["encryptUsingAES"] {
		conf.EncryptUsingAES = c.EncryptUsingAES
	}
	if keys["encryptKeyLength"] {
		conf.EncryptKeyLength = c.EncryptKeyLength
	}
	if keys["permissions"] {
		conf.Permissions = PermissionFlags(c.Permissions)
	}
}

func overlaySchema1Formatting(c configuration, keys map[string]bool, conf *Configuration) {
	if keys["unit"] {
		switch c.Unit {
		case "points":
			conf.Unit = types.POINTS
		case "inches":
			conf.Unit = types.INCHES
		case "cm":
			conf.Unit = types.CENTIMETRES
		case "mm":
			conf.Unit = types.MILLIMETRES
		}
	}
	if keys["timestampFormat"] {
		conf.TimestampFormat = c.TimestampFormat
	}
	if keys["dateFormat"] {
		conf.DateFormat = c.DateFormat
	}
}

func overlaySchema1Optimization(c configuration, keys map[string]bool, conf *Configuration) {
	if keys["optimize"] {
		conf.Optimize = c.Optimize
	}
	if keys["optimizeBeforeWriting"] {
		conf.OptimizeBeforeWriting = c.OptimizeBeforeWriting
	}
	if keys["optimizeResourceDicts"] {
		conf.OptimizeResourceDicts = c.OptimizeResourceDicts
	}
	if keys["optimizeDuplicateContentStreams"] {
		conf.OptimizeDuplicateContentStreams = c.OptimizeDuplicateContentStreams
	}
	if keys["createBookmarks"] {
		conf.CreateBookmarks = c.CreateBookmarks
	}
	if keys["needAppearances"] {
		conf.NeedAppearances = c.NeedAppearances
	}
}

func overlaySchema1Network(c configuration, keys map[string]bool, conf *Configuration) {
	if keys["offline"] {
		conf.Offline = c.Offline
	}
	if keys["timeout"] {
		conf.Timeout = c.Timeout
	}
	if keys["timeoutCRL"] {
		conf.TimeoutCRL = c.TimeoutCRL
	}
	if keys["timeoutOCSP"] {
		conf.TimeoutOCSP = c.TimeoutOCSP
	}
	if keys["allowedRevocationHosts"] {
		conf.AllowedRevocationHosts = append([]string(nil), c.AllowedRevocationHosts...)
	}
	if keys["preferredCertRevocationChecker"] {
		switch strings.ToLower(c.PreferredCertRevocationChecker) {
		case "crl":
			conf.PreferredCertRevocationChecker = CRL
		case "ocsp":
			conf.PreferredCertRevocationChecker = OCSP
		}
	}
	if keys["formFieldListMaxColWidth"] {
		conf.FormFieldListMaxColWidth = c.FormFieldListMaxColWidth
	}
}

func applyConfigurationLimits(c configuration, conf *Configuration) {
	if c.MaxInputBytes != nil {
		conf.Limits.MaxInputBytes = int64(*c.MaxInputBytes)
	}
	if c.MaxObjectBytes != nil {
		conf.Limits.MaxObjectBytes = int64(*c.MaxObjectBytes)
	}
	if c.MaxStreamBytes != nil {
		conf.Limits.MaxStreamBytes = int64(*c.MaxStreamBytes)
	}
	if c.MaxDecodeBytes != nil {
		conf.Limits.MaxDecodeBytes = int64(*c.MaxDecodeBytes)
	}
	if c.MaxImagePixels != nil {
		conf.Limits.MaxImagePixels = int64(*c.MaxImagePixels)
	}
	if c.MaxImageBytes != nil {
		conf.Limits.MaxImageBytes = int64(*c.MaxImageBytes)
	}
}

func loadedSchema1Config(c configuration, keys map[string]bool, configPath string) *Configuration {
	conf := NewStatelessConfiguration()
	conf.resources = configurationResources{}
	conf.Path = configPath

	overlaySchema1Metadata(c, keys, conf)
	overlaySchema1Output(c, keys, conf)
	overlaySchema1Formatting(c, keys, conf)
	overlaySchema1Optimization(c, keys, conf)
	overlaySchema1Network(c, keys, conf)
	applyConfigurationLimits(c, conf)

	return conf
}

func validateLegacyConfiguration(c *configuration) error {
	if !types.MemberOf(c.ValidationMode, []string{"ValidationStrict", "ValidationRelaxed"}) {
		return fmt.Errorf("invalid validationMode: %s", c.ValidationMode)
	}
	if !types.MemberOf(c.Eol, []string{"EolLF", "EolCR", "EolCRLF"}) {
		return fmt.Errorf("invalid eol: %s", c.Eol)
	}
	if !types.MemberOf(c.Unit, []string{"points", "inches", "cm", "mm"}) {
		return fmt.Errorf("invalid unit: %s", c.Unit)
	}
	if !types.IntMemberOf(c.EncryptKeyLength, []int{40, 128, 256}) {
		return fmt.Errorf("encryptKeyLength possible values: 40, 128, 256, got: %s", c.Unit)
	}
	if !types.MemberOf(c.PreferredCertRevocationChecker, []string{"crl", "ocsp"}) {
		if c.PreferredCertRevocationChecker != "" {
			return fmt.Errorf("invalid preferred certificate revocation checker: %s", c.PreferredCertRevocationChecker)
		}
		c.PreferredCertRevocationChecker = "crl"
	}
	if c.FormFieldListMaxColWidth < 0 {
		return fmt.Errorf("formFieldListMaxColWidth must be >= 0: %d", c.FormFieldListMaxColWidth)
	}
	return nil
}

func validateSchema1Configuration(c configuration, keys map[string]bool) error {
	if keys["validationMode"] && !types.MemberOf(c.ValidationMode, []string{"ValidationStrict", "ValidationRelaxed"}) {
		return fmt.Errorf("invalid validationMode: %s", c.ValidationMode)
	}
	if keys["eol"] && !types.MemberOf(c.Eol, []string{"EolLF", "EolCR", "EolCRLF"}) {
		return fmt.Errorf("invalid eol: %s", c.Eol)
	}
	if keys["unit"] && !types.MemberOf(c.Unit, []string{"points", "inches", "cm", "mm"}) {
		return fmt.Errorf("invalid unit: %s", c.Unit)
	}
	if keys["encryptKeyLength"] && !types.IntMemberOf(c.EncryptKeyLength, []int{40, 128, 256}) {
		return fmt.Errorf("encryptKeyLength possible values: 40, 128, 256, got: %d", c.EncryptKeyLength)
	}
	if keys["preferredCertRevocationChecker"] &&
		!types.MemberOf(c.PreferredCertRevocationChecker, []string{"crl", "ocsp", ""}) {
		return fmt.Errorf("invalid preferred certificate revocation checker: %s", c.PreferredCertRevocationChecker)
	}
	if keys["formFieldListMaxColWidth"] && c.FormFieldListMaxColWidth < 0 {
		return fmt.Errorf("formFieldListMaxColWidth must be >= 0: %d", c.FormFieldListMaxColWidth)
	}
	return nil
}

func readConfiguration(r io.Reader, configPath string) (*Configuration, error) {
	var c configuration

	// Enforce default for old config files.
	c.CheckFileNameExt = true

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		return nil, err
	}

	var document yaml.Node
	if err := yaml.Unmarshal(buf.Bytes(), &document); err != nil {
		return nil, err
	}
	schemaVersion, err := schemaVersionFromYAML(&document)
	if err != nil {
		return nil, err
	}
	var keys map[string]bool
	if schemaVersion == ConfigurationSchemaVersionCurrent {
		keys, err = schema1ConfigurationKeySet(configurationKeyNamesFromYAML(&document))
		if err != nil {
			return nil, err
		}
	}
	if err := document.Decode(&c); err != nil {
		return nil, err
	}
	c.SchemaVersion = schemaVersion
	if schemaVersion == ConfigurationSchemaVersionLegacy {
		if err := validateLegacyConfiguration(&c); err != nil {
			return nil, err
		}
		return loadedConfig(c, configPath), nil
	}
	if err := validateSchema1Configuration(c, keys); err != nil {
		return nil, err
	}
	return loadedSchema1Config(c, keys, configPath), nil
}

func parseConfigFile(r io.Reader, configPath string) error {
	conf, err := readConfiguration(r, configPath)
	if err != nil {
		return err
	}
	defaultConfigMu.Lock()
	defer defaultConfigMu.Unlock()
	loadedDefaultConfig = conf
	return nil
}

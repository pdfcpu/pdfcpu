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
// +build !js

package pdfcpu

import (
	"io"
	"io/ioutil"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type configuration struct {
	Reader15          bool   `yaml:"reader15"`
	DecodeAllStreams  bool   `yaml:"decodeAllStreams"`
	ValidationMode    string `yaml:"validationMode"`
	Eol               string `yaml:"eol"`
	WriteObjectStream bool   `yaml:"writeObjectStream"`
	WriteXRefStream   bool   `yaml:"writeXRefStream"`
	EncryptUsingAES   bool   `yaml:"encryptUsingAES"`
	EncryptKeyLength  int    `yaml:"encryptKeyLength"`
	Permissions       int    `yaml:"permissions"`
	Units             string `yaml:"units"`
}

func loadedConfig(c configuration, configPath string) *Configuration {
	var conf Configuration
	conf.Path = configPath

	conf.Reader15 = c.Reader15
	conf.DecodeAllStreams = c.DecodeAllStreams
	conf.WriteObjectStream = c.WriteObjectStream
	conf.WriteXRefStream = c.WriteXRefStream
	conf.EncryptUsingAES = c.EncryptUsingAES
	conf.EncryptKeyLength = c.EncryptKeyLength
	conf.Permissions = int16(c.Permissions)

	switch c.ValidationMode {
	case "ValidationStrict":
		conf.ValidationMode = ValidationStrict
	case "ValidationRelaxed":
		conf.ValidationMode = ValidationRelaxed
	case "ValidationNone":
		conf.ValidationMode = ValidationNone
	}

	switch c.Eol {
	case "EolLF":
		conf.Eol = EolLF
	case "EolCR":
		conf.Eol = EolCR
	case "EolCRLF":
		conf.Eol = EolCRLF
	}

	switch c.Units {
	case "points":
		conf.Units = POINTS
	case "inches":
		conf.Units = INCHES
	case "cm":
		conf.Units = CENTIMETRES
	case "mm":
		conf.Units = MILLIMETRES
	}

	return &conf
}

func parseConfigFile(r io.Reader, configPath string) error {
	var c configuration
	bb, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(bb, &c); err != nil {
		return err
	}
	if !MemberOf(c.ValidationMode, []string{"ValidationStrict", "ValidationRelaxed", "ValidationNone"}) {
		return errors.Errorf("invalid validationMode: %s", c.ValidationMode)
	}
	if !MemberOf(c.Eol, []string{"EolLF", "EolCR", "EolCRLF"}) {
		return errors.Errorf("invalid eol: %s", c.Eol)
	}
	if !MemberOf(c.Units, []string{"points", "inches", "cm", "mm"}) {
		errors.Errorf("invalid units: %s", c.Units)
	}
	if !IntMemberOf(c.EncryptKeyLength, []int{40, 128, 256}) {
		return errors.Errorf("encryptKeyLength possible values: 40, 128, 256, got: %s", c.Units)
	}
	loadedDefaultConfig = loadedConfig(c, configPath)
	return nil
}

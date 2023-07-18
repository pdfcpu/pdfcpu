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
	"bufio"
	"io"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

// This gets rid of the gopkg.in/yaml.v2 dependency for wasm builds.

func handleCheckFileNameExt(k, v string, c *Configuration) error {
	v = strings.ToLower(v)
	if v != "true" && v != "false" {
		return errors.Errorf("config key %s is boolean", k)
	}
	c.CheckFileNameExt = v == "true"
	return nil
}

func handleConfReader15(k, v string, c *Configuration) error {
	v = strings.ToLower(v)
	if v != "true" && v != "false" {
		return errors.Errorf("config key %s is boolean", k)
	}
	c.Reader15 = v == "true"
	return nil
}

func handleConfDecodeAllStreams(k, v string, c *Configuration) error {
	v = strings.ToLower(v)
	if v != "true" && v != "false" {
		return errors.Errorf("config key %s is boolean", k)
	}
	c.DecodeAllStreams = v == "true"
	return nil
}

func handleConfValidationMode(v string, c *Configuration) error {
	v1 := strings.ToLower(v)
	switch v1 {
	case "validationstrict":
		c.ValidationMode = ValidationStrict
	case "validationrelaxed":
		c.ValidationMode = ValidationRelaxed
	case "validationone":
		c.ValidationMode = ValidationNone
	default:
		return errors.Errorf("invalid validationMode: %s", v)
	}
	return nil
}

func handleConfEol(v string, c *Configuration) error {
	v1 := strings.ToLower(v)
	switch v1 {
	case "eollf":
		c.Eol = types.EolLF
	case "eolcr":
		c.Eol = types.EolCR
	case "eolcrlf":
		c.Eol = types.EolCRLF
	default:
		return errors.Errorf("invalid eol: %s", v)
	}
	return nil
}

func handleConfWriteObjectStream(k, v string, c *Configuration) error {
	v = strings.ToLower(v)
	if v != "true" && v != "false" {
		return errors.Errorf("config key %s is boolean", k)
	}
	c.WriteObjectStream = v == "true"
	return nil
}

func handleConfWriteXRefStream(k, v string, c *Configuration) error {
	v = strings.ToLower(v)
	if v != "true" && v != "false" {
		return errors.Errorf("config key %s is boolean", k)
	}
	c.WriteXRefStream = v == "true"
	return nil
}

func handleConfEncryptUsingAES(k, v string, c *Configuration) error {
	v = strings.ToLower(v)
	if v != "true" && v != "false" {
		return errors.Errorf("config key %s is boolean", k)
	}
	c.EncryptUsingAES = v == "true"
	return nil
}

func handleConfEncryptKeyLength(v string, c *Configuration) error {
	i, err := strconv.Atoi(v)
	if err != nil {
		return errors.Errorf("encryptKeyLength is numeric, got: %s", v)
	}
	if !types.IntMemberOf(i, []int{40, 128, 256}) {
		return errors.Errorf("encryptKeyLength possible values: 40, 128, 256, got: %s", v)
	}
	c.EncryptKeyLength = i
	return nil
}

func handleConfPermissions(v string, c *Configuration) error {
	i, err := strconv.Atoi(v)
	if err != nil {
		return errors.Errorf("permissions is numeric, got: %s", v)
	}
	c.Permissions = int16(i)
	return nil
}

func handleConfUnit(v string, c *Configuration) error {
	v1 := v
	switch v1 {
	case "points":
		c.Unit = types.POINTS
	case "inches":
		c.Unit = types.INCHES
	case "cm":
		c.Unit = types.CENTIMETRES
	case "mm":
		c.Unit = types.MILLIMETRES
	default:
		return errors.Errorf("invalid unit: %s", v)
	}
	return nil
}

func handleTimestampFormat(v string, c *Configuration) error {
	c.TimestampFormat = v
	return nil
}

func handleDateFormat(v string, c *Configuration) error {
	c.DateFormat = v
	return nil
}

func handleHeaderBufSize(k, v string, c *Configuration) error {
	i, err := strconv.Atoi(v)
	if err != nil {
		return errors.Errorf("%s is numeric, got: %s", k, v)
	}
	if i < 100 {
		return errors.Errorf("%s must be >= 100, got: %d", k, i)
	}
	c.HeaderBufSize = i
	return nil
}

func handleOptimizeDuplicateContentStreams(k, v string, c *Configuration) error {
	v = strings.ToLower(v)
	if v != "true" && v != "false" {
		return errors.Errorf("config key %s is boolean", k)
	}
	c.OptimizeDuplicateContentStreams = v == "true"
	return nil
}

func handleCreateBookmarks(k, v string, c *Configuration) error {
	v = strings.ToLower(v)
	if v != "true" && v != "false" {
		return errors.Errorf("config key %s is boolean", k)
	}
	c.CreateBookmarks = v == "true"
	return nil
}

func parseKeysPart1(k, v string, c *Configuration) (bool, error) {
	switch k {

	case "checkFileNameExt":
		return true, handleCheckFileNameExt(k, v, c)

	case "reader15":
		return true, handleConfReader15(k, v, c)

	case "decodeAllStreams":
		return true, handleConfDecodeAllStreams(k, v, c)

	case "validationMode":
		return true, handleConfValidationMode(v, c)

	case "eol":
		return true, handleConfEol(v, c)

	case "writeObjectStream":
		return true, handleConfWriteObjectStream(k, v, c)

	case "writeXRefStream":
		return true, handleConfWriteXRefStream(k, v, c)

	}

	return false, nil
}

func parseKeysPart2(k, v string, c *Configuration) error {
	switch k {

	case "encryptUsingAES":
		return handleConfEncryptUsingAES(k, v, c)

	case "encryptKeyLength":
		return handleConfEncryptKeyLength(v, c)

	case "permissions":
		return handleConfPermissions(v, c)

	case "unit", "units":
		return handleConfUnit(v, c)

	case "timestampFormat":
		return handleTimestampFormat(v, c)

	case "dateFormat":
		return handleDateFormat(v, c)

	case "headerBufSize":
		return handleHeaderBufSize(k, v, c)

	case "optimizeDuplicateContentStreams":
		return handleOptimizeDuplicateContentStreams(k, v, c)

	case "createBookmarks":
		return handleCreateBookmarks(k, v, c)
	}

	return nil
}

func parseKeyValue(k, v string, c *Configuration) error {
	ok, err := parseKeysPart1(k, v, c)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	return parseKeysPart2(k, v, c)
}

func parseConfigFile(r io.Reader, configPath string) error {
	//fmt.Println("parseConfigFile For JS")
	var conf Configuration
	conf.Path = configPath

	s := bufio.NewScanner(r)
	for s.Scan() {
		t := s.Text()
		if len(t) == 0 || t[0] == '#' {
			continue
		}
		ss := strings.Split(t, ": ")
		if len(ss) != 2 {
			return errors.Errorf("invalid entry: <%s>", t)
		}
		k := strings.TrimSpace(ss[0])
		v := strings.TrimSpace(ss[1])
		if len(k) == 0 || len(v) == 0 {
			return errors.Errorf("invalid entry: <%s>", t)
		}
		if err := parseKeyValue(k, v, &conf); err != nil {
			return err
		}
	}
	if err := s.Err(); err != nil {
		return err
	}

	loadedDefaultConfig = &conf
	return nil
}

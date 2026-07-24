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
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// This gets rid of the YAML dependency for wasm builds.

func handleCreationDate(v string, c *Configuration) error {
	c.CreationDate = v
	return nil
}

func handleVersion(v string, c *Configuration) error {
	c.Version = v
	return nil
}

func handleCheckFileNameExt(k, v string, c *Configuration) error {
	v = strings.ToLower(v)
	if v != "true" && v != "false" {
		return fmt.Errorf("config key %s is boolean", k)
	}
	c.CheckFileNameExt = v == "true"
	return nil
}

func handleConfReader15(k, v string, c *Configuration) error {
	v = strings.ToLower(v)
	if v != "true" && v != "false" {
		return fmt.Errorf("config key %s is boolean", k)
	}
	c.Reader15 = v == "true"
	return nil
}

func handleConfDecodeAllStreams(k, v string, c *Configuration) error {
	v = strings.ToLower(v)
	if v != "true" && v != "false" {
		return fmt.Errorf("config key %s is boolean", k)
	}
	c.DecodeAllStreams = v == "true"
	return nil
}

func handleConfPostProcessValidate(k, v string, c *Configuration) error {
	v = strings.ToLower(v)
	if v != "true" && v != "false" {
		return fmt.Errorf("config key %s is boolean", k)
	}
	c.PostProcessValidate = v == "true"
	return nil
}

func handleConfValidationMode(v string, c *Configuration) error {
	v1 := strings.ToLower(v)
	switch v1 {
	case "validationstrict":
		c.ValidationMode = ValidationStrict
	case "validationrelaxed":
		c.ValidationMode = ValidationRelaxed
	default:
		return fmt.Errorf("invalid validationMode: %s", v)
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
		return fmt.Errorf("invalid eol: %s", v)
	}
	return nil
}

func handleConfWriteObjectStream(k, v string, c *Configuration) error {
	v = strings.ToLower(v)
	if v != "true" && v != "false" {
		return fmt.Errorf("config key %s is boolean", k)
	}
	c.WriteObjectStream = v == "true"
	return nil
}

func handleConfWriteXRefStream(k, v string, c *Configuration) error {
	v = strings.ToLower(v)
	if v != "true" && v != "false" {
		return fmt.Errorf("config key %s is boolean", k)
	}
	c.WriteXRefStream = v == "true"
	return nil
}

func handleConfEncryptUsingAES(k, v string, c *Configuration) error {
	v = strings.ToLower(v)
	if v != "true" && v != "false" {
		return fmt.Errorf("config key %s is boolean", k)
	}
	c.EncryptUsingAES = v == "true"
	return nil
}

func handleConfEncryptKeyLength(v string, c *Configuration) error {
	i, err := strconv.Atoi(v)
	if err != nil {
		return fmt.Errorf("encryptKeyLength is numeric, got: %s", v)
	}
	if !types.IntMemberOf(i, []int{40, 128, 256}) {
		return fmt.Errorf("encryptKeyLength possible values: 40, 128, 256, got: %s", v)
	}
	c.EncryptKeyLength = i
	return nil
}

func handleFormFieldListMaxColWidth(v string, c *Configuration) error {
	i, err := strconv.Atoi(v)
	if err != nil || i < 0 {
		return fmt.Errorf("formFieldListMaxColWidth is numeric >= 0, got: %s", v)
	}
	c.FormFieldListMaxColWidth = i
	return nil
}

func handleTimeout(v string, c *Configuration) error {
	i, err := strconv.Atoi(v)
	if err != nil || i <= 0 {
		return fmt.Errorf("timeout is numeric > 0, got: %s", v)
	}
	c.Timeout = i
	return nil
}

func handleTimeoutCRL(v string, c *Configuration) error {
	i, err := strconv.Atoi(v)
	if err != nil || i <= 0 {
		return fmt.Errorf("timeoutCRL is numeric > 0, got: %s", v)
	}
	c.TimeoutCRL = i
	return nil
}

func handleTimeoutOCSP(v string, c *Configuration) error {
	i, err := strconv.Atoi(v)
	if err != nil || i <= 0 {
		return fmt.Errorf("timeoutOCSP is numeric > 0, got: %s", v)
	}
	c.TimeoutOCSP = i
	return nil
}

func handleAllowedRevocationHosts(v string, c *Configuration) error {
	v = strings.TrimSpace(v)
	if len(v) < 2 || v[0] != '[' || v[len(v)-1] != ']' {
		return fmt.Errorf("allowedRevocationHosts must be an inline list, got: %s", v)
	}
	v = strings.TrimSpace(v[1 : len(v)-1])
	if v == "" {
		c.AllowedRevocationHosts = nil
		return nil
	}

	hosts := strings.Split(v, ",")
	c.AllowedRevocationHosts = make([]string, 0, len(hosts))
	for _, host := range hosts {
		host = strings.Trim(strings.TrimSpace(host), `"'`)
		if host == "" {
			return fmt.Errorf("allowedRevocationHosts contains an empty host")
		}
		c.AllowedRevocationHosts = append(c.AllowedRevocationHosts, host)
	}
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

func handleLimitInt64(k, v string, dst *int64) error {
	i, err := parseReadableInt64(v)
	if err != nil {
		return fmt.Errorf("%s: %w", k, err)
	}
	*dst = i
	return nil
}

func handleLimitInt(k, v string, dst *int) error {
	i, err := strconv.Atoi(v)
	if err != nil || i <= 0 {
		return fmt.Errorf("%s is numeric > 0, got: %s", k, v)
	}
	*dst = i
	return nil
}

func handleConfPermissions(v string, c *Configuration) error {
	i, err := strconv.Atoi(v)
	if err != nil {
		return fmt.Errorf("permissions is numeric, got: %s", v)
	}
	c.Permissions = PermissionFlags(i)
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
		return fmt.Errorf("invalid unit: %s", v)
	}
	return nil
}

func handlePreferredCertRevocationChecker(v string, c *Configuration) error {
	v1 := strings.ToLower(v)
	switch v1 {
	case "crl":
		c.PreferredCertRevocationChecker = CRL
	case "ocsp":
		c.PreferredCertRevocationChecker = OCSP
	case "":
		c.PreferredCertRevocationChecker = CRL
	default:
		return fmt.Errorf("invalid preferredCertRevocationChecker: %s", v)
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

func boolean(k, v string) (bool, error) {
	v = strings.ToLower(v)
	if v != "true" && v != "false" {
		return false, fmt.Errorf("config key %s is boolean", k)
	}
	return v == "true", nil
}

func parseKeysPart1(k, v string, c *Configuration) (bool, error) {
	switch k {

	case "created":
		return true, handleCreationDate(v, c)

	case "version":
		return true, handleVersion(v, c)

	case "checkFileNameExt":
		return true, handleCheckFileNameExt(k, v, c)

	case "reader15":
		return true, handleConfReader15(k, v, c)

	case "decodeAllStreams":
		return true, handleConfDecodeAllStreams(k, v, c)

	case "validationMode":
		return true, handleConfValidationMode(v, c)

	case "postProcessValidate":
		return true, handleConfPostProcessValidate(k, v, c)

	case "eol":
		return true, handleConfEol(v, c)

	case "writeObjectStream":
		return true, handleConfWriteObjectStream(k, v, c)

	case "writeXRefStream":
		return true, handleConfWriteXRefStream(k, v, c)
	}

	return false, nil
}

func parseKeysPart2(k, v string, c *Configuration) (bool, error) {
	switch k {

	case "encryptUsingAES":
		return true, handleConfEncryptUsingAES(k, v, c)

	case "encryptKeyLength":
		return true, handleConfEncryptKeyLength(v, c)

	case "permissions":
		return true, handleConfPermissions(v, c)

	case "unit", "units":
		return true, handleConfUnit(v, c)

	case "timestampFormat":
		return true, handleTimestampFormat(v, c)

	case "dateFormat":
		return true, handleDateFormat(v, c)

	case "timeout":
		return true, handleTimeout(v, c)

	case "timeoutCRL":
		return true, handleTimeoutCRL(v, c)

	case "timeoutOCSP":
		return true, handleTimeoutOCSP(v, c)

	case "allowedRevocationHosts":
		return true, handleAllowedRevocationHosts(v, c)

	case "formFieldListMaxColWidth":
		return true, handleFormFieldListMaxColWidth(v, c)

	case "preferredCertRevocationChecker":
		return true, handlePreferredCertRevocationChecker(v, c)
	}

	return false, nil
}

func parseKeysPart3(k, v string, c *Configuration) (bool, error) {
	switch k {

	case "maxStreamBytes":
		return true, handleLimitInt64(k, v, &c.Limits.MaxStreamBytes)

	case "maxDecodeBytes":
		return true, handleLimitInt64(k, v, &c.Limits.MaxDecodeBytes)

	case "maxImagePixels":
		return true, handleLimitInt64(k, v, &c.Limits.MaxImagePixels)

	case "maxImageBytes":
		return true, handleLimitInt64(k, v, &c.Limits.MaxImageBytes)

	case "maxObjectCount":
		return true, handleLimitInt(k, v, &c.Limits.MaxObjectCount)

	case "maxObjectStreamCount":
		return true, handleLimitInt(k, v, &c.Limits.MaxObjectStreamCount)

	case "maxObjectStreamFirst":
		return true, handleLimitInt64(k, v, &c.Limits.MaxObjectStreamFirst)

	case "maxXRefEntries":
		return true, handleLimitInt(k, v, &c.Limits.MaxXRefEntries)
	}

	return false, nil
}

func parseKeysPart4(k, v string, c *Configuration) (err error) {
	switch k {

	case "optimize":
		c.Optimize, err = boolean(k, v)

	case "optimizeResourceDicts":
		c.OptimizeResourceDicts, err = boolean(k, v)

	case "optimizeDuplicateContentStreams":
		c.OptimizeDuplicateContentStreams, err = boolean(k, v)

	case "createBookmarks":
		c.CreateBookmarks, err = boolean(k, v)

	case "needAppearances":
		c.NeedAppearances, err = boolean(k, v)

	case "offline":
		c.Offline, err = boolean(k, v)

	}

	return err
}

func parseKeyValue(k, v string, c *Configuration) error {
	ok, err := parseKeysPart1(k, v, c)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}

	ok, err = parseKeysPart2(k, v, c)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}

	ok, err = parseKeysPart3(k, v, c)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}

	return parseKeysPart4(k, v, c)
}

func parseConfigFile(r io.Reader, configPath string) error {
	//fmt.Println("parseConfigFile For JS")
	conf := *newDefaultConfiguration()
	conf.Path = configPath

	s := bufio.NewScanner(r)
	for s.Scan() {
		t := s.Text()
		if len(t) == 0 || t[0] == '#' {
			continue
		}
		if i := strings.Index(t, "#"); i >= 0 {
			t = strings.TrimSpace(t[:i])
			if t == "" {
				continue
			}
		}
		if strings.HasSuffix(t, ":") {
			continue
		}
		ss := strings.Split(t, ": ")
		if len(ss) != 2 {
			return fmt.Errorf("invalid entry: <%s>", t)
		}
		k := strings.TrimSpace(ss[0])
		v := strings.TrimSpace(ss[1])
		if len(k) == 0 || len(v) == 0 {
			return fmt.Errorf("invalid entry: <%s>", t)
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

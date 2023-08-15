/*
Copyright 2018 The pdfcpu Authors.

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
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

const (
	// ValidationStrict ensures 100% compliance with the spec (PDF 32000-1:2008).
	ValidationStrict int = iota

	// ValidationRelaxed ensures PDF compliance based on frequently encountered validation errors.
	ValidationRelaxed

	// ValidationNone bypasses validation.
	ValidationNone
)

const (

	// StatsFileNameDefault is the standard stats filename.
	StatsFileNameDefault = "stats.csv"

	// PermissionsAll enables all user access permission bits.
	PermissionsAll int16 = -1 // 0xFFFF

	// PermissionsPrint disables all user access permissions bits except for printing.
	PermissionsPrint int16 = -1849 // 0xF8C7

	// PermissionsNone disables all user access permissions bits.
	PermissionsNone int16 = -3901 // 0xF0C3

)

// CommandMode specifies the operation being executed.
type CommandMode int

// The available commands.
const (
	VALIDATE CommandMode = iota
	LISTINFO
	OPTIMIZE
	SPLIT
	MERGECREATE
	MERGEAPPEND
	EXTRACTIMAGES
	EXTRACTFONTS
	EXTRACTPAGES
	EXTRACTCONTENT
	EXTRACTMETADATA
	TRIM
	LISTATTACHMENTS
	EXTRACTATTACHMENTS
	ADDATTACHMENTS
	ADDATTACHMENTSPORTFOLIO
	REMOVEATTACHMENTS
	LISTPERMISSIONS
	SETPERMISSIONS
	ADDWATERMARKS
	REMOVEWATERMARKS
	IMPORTIMAGES
	INSERTPAGESBEFORE
	INSERTPAGESAFTER
	REMOVEPAGES
	LISTKEYWORDS
	ADDKEYWORDS
	REMOVEKEYWORDS
	LISTPROPERTIES
	ADDPROPERTIES
	REMOVEPROPERTIES
	COLLECT
	CROP
	LISTBOXES
	ADDBOXES
	REMOVEBOXES
	LISTANNOTATIONS
	ADDANNOTATIONS
	REMOVEANNOTATIONS
	ROTATE
	NUP
	BOOKLET
	LISTBOOKMARKS
	ADDBOOKMARKS
	REMOVEBOOKMARKS
	IMPORTBOOKMARKS
	EXPORTBOOKMARKS
	LISTIMAGES
	CREATE
	DUMP
	LISTFORMFIELDS
	REMOVEFORMFIELDS
	LOCKFORMFIELDS
	UNLOCKFORMFIELDS
	RESETFORMFIELDS
	EXPORTFORMFIELDS
	FILLFORMFIELDS
	MULTIFILLFORMFIELDS
	ENCRYPT
	DECRYPT
	CHANGEUPW
	CHANGEOPW
	CHEATSHEETSFONTS
	INSTALLFONTS
	LISTFONTS
	RESIZE
	POSTER
	NDOWN
	CUT
)

// Configuration of a Context.
type Configuration struct {
	// Location of corresponding config.yml
	Path string

	// Check filename extensions.
	CheckFileNameExt bool

	// Enables PDF V1.5 compatible processing of object streams, xref streams, hybrid PDF files.
	Reader15 bool

	// Enables decoding of all streams (fontfiles, images..) for logging purposes.
	DecodeAllStreams bool

	// Validate against ISO-32000: strict or relaxed.
	ValidationMode int

	// Check for broken links in LinkedAnnotations/URIActions.
	ValidateLinks bool

	// End of line char sequence for writing.
	Eol string

	// Turns on object stream generation.
	// A signal for compressing any new non-stream-object into an object stream.
	// true enforces WriteXRefStream to true.
	// false does not prevent xRefStream generation.
	WriteObjectStream bool

	// Switches between xRefSection (<=V1.4) and objectStream/xRefStream (>=V1.5) writing.
	WriteXRefStream bool

	// Turns on stats collection.
	// TODO Decision - unused.
	CollectStats bool

	// A CSV-filename holding the statistics.
	StatsFileName string

	// Supplied user password.
	UserPW    string
	UserPWNew *string

	// Supplied owner password.
	OwnerPW    string
	OwnerPWNew *string

	// EncryptUsingAES ensures AES encryption.
	// true: AES encryption
	// false: RC4 encryption.
	EncryptUsingAES bool

	// AES:40,128,256 RC4:40,128
	EncryptKeyLength int

	// Supplied user access permissions, see Table 22.
	Permissions int16

	// Command being executed.
	Cmd CommandMode

	// Display unit in effect.
	Unit types.DisplayUnit

	// Timestamp format.
	TimestampFormat string

	// Date format.
	DateFormat string

	// Buffersize for locating PDF header <= 100
	HeaderBufSize int

	// Optimize duplicate content streams across pages.
	OptimizeDuplicateContentStreams bool

	// Merge creates bookmarks
	CreateBookmarks bool
}

// ConfigPath defines the location of pdfcpu's configuration directory.
// If set to a file path, pdfcpu will ensure the config dir at this location.
// Other possible values:
//
//	default:	Ensure config dir at default location
//	disable:	Disable config dir usage
//
// If you want to disable config dir usage in a multi threaded environment
// you are encouraged to use api.DisableConfigDir().
var ConfigPath string = "default"

var loadedDefaultConfig *Configuration

//go:embed config.yml
var configFileBytes []byte

func ensureConfigFileAt(path string) error {
	f, err := os.Open(path)
	if err != nil {
		f.Close()
		s := fmt.Sprintf("#############################\n# pdfcpu %s         #\n# Created: %s #\n", VersionStr, time.Now().Format("2006-01-02 15:04"))
		bb := append([]byte(s), configFileBytes...)
		if err := os.WriteFile(path, bb, os.ModePerm); err != nil {
			return err
		}
		f, err = os.Open(path)
		if err != nil {
			return err
		}
	}
	defer f.Close()
	// Load configuration into loadedDefaultConfig.
	return parseConfigFile(f, path)
}

// EnsureDefaultConfigAt tries to load the default configuration from path.
// If path/pdfcpu/config.yaml is not found, it will be created.
func EnsureDefaultConfigAt(path string) error {
	configDir := filepath.Join(path, "pdfcpu")
	font.UserFontDir = filepath.Join(configDir, "fonts")
	if err := os.MkdirAll(font.UserFontDir, os.ModePerm); err != nil {
		return err
	}
	if err := ensureConfigFileAt(filepath.Join(configDir, "config.yml")); err != nil {
		return err
	}
	//fmt.Println(loadedDefaultConfig)
	return font.LoadUserFonts()
}

func newDefaultConfiguration() *Configuration {
	// NOTE: Needs to stay in sync with config.yml
	//
	// Takes effect whenever the installed config.yml is disabled:
	// 		cli: supply -conf disable
	// 		api: call api.DisableConfigDir()
	return &Configuration{
		CheckFileNameExt:                true,
		Reader15:                        true,
		DecodeAllStreams:                false,
		ValidationMode:                  ValidationRelaxed,
		ValidateLinks:                   false,
		Eol:                             types.EolLF,
		WriteObjectStream:               true,
		WriteXRefStream:                 true,
		EncryptUsingAES:                 true,
		EncryptKeyLength:                256,
		Permissions:                     PermissionsNone,
		TimestampFormat:                 "2006-01-02 15:04",
		DateFormat:                      "2006-01-02",
		HeaderBufSize:                   100,
		OptimizeDuplicateContentStreams: false,
		CreateBookmarks:                 true,
	}
}

// NewDefaultConfiguration returns the default pdfcpu configuration.
func NewDefaultConfiguration() *Configuration {
	if loadedDefaultConfig != nil {
		c := *loadedDefaultConfig
		return &c
	}
	if ConfigPath != "disable" {
		path, err := os.UserConfigDir()
		if err != nil {
			path = os.TempDir()
		}
		if err = EnsureDefaultConfigAt(path); err == nil {
			c := *loadedDefaultConfig
			return &c
		}
		fmt.Fprintf(os.Stderr, "pdfcpu: config dir problem: %v\n", err)
		os.Exit(1)
	}
	// Bypass config.yml
	return newDefaultConfiguration()
}

// NewAESConfiguration returns a default configuration for AES encryption.
func NewAESConfiguration(userPW, ownerPW string, keyLength int) *Configuration {
	c := NewDefaultConfiguration()
	c.UserPW = userPW
	c.OwnerPW = ownerPW
	c.EncryptUsingAES = true
	c.EncryptKeyLength = keyLength
	return c
}

// NewRC4Configuration returns a default configuration for RC4 encryption.
func NewRC4Configuration(userPW, ownerPW string, keyLength int) *Configuration {
	c := NewDefaultConfiguration()
	c.UserPW = userPW
	c.OwnerPW = ownerPW
	c.EncryptUsingAES = false
	c.EncryptKeyLength = keyLength
	return c
}

func (c Configuration) String() string {
	path := "default"
	if len(c.Path) > 0 {
		path = c.Path
	}
	return fmt.Sprintf("pdfcpu configuration:\n"+
		"Path:              %s\n"+
		"CheckFileNameExt:  %t\n"+
		"Reader15:          %t\n"+
		"DecodeAllStreams:  %t\n"+
		"ValidationMode:    %s\n"+
		"Eol:               %s\n"+
		"WriteObjectStream: %t\n"+
		"WriteXrefStream:   %t\n"+
		"EncryptUsingAES:   %t\n"+
		"EncryptKeyLength:  %d\n"+
		"Permissions:       %d\n"+
		"Unit :             %s\n"+
		"TimestampFormat:	%s\n"+
		"DateFormat:		%s\n"+
		"HeaderBufSize:		%d\n"+
		"OptimizeDuplicateContentStreams %t\n"+
		"CreateBookmarks %t\n",
		path,
		c.CheckFileNameExt,
		c.Reader15,
		c.DecodeAllStreams,
		c.ValidationModeString(),
		c.EolString(),
		c.WriteObjectStream,
		c.WriteXRefStream,
		c.EncryptUsingAES,
		c.EncryptKeyLength,
		c.Permissions,
		c.UnitString(),
		c.TimestampFormat,
		c.DateFormat,
		c.HeaderBufSize,
		c.OptimizeDuplicateContentStreams,
		c.CreateBookmarks,
	)
}

// EolString returns a string rep for the eol in effect.
func (c *Configuration) EolString() string {
	var s string
	switch c.Eol {
	case types.EolLF:
		s = "EolLF"
	case types.EolCR:
		s = "EolCR"
	case types.EolCRLF:
		s = "EolCRLF"
	}
	return s
}

// ValidationModeString returns a string rep for the validation mode in effect.
func (c *Configuration) ValidationModeString() string {
	if c.ValidationMode == ValidationStrict {
		return "strict"
	}
	if c.ValidationMode == ValidationRelaxed {
		return "relaxed"
	}
	return "none"
}

// UnitString returns a string rep for the display unit in effect.
func (c *Configuration) UnitString() string {
	var s string
	switch c.Unit {
	case types.POINTS:
		s = "points"
	case types.INCHES:
		s = "inches"
	case types.CENTIMETRES:
		s = "cm"
	case types.MILLIMETRES:
		s = "mm"
	}
	return s
}

// ApplyReducedFeatureSet returns true if complex entries like annotations shall not be written.
func (c *Configuration) ApplyReducedFeatureSet() bool {
	switch c.Cmd {
	case SPLIT, TRIM, EXTRACTPAGES, MERGECREATE, MERGEAPPEND, IMPORTIMAGES:
		return true
	}
	return false
}

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
	"embed"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

const (
	// ValidationStrict ensures 100% compliance with the spec (PDF 32000-1:2008).
	ValidationStrict int = iota

	// ValidationRelaxed ensures PDF compliance based on frequently encountered validation errors.
	ValidationRelaxed
)

// See table 22 - User access permissions
type PermissionFlags int

const (
	UnusedFlag1              PermissionFlags = 1 << iota // Bit 1:  unused
	UnusedFlag2                                          // Bit 2:  unused
	PermissionPrintRev2                                  // Bit 3:  Print (security handlers rev.2), draft print (security handlers >= rev.3)
	PermissionModify                                     // Bit 4:  Modify contents by operations other than controlled by bits 6, 9, 11.
	PermissionExtract                                    // Bit 5:  Copy, extract text & graphics
	PermissionModAnnFillForm                             // Bit 6:  Add or modify annotations, fill form fields, in conjunction with bit 4 create/mod form fields.
	UnusedFlag7                                          // Bit 7:  unused
	UnusedFlag8                                          // Bit 8:  unused
	PermissionFillRev3                                   // Bit 9:  Fill form fields (security handlers >= rev.3)
	PermissionExtractRev3                                // Bit 10: Copy, extract text & graphics (security handlers >= rev.3) (unused since PDF 2.0)
	PermissionAssembleRev3                               // Bit 11: Assemble document (security handlers >= rev.3)
	PermissionPrintRev3                                  // Bit 12: Print (security handlers >= rev.3)
)

const (
	PermissionsNone  = PermissionFlags(0xF0C3)
	PermissionsPrint = PermissionsNone + PermissionPrintRev2 + PermissionPrintRev3
	PermissionsAll   = PermissionFlags(0xFFFF)
)

const (

	// StatsFileNameDefault is the standard stats filename.
	StatsFileNameDefault = "stats.csv"
)

// CommandMode specifies the operation being executed.
type CommandMode int

// The available commands.
const (
	VALIDATE CommandMode = iota
	LISTINFO
	OPTIMIZE
	SPLIT
	SPLITBYPAGENR
	MERGECREATE
	MERGECREATEZIP
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
	UPDATEIMAGES
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
	LISTPAGELAYOUT
	SETPAGELAYOUT
	RESETPAGELAYOUT
	LISTPAGEMODE
	SETPAGEMODE
	RESETPAGEMODE
	LISTVIEWERPREFERENCES
	SETVIEWERPREFERENCES
	RESETVIEWERPREFERENCES
	ZOOM
	ADDSIGNATURE
	VALIDATESIGNATURE
	LISTCERTIFICATES
	INSPECTCERTIFICATES
	IMPORTCERTIFICATES
	VALIDATESIGNATURES
)

// Configuration of a Context.
type Configuration struct {
	// Location of corresponding config.yml
	Path string

	CreationDate string

	Version string

	// Check filename extensions.
	CheckFileNameExt bool

	// Enables PDF V1.5 compatible processing of object streams, xref streams, hybrid PDF files.
	Reader15 bool

	// Enables decoding of all streams (fontfiles, images..) for logging purposes.
	DecodeAllStreams bool

	// Validate against ISO-32000: strict or relaxed.
	ValidationMode int

	// Enable validation right before writing.
	PostProcessValidate bool

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
	Permissions PermissionFlags // int16

	// Command being executed.
	Cmd CommandMode

	// Display unit in effect.
	Unit types.DisplayUnit

	// Timestamp format.
	TimestampFormat string

	// Date format.
	DateFormat string

	// Optimize after reading and validating the xreftable but before processing.
	Optimize bool

	// Optimize after processing but before writing.
	// TODO add to config.yml
	OptimizeBeforeWriting bool

	// Optimize page resources via content stream analysis. (assuming Optimize == true || OptimizeBeforeWriting == true)
	OptimizeResourceDicts bool

	// Optimize duplicate content streams across pages. (assuming Optimize == true || OptimizeBeforeWriting == true)
	OptimizeDuplicateContentStreams bool

	// Merge creates bookmarks.
	CreateBookmarks bool

	// PDF Viewer is expected to supply appearance streams for form fields.
	NeedAppearances bool

	// Internet availability.
	Offline bool

	// HTTP timeout in seconds.
	Timeout int

	// Http timeout in seconds for CRL revocation checking.
	TimeoutCRL int

	// Http timeout in seconds for OCSP revocation checking.
	TimeoutOCSP int

	// Preferred certificate revocation checking mechanism: CRL, OSCP
	PreferredCertRevocationChecker int
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

//go:embed resources/config.yml
var configFileBytes []byte

//go:embed resources/Roboto-Regular.ttf
var robotoFontFileBytes []byte

//go:embed resources/certs/*.p7c
var certFilesEU embed.FS

func ensureConfigFileAt(path string, override bool) error {
	f, err := os.Open(path)
	if err != nil || override {
		f.Close()

		s := fmt.Sprintf(`
#############################
#   Default configuration   #
#############################

# Creation date
created: %s 

# version (Do not edit!)
version: %s 

`,
			time.Now().Format("2006-01-02 15:04"),
			VersionStr)

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

func onlyHidden(files []os.DirEntry) bool {
	for _, file := range files {
		if !strings.HasPrefix(file.Name(), ".") {
			return false
		}
	}
	return true
}

func initUserFonts() error {
	files, err := os.ReadDir(font.UserFontDir)
	if err != nil {
		return err
	}

	if onlyHidden(files) {
		// Ensure Roboto font for form filling.
		fontname := "Roboto-Regular"
		if log.CLIEnabled() {
			log.CLI.Printf("installing user font:")
		}
		if err := font.InstallFontFromBytes(font.UserFontDir, fontname, robotoFontFileBytes); err != nil {
			return err
		}
	}

	return font.LoadUserFonts()
}

// ensureFontDirInitialized sets up the font directory without loading fonts.
// Font loading is deferred until fonts are actually needed.
func ensureFontDirInitialized() error {
	files, err := os.ReadDir(font.UserFontDir)
	if err != nil {
		return err
	}

	if onlyHidden(files) {
		// Ensure Roboto font for form filling.
		fontname := "Roboto-Regular"
		if log.CLIEnabled() {
			log.CLI.Printf("installing user font:")
		}
		if err := font.InstallFontFromBytes(font.UserFontDir, fontname, robotoFontFileBytes); err != nil {
			return err
		}
	}

	// Don't load fonts here - they will be loaded lazily when first accessed
	return nil
}

func initCertificates() error {
	// NOTE
	// Load certs managed by The European Union Trusted Lists (EUTL) (https://eidas.ec.europa.eu/efda/trust-services/browse/eidas/tls).
	// Additional certificates may be loaded using the corresponding CLI command: pdfcpu certificates import
	// Certificates will be loaded by corresponding commands where applicable.

	files, err := os.ReadDir(CertDir)
	if err != nil {
		return err
	}
	if !onlyHidden(files) {
		return nil
	}

	files, err = certFilesEU.ReadDir("resources/certs")
	if err != nil {
		return err
	}

	euDir := filepath.Join(CertDir, "eu")
	if err := os.MkdirAll(euDir, os.ModePerm); err != nil {
		return err
	}

	for _, file := range files {
		//fmt.Println("Embedded file:", file.Name())

		content, err := certFilesEU.ReadFile("resources/certs/" + file.Name())
		if err != nil {
			return err
		}

		path := filepath.Join(euDir, file.Name())
		//fmt.Printf("writing to %s\n", path)

		destFile, err := os.Create(path)
		if err != nil {
			return err
		}
		defer destFile.Close()

		_, err = destFile.Write(content)
		if err != nil {
			return err
		}
	}

	return nil
}

// EnsureDefaultConfigAt tries to load the default configuration from path.
// If path/pdfcpu/config.yaml is not found, it will be created.
func EnsureDefaultConfigAt(path string, override bool) error {
	configDir := filepath.Join(path, "pdfcpu")
	if err := os.MkdirAll(configDir, os.ModePerm); err != nil {
		return err
	}
	if err := ensureConfigFileAt(filepath.Join(configDir, "config.yml"), override); err != nil {
		return err
	}

	font.UserFontDir = filepath.Join(configDir, "fonts")
	if err := os.MkdirAll(font.UserFontDir, os.ModePerm); err != nil {
		return err
	}
	// Initialize font directory and install Roboto if needed, but defer actual loading
	if err := ensureFontDirInitialized(); err != nil {
		return err
	}

	CertDir = filepath.Join(configDir, "certs")
	if err := os.MkdirAll(CertDir, os.ModePerm); err != nil {
		return err
	}
	if err := initCertificates(); err != nil {
		return err
	}

	//fmt.Println(loadedDefaultConfig)

	return nil
}

func newDefaultConfiguration() *Configuration {
	// NOTE: Needs to stay in sync with config.yml
	//
	// Takes effect whenever the installed config.yml is disabled:
	// 		cli: supply -conf disable
	// 		api: call api.DisableConfigDir()
	return &Configuration{
		CreationDate:                    time.Now().Format("2006-01-02 15:04"),
		Version:                         VersionStr,
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
		Permissions:                     PermissionsPrint,
		TimestampFormat:                 "2006-01-02 15:04",
		DateFormat:                      "2006-01-02",
		Optimize:                        true,
		OptimizeBeforeWriting:           true,
		OptimizeResourceDicts:           true,
		OptimizeDuplicateContentStreams: false,
		CreateBookmarks:                 true,
		NeedAppearances:                 false,
		Offline:                         false,
		Timeout:                         5,
		PreferredCertRevocationChecker:  CRL,
	}
}

func ResetConfig() error {
	path, err := os.UserConfigDir()
	if err != nil {
		path = os.TempDir()
	}
	return EnsureDefaultConfigAt(path, true)
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
		if err = EnsureDefaultConfigAt(path, false); err == nil {
			c := *loadedDefaultConfig
			return &c
		}
		fmt.Fprintf(os.Stderr, "pdfcpu: config problem: %v\n", err)
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
	return "relaxed"
}

// PreferredCertRevocationCheckerString returns a string rep for the preferred certificate revocation checker in effect.
func (c *Configuration) PreferredCertRevocationCheckerString() string {
	if c.PreferredCertRevocationChecker == CRL {
		return "CRL"
	}
	return "OSCP"
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

// SetUnit configures the display unit.
func (c *Configuration) SetUnit(s string) {
	switch s {
	case "points":
		c.Unit = types.POINTS
	case "inches":
		c.Unit = types.INCHES
	case "cm":
		c.Unit = types.CENTIMETRES
	case "mm":
		c.Unit = types.MILLIMETRES
	}
}

// ApplyReducedFeatureSet returns true if complex entries like annotations shall not be written.
func (c *Configuration) ApplyReducedFeatureSet() bool {
	switch c.Cmd {
	case SPLIT, TRIM, EXTRACTPAGES, IMPORTIMAGES:
		return true
	}
	return false
}
